package configurations

import (
	"context"
	"go-serviceboilerplate/commons/utils"
	"log/slog"
	"os"
)

type SlogLogger struct {
    handler *slog.Logger
}

func NewSlogLogger() *SlogLogger {
    handler := slog.New(contextHandler{slog.NewJSONHandler(os.Stdout, nil)})
    return &SlogLogger{handler: handler}
}

// Use the *Context variants whenever a ctx is available, so the log line carries the request_id
func (l *SlogLogger) Info(msg string, args ...any)  { l.handler.Info(msg, args...) }
func (l *SlogLogger) Error(msg string, args ...any) { l.handler.Error(msg, args...) }
func (l *SlogLogger) Fatal(msg string, args ...any) { l.handler.Error(msg, args...); os.Exit(1) }
func (l *SlogLogger) Warn(msg string, args ...any)  { l.handler.Warn(msg, args...) }
func (l *SlogLogger) Debug(msg string, args ...any)  { l.handler.Debug(msg, args...) }
func (l *SlogLogger) InfoContext(ctx context.Context, msg string, args ...any)  { l.handler.InfoContext(ctx, msg, args...) }
func (l *SlogLogger) ErrorContext(ctx context.Context, msg string, args ...any) { l.handler.ErrorContext(ctx, msg, args...) }
func (l *SlogLogger) WarnContext(ctx context.Context, msg string, args ...any)  { l.handler.WarnContext(ctx, msg, args...) }
func (l *SlogLogger) DebugContext(ctx context.Context, msg string, args ...any) { l.handler.DebugContext(ctx, msg, args...) }
func (l *SlogLogger) LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
    l.handler.LogAttrs(ctx, level, msg, attrs...)
}

// contextHandler adds request-scoped values from ctx (request_id) to every log record
type contextHandler struct {
	slog.Handler
}

func (h contextHandler) Handle(ctx context.Context, record slog.Record) error {
	if requestID := utils.RequestIDFromContext(ctx); requestID != "" {
		record.AddAttrs(slog.String("request_id", requestID))
	}

	return h.Handler.Handle(ctx, record)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{h.Handler.WithGroup(name)}
}
