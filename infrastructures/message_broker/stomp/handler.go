package stomp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/commons/utils"
	"go-serviceboilerplate/infrastructures/configurations"
	"net"
	"sync"
	"time"

	goStomp "github.com/go-stomp/stomp/v3"
	"github.com/go-stomp/stomp/v3/frame"
)

// StompInstance wraps a single STOMP broker connection. Create one per connection the service needs, e.g.
//
//	mainStomp := stomp.NewStompInstance(configs, configs.Envs.MainStomp)
//	eventsStomp := stomp.NewStompInstance(configs, configs.Envs.EventsStomp)
//
// The connection is re-established automatically when it drops, and active subscriptions are resubscribed.
type StompInstance struct {
	Name string

	configs     *configurations.Configs
	stompConfig models.StompConfig

	connMu sync.Mutex
	conn   *goStomp.Conn

	// ctx is cancelled on Close to stop consumers and reconnect loops
	ctx       context.Context
	cancel    context.CancelFunc
	consumers sync.WaitGroup
	closeOnce sync.Once
}

type subscription struct {
	destination string
	handler     models.BrokerMessageHandler
	headers     map[string]string
}

func NewStompInstance(configs *configurations.Configs, stompConfig models.StompConfig) *StompInstance {
	conn, err := dial(configs, stompConfig)
	if err != nil {
		configs.Logger.Fatal("STOMP connection failed", "name", stompConfig.Name, "address", stompConfig.Address, "error", err)
	}

	configs.Logger.Info("STOMP Successfully Connected", "name", stompConfig.Name, "address", stompConfig.Address, "server", conn.Server())

	ctx, cancel := context.WithCancel(context.Background())

	return &StompInstance{
		Name:        stompConfig.Name,
		configs:     configs,
		stompConfig: stompConfig,
		conn:        conn,
		ctx:         ctx,
		cancel:      cancel,
	}
}

func dial(configs *configurations.Configs, stompConfig models.StompConfig) (*goStomp.Conn, error) {
	host, _, err := net.SplitHostPort(stompConfig.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid address %q: %w", stompConfig.Address, err)
	}

	dialer := &net.Dialer{Timeout: stompConfig.ConnectTimeout}

	var netConn net.Conn
	if stompConfig.UseTLS {
		netConn, err = tls.DialWithDialer(dialer, "tcp", stompConfig.Address, &tls.Config{ServerName: host})
	} else {
		netConn, err = dialer.Dial("tcp", stompConfig.Address)
	}

	if err != nil {
		return nil, err
	}

	virtualHost := stompConfig.Host
	if virtualHost == "" {
		virtualHost = host
	}

	opts := []func(*goStomp.Conn) error{
		goStomp.ConnOpt.Host(virtualHost),
		goStomp.ConnOpt.Logger(&stompLogger{logger: configs.Logger, name: stompConfig.Name}),
	}
	if stompConfig.Login != "" {
		opts = append(opts, goStomp.ConnOpt.Login(stompConfig.Login, stompConfig.Passcode))
	}
	if stompConfig.HeartBeat > 0 {
		opts = append(opts, goStomp.ConnOpt.HeartBeat(stompConfig.HeartBeat, stompConfig.HeartBeat))
	}

	ctx := context.Background()
	if stompConfig.ConnectTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, stompConfig.ConnectTimeout)
		defer cancel()
	}

	conn, err := goStomp.ConnectWithContext(ctx, netConn, opts...)
	if err != nil {
		netConn.Close()
		return nil, err
	}

	return conn, nil
}

func (s *StompInstance) currentConn() *goStomp.Conn {
	s.connMu.Lock()
	defer s.connMu.Unlock()

	return s.conn
}

// reconnect replaces the failed connection. If another goroutine already replaced it, the current connection is returned.
func (s *StompInstance) reconnect(ctx context.Context, failed *goStomp.Conn) (*goStomp.Conn, error) {
	s.connMu.Lock()
	defer s.connMu.Unlock()

	if s.conn != failed {
		return s.conn, nil
	}

	if s.stompConfig.ReconnectInterval <= 0 {
		return nil, fmt.Errorf("stomp %s: connection lost and auto-reconnect is disabled", s.Name)
	}

	// Release the old connection's resources. It is already broken, so the error is irrelevant
	_ = failed.MustDisconnect()

	for attempt := 1; ; attempt++ {
		if s.ctx.Err() != nil {
			return nil, models.ErrMessageBrokerClosed
		}

		conn, err := dial(s.configs, s.stompConfig)
		if err == nil {
			s.conn = conn
			s.configs.Logger.Info("STOMP Reconnected", "name", s.Name, "attempt", attempt)
			return conn, nil
		}

		s.configs.Logger.Warn("STOMP reconnect failed", "name", s.Name, "attempt", attempt, "error", err)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-s.ctx.Done():
			return nil, models.ErrMessageBrokerClosed
		case <-time.After(s.stompConfig.ReconnectInterval):
		}
	}
}

