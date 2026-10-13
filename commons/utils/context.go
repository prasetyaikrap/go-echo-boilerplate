package utils

import (
	"context"
	"go-serviceboilerplate/commons/models"

	"github.com/labstack/echo/v4"
)

// SetRequestContext replaces the context of the current HTTP request, so the next middlewares and the handler
// receive the new values through c.Request().Context()
func SetRequestContext(c echo.Context, ctx context.Context) {
	c.SetRequest(c.Request().WithContext(ctx))
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, models.ContextKeyRequestID, requestID)
}

// RequestIDFromContext returns the request ID, or "" when the context has none
func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(models.ContextKeyRequestID).(string)
	return requestID
}

func WithAccessTokenClaims(ctx context.Context, claims *models.AccessTokenClaims) context.Context {
	return context.WithValue(ctx, models.ContextKeyAccessTokenClaims, claims)
}

// AccessTokenClaimsFromContext returns the caller's verified access token claims. ok is false on unauthenticated routes
func AccessTokenClaimsFromContext(ctx context.Context) (*models.AccessTokenClaims, bool) {
	claims, ok := ctx.Value(models.ContextKeyAccessTokenClaims).(*models.AccessTokenClaims)
	return claims, ok && claims != nil
}

func WithRefreshTokenClaims(ctx context.Context, claims *models.RefreshTokenClaims) context.Context {
	return context.WithValue(ctx, models.ContextKeyRefreshTokenClaims, claims)
}

// RefreshTokenClaimsFromContext returns the verified refresh token claims. ok is false unless the request renews a token
func RefreshTokenClaimsFromContext(ctx context.Context) (*models.RefreshTokenClaims, bool) {
	claims, ok := ctx.Value(models.ContextKeyRefreshTokenClaims).(*models.RefreshTokenClaims)
	return claims, ok && claims != nil
}
