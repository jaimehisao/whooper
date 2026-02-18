package models

type PaginatedResponse[T any] struct {
	Records       []T    `json:"records"`
	NextToken     string `json:"next_token,omitempty"`
}
