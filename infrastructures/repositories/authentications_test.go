package repositories

import (
	"context"
	"testing"

	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/infrastructures/configurations"
)

func newTestRepository(application models.ApplicationConfig) *AuthenticationsRepositories {
	configs := &configurations.Configs{
		Envs:   &models.ENVConfig{Application: application},
		Logger: configurations.NewSlogLogger(),
	}

	return NewAuthenticationsRepositories(nil, configs)
}

func TestVerifyClientID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		allowed    []string
		clientID   string
		wantAccept bool
	}{
		{name: "allowed client", allowed: []string{"web", "mobile"}, clientID: "web", wantAccept: true},
		{name: "unknown client", allowed: []string{"web"}, clientID: "evil", wantAccept: false},
		{name: "wildcard allows any", allowed: []string{"*"}, clientID: "anything", wantAccept: true},
		{name: "empty client rejected even with wildcard", allowed: []string{"*"}, clientID: "", wantAccept: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := newTestRepository(models.ApplicationConfig{AllowedCleintIDs: tt.allowed})

			err := repository.VerifyClientID(ctx, tt.clientID)
			if (err == nil) != tt.wantAccept {
				t.Errorf("VerifyClientID(%q) error = %v, wantAccept %v", tt.clientID, err, tt.wantAccept)
			}
		})
	}
}

func TestVerifySecretToken(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		configured string
		token      string
		wantAccept bool
	}{
		{name: "correct token", configured: "s3cr3t-token", token: "s3cr3t-token", wantAccept: true},
		{name: "wrong token", configured: "s3cr3t-token", token: "s3cr3t-tokeX", wantAccept: false},
		{name: "prefix of token", configured: "s3cr3t-token", token: "s3cr3t", wantAccept: false},
		{name: "empty token", configured: "s3cr3t-token", token: "", wantAccept: false},
		{name: "not configured rejects everything", configured: "", token: "", wantAccept: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := newTestRepository(models.ApplicationConfig{SecretToken: tt.configured})

			err := repository.VerifySecretToken(ctx, tt.token)
			if (err == nil) != tt.wantAccept {
				t.Errorf("VerifySecretToken(%q) error = %v, wantAccept %v", tt.token, err, tt.wantAccept)
			}
		})
	}
}

func TestVerifyAuthToken(t *testing.T) {
	repository := newTestRepository(models.ApplicationConfig{})
	ctx := context.Background()

	token, err := repository.VerifyAuthToken(ctx, "Bearer abc.def.ghi")
	if err != nil || token != "abc.def.ghi" {
		t.Errorf("valid bearer: got (%q, %v)", token, err)
	}

	for _, header := range []string{"", "abc.def.ghi", "Basic abc", "Bearer "} {
		if _, err := repository.VerifyAuthToken(ctx, header); err == nil {
			t.Errorf("header %q should be rejected", header)
		}
	}
}
