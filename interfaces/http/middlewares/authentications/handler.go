package authentications

import (
	"go-serviceboilerplate/applications/usecases"
	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/commons/utils"

	"github.com/labstack/echo/v4"
)

type AuthMiddlewareHandler struct {
	AuthenticationUsecase		*usecases.AuthenticationsUsecase
}

func NewAuthMiddlewareHandler(authenticationsUsecase *usecases.AuthenticationsUsecase) *AuthMiddlewareHandler {
	return &AuthMiddlewareHandler{authenticationsUsecase}
}

func (m *AuthMiddlewareHandler) VerifyClient(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		clientID := c.Request().Header.Get(models.XClientIdHeader)

		err := m.AuthenticationUsecase.VerifyClient(ctx, clientID)
		if(err != nil) {
			return utils.ErrorResponse(c, err)
		}

		return next(c)
	}
}

// VerifyAuthentication stores the verified claims in both echo.Context (c.Get(models.ContextUserClaim)) for handlers,
// and the request context (utils.AccessTokenClaimsFromContext / utils.RefreshTokenClaimsFromContext) for use cases and repositories
func (m *AuthMiddlewareHandler) VerifyAuthentication(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		bearerToken := c.Request().Header.Get(models.AuthorizationHeader)
		isRenewToken := c.Request().Header.Get(models.XRenewTokenHeader) == "true"

		if isRenewToken {
			refreshTokenClaims, err := m.AuthenticationUsecase.VerifyRefreshToken(ctx, bearerToken)
			if(err != nil) {
				return utils.ErrorResponse(c, err)
			}

			c.Set(models.ContextUserClaim, refreshTokenClaims)
			utils.SetRequestContext(c, utils.WithRefreshTokenClaims(ctx, refreshTokenClaims))
			return next(c)
		}

		accessTokenClaims, err := m.AuthenticationUsecase.VerifyAccessToken(ctx, bearerToken)
		if(err != nil) {
			return utils.ErrorResponse(c, err)
		}

		c.Set(models.ContextUserClaim, accessTokenClaims)
		utils.SetRequestContext(c, utils.WithAccessTokenClaims(ctx, accessTokenClaims))

		return next(c)
	}
}

func (m *AuthMiddlewareHandler) VerifyAuthenticationWithServiceToken(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		serviceToken := c.Request().Header.Get(models.XServiceTokenHeader)
		bearerToken := c.Request().Header.Get(models.AuthorizationHeader)

		err := m.AuthenticationUsecase.VerifySecretToken(ctx, serviceToken)
		if(err != nil) {
			return utils.ErrorResponse(c, err)
		}

		accessTokenClaims, err := m.AuthenticationUsecase.VerifyAccessToken(ctx, bearerToken)
		if(err != nil) {
			return utils.ErrorResponse(c, err)
		}

		c.Set(models.ContextUserClaim, accessTokenClaims)
		utils.SetRequestContext(c, utils.WithAccessTokenClaims(ctx, accessTokenClaims))

		return next(c)
	}
}
