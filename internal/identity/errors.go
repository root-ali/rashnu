package identity

import "errors"

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrUserAlreadyExist = errors.New("user already exist")
	ErrPasswordMismatch = errors.New("password mismatch")
	ErrTokenRevoked     = errors.New("token has been revoked")
)
