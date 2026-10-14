package openai

import (
	"net"
	"net/http"
	"time"
)

// resilientTransport retries the initial connection to the OpenAI API a
// limited number of times on transient network errors (e.g. connection
// reset, DNS hiccup) before giving up. It only guards the connect/handshake
// phase (DialContext/TLSHandshakeTimeout/ResponseHeaderTimeout) — once
// response headers arrive and streaming has started, reads are never
// capped, so a long completion streamed token-by-token is never cut short
// by this transport.
type resilientTransport struct {
	base    http.RoundTripper
	retries int
	backoff time.Duration
}

func (t *resilientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= t.retries; attempt++ {
		attemptReq := req
		if attempt > 0 {
			// A body can only be safely retried if the standard library
			// captured a way to re-read it (GetBody is set automatically
			// for bytes.Buffer/bytes.Reader/strings.Reader bodies, which is
			// what langchaingo uses for its JSON request payloads).
			if req.GetBody == nil {
				return nil, lastErr
			}

			body, err := req.GetBody()
			if err != nil {
				return nil, lastErr
			}

			clone := req.Clone(req.Context())
			clone.Body = body
			attemptReq = clone
		}

		resp, err := t.base.RoundTrip(attemptReq)
		if err == nil {
			return resp, nil
		}
		lastErr = err

		if _, isNetErr := err.(net.Error); !isNetErr {
			return nil, err
		}
		if attempt < t.retries {
			time.Sleep(t.backoff * time.Duration(attempt+1))
		}
	}

	return nil, lastErr
}

// newResilientHTTPClient builds an http.Client for the OpenAI provider that
// tolerates transient connect-phase failures on large request bodies (e.g.
// a long PRD) instead of surfacing them immediately as a hard
// "network error: failed to reach API server" before any token streams.
func newResilientHTTPClient() *http.Client {
	return &http.Client{
		Transport: &resilientTransport{
			retries: 2,
			backoff: 250 * time.Millisecond,
			base: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: 15 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout:   15 * time.Second,
				ResponseHeaderTimeout: 60 * time.Second,
			},
		},
	}
}
