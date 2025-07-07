package utils

import (
	"net/http"
)

type Exceptions struct {
	Type 		string
	Code		int
	ErrorObject	any
	Err  		error
}

const (
	ClientError = "CLIENT_ERROR"
	AuthenticationError = "AUTHENTICATION_ERROR"
	AuthorizationError = "AUTHORIZATION_ERROR"
	NotFoundError = "NOTFOUND_ERROR"
	InvariantError = "INVARIANT_ERROR"
)

func NewClientError(err error) *Exceptions{
	return &Exceptions{
		Type: ClientError,
		Code: http.StatusBadRequest,
		Err: err,
		ErrorObject: nil,
	}
}

func NewAuthenticationError(err error) *Exceptions {
	return &Exceptions{
		Type: AuthenticationError,
		Code: http.StatusUnauthorized,
		Err: err,
		ErrorObject: nil,
	}
}

func NewAuthorizationError(err error) *Exceptions {
	return &Exceptions{
		Type: AuthorizationError,
		Code: http.StatusForbidden,
		Err: err,
		ErrorObject: nil,
	}
}

func NewNotFoundError(err error) *Exceptions {
	return &Exceptions{
		Type: NotFoundError,
		Code: http.StatusNotFound,
		Err: err,
		ErrorObject: nil,
	}
}

func NewInvariantError(err error) *Exceptions {
	return &Exceptions{
		Type: InvariantError,
		Code: http.StatusInternalServerError,
		Err: err,
		ErrorObject: nil,
	}
}

func (e *Exceptions) Error() string {
	return e.Err.Error()
}

func (e *Exceptions) Unwrap() error {
	return e.Err
}

func (e *Exceptions) SetCode(code int) *Exceptions {
	if (e.Type != ClientError && e.Type != InvariantError) {
		return e
	} 
	
	e.Code = code
	return e
}

func (e *Exceptions) SetErrorObject(err any) *Exceptions {
	e.ErrorObject = err
	return e
}