// Publish sends payload to a destination (e.g. "/queue/orders", "/topic/events").
// []byte is sent as-is, string as text/plain, anything else as JSON. headers is optional.
// If the connection has dropped, Publish reconnects and retries once.
func (s *StompInstance) Publish(ctx context.Context, destination string, payload any, headers map[string]string) error {
	if s.ctx.Err() != nil {
		return models.ErrMessageBrokerClosed
	}

	body, contentType, err := utils.EncodeMessagePayload(payload)
	if err != nil {
		return fmt.Errorf("stomp %s publish %q: encode: %w", s.Name, destination, err)
	}

	opts := make([]func(*frame.Frame) error, 0, len(headers)+1)
	for key, value := range headers {
		opts = append(opts, goStomp.SendOpt.Header(key, value))
	}

	// Propagate the caller's request ID so the consumer's logs can be traced back to the originating request
	if _, exists := headers[models.XRequestIDHeader]; !exists {
		if requestID := utils.RequestIDFromContext(ctx); requestID != "" {
			opts = append(opts, goStomp.SendOpt.Header(models.XRequestIDHeader, requestID))
		}
	}

	conn := s.currentConn()
	err = conn.Send(destination, contentType, body, opts...)

	if isConnectionError(err) {
		conn, err = s.reconnect(ctx, conn)
		if err == nil {
			err = conn.Send(destination, contentType, body, opts...)
		}
	}

	if err != nil {
		return fmt.Errorf("stomp %s publish %q: %w", s.Name, destination, err)
	}

	return nil
}

// Subscribe starts consuming a destination in the background. Each message is passed to handler one at a time:
// returning nil acknowledges it, returning an error (or panicking) sends a NACK so the broker can redeliver it.
// headers is optional and passed on SUBSCRIBE (e.g. RabbitMQ "prefetch-count", ActiveMQ "activemq.prefetchSize").
func (s *StompInstance) Subscribe(destination string, handler models.BrokerMessageHandler, headers map[string]string) error {
	if s.ctx.Err() != nil {
		return models.ErrMessageBrokerClosed
	}

	sub := subscription{destination: destination, handler: handler, headers: headers}

	conn := s.currentConn()
	stompSub, err := subscribe(conn, sub)
	if err != nil {
		return fmt.Errorf("stomp %s subscribe %q: %w", s.Name, destination, err)
	}

	s.configs.Logger.Info("STOMP Subscribed", "name", s.Name, "destination", destination)

	s.consumers.Add(1)
	go s.consume(conn, stompSub, sub)

	return nil
}

func subscribe(conn *goStomp.Conn, sub subscription) (*goStomp.Subscription, error) {
	opts := make([]func(*frame.Frame) error, 0, len(sub.headers))
	for key, value := range sub.headers {
		opts = append(opts, goStomp.SubscribeOpt.Header(key, value))
	}

	return conn.Subscribe(sub.destination, goStomp.AckClientIndividual, opts...)
}

// consume processes messages until Close is called, resubscribing whenever the connection drops
func (s *StompInstance) consume(conn *goStomp.Conn, stompSub *goStomp.Subscription, sub subscription) {
	defer s.consumers.Done()

	for {
		err := s.receive(stompSub, sub.handler)
		if err == nil {
			return
		}

		s.configs.Logger.Warn("STOMP subscription interrupted", "name", s.Name, "destination", sub.destination, "error", err)

		// The subscription channel only closes when the connection drops or the broker sends an ERROR frame
		// (after which it closes the connection), so the connection must be replaced before resubscribing
		for {
			conn, err = s.reconnect(s.ctx, conn)
			if err != nil {
				if !errors.Is(err, models.ErrMessageBrokerClosed) {
					s.configs.Logger.Error("STOMP subscription stopped", "name", s.Name, "destination", sub.destination, "error", err)
				}
				return
			}

			stompSub, err = subscribe(conn, sub)
			if err == nil {
				s.configs.Logger.Info("STOMP Resubscribed", "name", s.Name, "destination", sub.destination)
				break
			}

			if !isConnectionError(err) {
				s.configs.Logger.Error("STOMP subscription stopped", "name", s.Name, "destination", sub.destination, "error", err)
				return
			}
		}
	}
}

