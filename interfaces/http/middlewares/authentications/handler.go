package authentications

import (
	"go-serviceboilerplate/applications/usecases"
	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/interfaces/utils"

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
		clientID := c.Request().Header.Get(models.XClientIdHeader)

		err := m.AuthenticationUsecase.VerifyClient(clientID)
		if(err != nil) {
			return utils.ErrorResponse(c, err)
		}

		return next(c)
	}
}

func (m *AuthMiddlewareHandler) VerifyAuthentication(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		bearerToken := c.Request().Header.Get(models.AuthorizationHeader)
		isRenewToken := c.Request().Header.Get(models.XRenewTokenHeader) == "true"

		if isRenewToken {
			refreshTokenClaims, err := m.AuthenticationUsecase.VerifyRefreshToken(bearerToken)
			if(err != nil) {
				return utils.ErrorResponse(c, err)
			}

			c.Set(models.ContextUserClaim, refreshTokenClaims)
			return next(c)
		}
 
		accessTokenClaims, err := m.AuthenticationUsecase.VerifyAccessToken(bearerToken)
		if(err != nil) {
			return utils.ErrorResponse(c, err)
		}

		c.Set(models.ContextUserClaim, accessTokenClaims)

		return next(c)
	}
}

func (m *AuthMiddlewareHandler) VerifyAuthenticationWithServiceToken(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		serviceToken := c.Request().Header.Get(models.XServiceTokenHeader)
		bearerToken := c.Request().Header.Get(models.AuthorizationHeader)

		err := m.AuthenticationUsecase.VerifySecretToken(serviceToken)
		if(err != nil) {
			return utils.ErrorResponse(c, err)
		}

		accessTokenClaims, err := m.AuthenticationUsecase.VerifyAccessToken(bearerToken)
		if(err != nil) {
			return utils.ErrorResponse(c, err)
		}

		c.Set(models.ContextUserClaim, accessTokenClaims)

		return next(c)
	}
}



