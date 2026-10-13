package security

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashAndComparePassword(t *testing.T) {
	passwordHash := NewPasswordHashSecurity(nil, bcrypt.MinCost)

	hash, err := passwordHash.Hash("Secret123!")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	if !passwordHash.ComparePassword("Secret123!", hash) {
		t.Error("correct password should match")
	}
	if passwordHash.ComparePassword("Secret123?", hash) {
		t.Error("wrong password should not match")
	}
}

func TestValidatePassword(t *testing.T) {
	passwordHash := NewPasswordHashSecurity(nil, bcrypt.MinCost)

	tests := []struct {
		password  string
		wantValid bool
	}{
		{password: "Secret123!", wantValid: true},
		{password: "Sh0rt!", wantValid: false},
		{password: "NoDigits!!", wantValid: false},
		{password: "12345678!", wantValid: false},
		{password: "NoSpecial123", wantValid: false},
		{password: "Has Space1!", wantValid: false},
		{password: "Unsafe123<", wantValid: false},
	}

	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			err := passwordHash.ValidatePassword(tt.password)
			if (err == nil) != tt.wantValid {
				t.Errorf("ValidatePassword(%q) error = %v, wantValid %v", tt.password, err, tt.wantValid)
			}
		})
	}
}