// receive returns nil when the instance is closing, or the error that interrupted the subscription
func (s *StompInstance) receive(stompSub *goStomp.Subscription, handler models.BrokerMessageHandler) error {
	for {
		select {
		case <-s.ctx.Done():
			return nil
		case msg, ok := <-stompSub.C:
			if !ok {
				return goStomp.ErrCompletedSubscription
			}
			if msg.Err != nil {
				return msg.Err
			}

			s.handleMessage(msg, handler)
		}
	}
}

func (s *StompInstance) handleMessage(msg *goStomp.Message, handler models.BrokerMessageHandler) {
	brokerMessage := toBrokerMessage(msg)

	// Each message gets its own ctx. It is not cancelled by Close, so in-flight handlers can finish their work
	requestID := brokerMessage.Headers[models.XRequestIDHeader]
	if requestID == "" {
		requestID = brokerMessage.Headers[frame.MessageId]
	}
	ctx := utils.WithRequestID(context.Background(), requestID)

	if err := runHandler(ctx, handler, brokerMessage); err != nil {
		s.configs.Logger.WarnContext(ctx, "STOMP message handler failed, message rejected", "name", s.Name, "destination", msg.Destination, "error", err)

		if nackErr := msg.Conn.Nack(msg); nackErr != nil {
			s.configs.Logger.ErrorContext(ctx, "STOMP nack failed", "name", s.Name, "destination", msg.Destination, "error", nackErr)
		}
		return
	}

	if ackErr := msg.Conn.Ack(msg); ackErr != nil {
		s.configs.Logger.ErrorContext(ctx, "STOMP ack failed", "name", s.Name, "destination", msg.Destination, "error", ackErr)
	}
}

func runHandler(ctx context.Context, handler models.BrokerMessageHandler, msg models.BrokerMessage) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("handler panic: %v", r)
		}
	}()

	return handler(ctx, msg)
}

func toBrokerMessage(msg *goStomp.Message) models.BrokerMessage {
	headers := make(map[string]string, msg.Header.Len())
	for i := 0; i < msg.Header.Len(); i++ {
		key, value := msg.Header.GetAt(i)
		// STOMP: when a header is repeated, the first occurrence wins
		if _, exists := headers[key]; !exists {
			headers[key] = value
		}
	}

	return models.BrokerMessage{
		Destination: msg.Destination,
		ContentType: msg.ContentType,
		Headers:     headers,
		Body:        msg.Body,
	}
}

// stompLogger routes go-stomp's internal logs through the service's slog logger
type stompLogger struct {
	logger *configurations.SlogLogger
	name   string
}

func (l *stompLogger) Debugf(format string, value ...any)   { l.Debug(fmt.Sprintf(format, value...)) }
func (l *stompLogger) Infof(format string, value ...any)    { l.Info(fmt.Sprintf(format, value...)) }
func (l *stompLogger) Warningf(format string, value ...any) { l.Warning(fmt.Sprintf(format, value...)) }
func (l *stompLogger) Errorf(format string, value ...any)   { l.Error(fmt.Sprintf(format, value...)) }
func (l *stompLogger) Debug(message string)                 { l.logger.Debug("STOMP client: "+message, "name", l.name) }
func (l *stompLogger) Info(message string)                  { l.logger.Info("STOMP client: "+message, "name", l.name) }
func (l *stompLogger) Warning(message string) {
	l.logger.Warn("STOMP client: "+message, "name", l.name)
}
func (l *stompLogger) Error(message string) { l.logger.Error("STOMP client: "+message, "name", l.name) }

func isConnectionError(err error) bool {
	return errors.Is(err, goStomp.ErrClosedUnexpectedly) || errors.Is(err, goStomp.ErrAlreadyClosed)
}

// Close stops all subscriptions, waits for in-flight handlers to finish, then disconnects.
// Unacknowledged messages are redelivered by the broker.
func (s *StompInstance) Close() error {
	var err error

	s.closeOnce.Do(func() {
		s.cancel()
		s.consumers.Wait()

		err = s.currentConn().Disconnect()
		s.configs.Logger.Info("STOMP Disconnected", "name", s.Name)
	})

	return err
}
