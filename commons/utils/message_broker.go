package utils

import "encoding/json"

// EncodeMessagePayload converts a payload into a message body and its content type.
// []byte is sent as-is, string as plain text, and anything else is encoded as JSON.
func EncodeMessagePayload(payload any) ([]byte, string, error) {
	switch p := payload.(type) {
	case []byte:
		return p, "application/octet-stream", nil
	case string:
		return []byte(p), "text/plain", nil
	default:
		data, err := json.Marshal(p)
		if err != nil {
			return nil, "", err
		}

		return data, "application/json", nil
	}
}
