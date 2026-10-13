package utils_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"go-serviceboilerplate/commons/utils"
)

type sseEvent struct {
	name string
	data string
}

// parseSSE mirrors the frontend useSSEventStream parser: events are split by a blank line,
// and the "data:" lines of an event are joined with "\n"
func parseSSE(t *testing.T, body string) []sseEvent {
	t.Helper()

	var events []sseEvent
	for _, part := range strings.Split(body, "\n\n") {
		if strings.TrimSpace(part) == "" {
			continue
		}

		event := sseEvent{name: "message"}
		var dataLines []string
		for _, line := range strings.Split(part, "\n") {
			switch {
			case strings.HasPrefix(line, "event:"):
				event.name = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			case strings.HasPrefix(line, "data:"):
				dataLines = append(dataLines, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
			default:
				t.Errorf("line outside a known field would be dropped by the client: %q", line)
			}
		}
		event.data = strings.Join(dataLines, "\n")
		events = append(events, event)
	}

	return events
}

func writeEvents(t *testing.T, write func(c echo.Context)) []sseEvent {
	t.Helper()

	rec := httptest.NewRecorder()
	c := echo.New().NewContext(httptest.NewRequest(http.MethodPost, "/", nil), rec)
	write(c)

	if contentType := rec.Header().Get(echo.HeaderContentType); contentType != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", contentType)
	}

	return parseSSE(t, rec.Body.String())
}

func TestSuccessResponseSSEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    string
		data     any
		wantName string
		wantData string
	}{
		{name: "pre-encoded JSON string is not encoded twice", event: "token", data: `{"token":"Hi"}`, wantName: "token", wantData: `{"token":"Hi"}`},
		{name: "plain string is sent as-is", event: "complete", data: "[DONE]", wantName: "complete", wantData: "[DONE]"},
		{name: "struct is JSON-encoded", event: "token", data: struct {
			Token string `json:"token"`
		}{Token: "Hi"}, wantName: "token", wantData: `{"token":"Hi"}`},
		{name: "map is JSON-encoded", event: "meta", data: map[string]int{"count": 2}, wantName: "meta", wantData: `{"count":2}`},
		{name: "multi-line string survives", event: "token", data: "## Title\n\n- item\r\n- item 2", wantName: "token", wantData: "## Title\n\n- item\n- item 2"},
		{name: "struct with newlines stays one JSON value", event: "token", data: map[string]string{"token": "a\nb"}, wantName: "token", wantData: `{"token":"a\nb"}`},
		{name: "newline in event name is removed", event: "tok\nen", data: "x", wantName: "token", wantData: "x"},
		{name: "no event name defaults to message", event: "", data: "x", wantName: "message", wantData: "x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := writeEvents(t, func(c echo.Context) {
				if err := utils.SuccessResponseSSEvent(c, utils.SuccessResponseSSEventConfig{Event: tt.event, Data: tt.data}); err != nil {
					t.Fatalf("SuccessResponseSSEvent: %v", err)
				}
			})

			if len(events) != 1 {
				t.Fatalf("got %d events, want 1: %+v", len(events), events)
			}
			if events[0].name != tt.wantName || events[0].data != tt.wantData {
				t.Errorf("got (%q, %q), want (%q, %q)", events[0].name, events[0].data, tt.wantName, tt.wantData)
			}
		})
	}
}

func TestSSEStreamOfManyEvents(t *testing.T) {
	events := writeEvents(t, func(c echo.Context) {
		for _, token := range []string{"Hello", "\n\n", " world"} {
			payload, _ := json.Marshal(map[string]string{"token": token})
			utils.SuccessResponseSSEvent(c, utils.SuccessResponseSSEventConfig{Event: "token", Data: string(payload)})
		}
		utils.SuccessResponseSSEvent(c, utils.SuccessResponseSSEventConfig{Event: "complete", Data: "[DONE]"})
	})

	var text string
	for _, event := range events[:3] {
		var payload map[string]string
		if err := json.Unmarshal([]byte(event.data), &payload); err != nil {
			t.Fatalf("token event is not JSON: %q", event.data)
		}
		text += payload["token"]
	}

	if text != "Hello\n\n world" || len(events) != 4 || events[3].name != "complete" {
		t.Errorf("stream reassembled as %q with %d events", text, len(events))
	}
}

func TestErrorResponseSSEvent(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantCode    float64
		wantMessage string
		wantType    string
	}{
		{name: "application exception", err: utils.NewClientError(errors.New("invalid prompt")), wantCode: 400, wantMessage: "invalid prompt", wantType: utils.ClientError},
		{name: "internal error is hidden", err: errors.New("dial tcp llm-provider.internal:443: timeout"), wantCode: 500, wantMessage: "Internal Server Error", wantType: "INTERNALSERVER_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := writeEvents(t, func(c echo.Context) {
				if err := utils.ErrorResponseSSEvent(c, tt.err); err != nil {
					t.Fatalf("ErrorResponseSSEvent: %v", err)
				}
			})

			if len(events) != 1 || events[0].name != "error" {
				t.Fatalf("want one error event, got %+v", events)
			}

			var body map[string]any
			if err := json.Unmarshal([]byte(events[0].data), &body); err != nil {
				t.Fatalf("error event data is not JSON: %q", events[0].data)
			}

			errorBody := body["error"].(map[string]any)
			if body["code"] != tt.wantCode || body["message"] != tt.wantMessage || errorBody["type"] != tt.wantType {
				t.Errorf("unexpected error body: %v", body)
			}
			if id, _ := errorBody["id"].(string); id == "" {
				t.Error("error event must include an error id")
			}
			if strings.Contains(events[0].data, "llm-provider.internal") {
				t.Errorf("error event leaks internal details: %s", events[0].data)
			}
		})
	}
}
