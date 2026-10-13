package utils_test

import (
	"testing"

	"go-serviceboilerplate/commons/utils"
)

func TestEncodeMessagePayload(t *testing.T) {
	tests := []struct {
		name            string
		payload         any
		wantBody        string
		wantContentType string
	}{
		{name: "bytes as-is", payload: []byte{0x01, 0x02}, wantBody: "\x01\x02", wantContentType: "application/octet-stream"},
		{name: "string as text", payload: "hello", wantBody: "hello", wantContentType: "text/plain"},
		{name: "map as JSON", payload: map[string]string{"a": "b"}, wantBody: `{"a":"b"}`, wantContentType: "application/json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType, err := utils.EncodeMessagePayload(tt.payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(body) != tt.wantBody || contentType != tt.wantContentType {
				t.Errorf("got (%q, %q), want (%q, %q)", body, contentType, tt.wantBody, tt.wantContentType)
			}
		})
	}

	if _, _, err := utils.EncodeMessagePayload(make(chan int)); err == nil {
		t.Error("unencodable payload should return an error")
	}
}
