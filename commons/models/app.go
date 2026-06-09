package models

import "context"

type BaseRequest struct {
	Ctx context.Context
}

type BasePayload struct {
	Ctx context.Context
}