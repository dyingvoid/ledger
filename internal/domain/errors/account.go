package domain_errors

import "errors"

var (
	ErrAccountIDRequired   = errors.New("account ID is required")
	ErrAccountNameRequired = errors.New("account name is required")
	ErrAccountUserRequired = errors.New("account user ID is required")
)
