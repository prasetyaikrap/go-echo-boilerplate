package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserProfileClaims struct {
	ID			string				`json:"id"`
	Name		string				`json:"name"`
	Email		string				`json:"email"`
	Permissions	[]string			`json:"permissions"`
}

type AccessTokenClaims struct {
	SessionID		string	 			`json:"session_id"`
	Profile			UserProfileClaims	`json:"profile"`
	jwt.RegisteredClaims
}

type RefreshTokenClaims struct {
	SessionID		string	 			`json:"session_id"`
	Profile			UserProfileClaims	`json:"profile"`
	jwt.RegisteredClaims
}

type GetTokenExpirationDuration struct {
	AccessToken			time.Duration
	RefreshToken		time.Duration
}

type AccountLoginRequest struct {
	Email			string			`json:"email" validate:"required,min=3,max=128"`
	Password		string			`json:"password" validate:"required,min=8"`
}

type AccountLoginPayload struct {
	Email			string			
	Password		string
}

type AccountLoginResponse struct {
	Code 			string			`json:"code"`
	GrantType		string			`json:"grant_type"`
}

type ExchangeAuthCodeRequest struct {
	GrantType		string			`json:"grant_type" validate:"required,oneof=authorization_code"`
	Code			string			`json:"code" validate:"required"`
}

type AuthTokenResponse struct {
	AccessToken		string 			`json:"access_token"`
	RefreshToken	string 			`json:"refresh_token"`
}