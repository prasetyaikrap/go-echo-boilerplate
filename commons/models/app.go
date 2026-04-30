package models

import "context"

type BaseRequest struct {
	Ctx context.Context
}

type BasePayload struct {
	Ctx context.Context
}

type BaseQueriesRequest struct {
	Queries string `query:"queries"`
	Limit   int64  `query:"_limit"`
	Page  	int64  `query:"_page"`
	SortBy  string `query:"_sort"`
}