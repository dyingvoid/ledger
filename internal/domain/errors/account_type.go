package domain_errors

import "errors"

var ErrAccountTypeInvalid = errors.New("account type must be a four digit code between 1000 and 5999")
