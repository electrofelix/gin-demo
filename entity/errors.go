package entity

import "errors"

var (
	ErrEmailDuplicate = errors.New("email already associated with annother user")

	ErrIDMissing   = errors.New("user Id cannot be blank")
	ErrIDInvalid   = errors.New("user Id is an invalid format")
	ErrNotFound    = errors.New("user does not exist")

	ErrInternalError         = errors.New("internal server error")
	ErrUpdateFieldNotAllowed = errors.New("requested field update not allowed")
	ErrBadCredentials        = errors.New("invalid credentials")
)
