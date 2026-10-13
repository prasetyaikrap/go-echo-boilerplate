package utils_test

import (
	"context"
	"testing"

	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/commons/utils"
)

func TestRequestIDFromContext(t *testing.T) {
	ctx := utils.WithRequestID(context.Background(), "req-1")

	if got := utils.RequestIDFromContext(ctx); got != "req-1" {
		t.Errorf("got %q, want req-1", got)
	}
	if got := utils.RequestIDFromContext(context.Background()); got != "" {
		t.Errorf("empty context: got %q, want empty", got)
	}
}

func TestAccessTokenClaimsFromContext(t *testing.T) {
	ctx := utils.WithAccessTokenClaims(context.Background(), &models.AccessTokenClaims{SessionID: "session-1"})

	got, ok := utils.AccessTokenClaimsFromContext(ctx)
	if !ok || got.SessionID != "session-1" {
		t.Errorf("got (%+v, %v), want session-1", got, ok)
	}

	if _, ok := utils.AccessTokenClaimsFromContext(context.Background()); ok {
		t.Error("empty context: ok should be false")
	}
	if _, ok := utils.AccessTokenClaimsFromContext(utils.WithAccessTokenClaims(context.Background(), nil)); ok {
		t.Error("nil claims: ok should be false")
	}

	refreshCtx := utils.WithRefreshTokenClaims(context.Background(), &models.RefreshTokenClaims{SessionID: "r"})
	if _, ok := utils.AccessTokenClaimsFromContext(refreshCtx); ok {
		t.Error("refresh claims must not be readable as access claims")
	}
}

func TestRefreshTokenClaimsFromContext(t *testing.T) {
	ctx := utils.WithRefreshTokenClaims(context.Background(), &models.RefreshTokenClaims{SessionID: "session-2"})

	got, ok := utils.RefreshTokenClaimsFromContext(ctx)
	if !ok || got.SessionID != "session-2" {
		t.Errorf("got (%+v, %v), want session-2", got, ok)
	}
}
