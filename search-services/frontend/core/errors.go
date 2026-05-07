package core

import "errors"

var (
	ErrRateLimited  = errors.New("rate limited")
	ErrUnauthorized = errors.New("unauthorized")
)
