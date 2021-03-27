package entity

import "errors"

var (
	ErrIDCollision = errors.New("user already exists")
	ErrIDMissing   = errors.New("user email cannot be blank")
	ErrNotFound    = errors.New("user does not exist")
)
