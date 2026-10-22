package contract

import "errors"

var (
	ErrNotFound             = errors.New("site connection not found")
	ErrInvalid              = errors.New("site connection is invalid")
	ErrAuthenticationPaused = errors.New("site connection authentication paused")
)
