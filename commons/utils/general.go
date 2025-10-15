package utils

import (
	"crypto/rand"
	"encoding/hex"
)

func IsSafeSpecialChar(r rune) bool {
	var safeSpecialChars = "!@#$%^&*-_=+?."
	for _, c := range safeSpecialChars {
		if r == c {
			return true
		}
	}
	return false
}

func GenerateTokenID(length int) (string, error) {
    // Allocate byte slice
    b := make([]byte, length)
    // Fill with secure random data
    _, err := rand.Read(b)
    if err != nil {
        return "", err
    }
    // Encode as hex string
    return hex.EncodeToString(b), nil
}