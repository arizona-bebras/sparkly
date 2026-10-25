package domain

import "errors"

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrDuplicatedUser      = errors.New("this user already exists")
	ErrWrongPassword       = errors.New("wrong password")
	ErrTokenNotFound       = errors.New("token not found")
	ErrExpiredRefreshToken = errors.New("refresh token expired")
	ErrRevokedRefreshToken = errors.New("refresh token has been revoked")

	ErrInvalidID   = errors.New("invalid id, expected UUID")
	ErrUnknownUser = errors.New("user's ID not provided")

	ErrParkNotFound = errors.New("park with this ID doesnt exists")

	ErrKeyNotFound = errors.New("unknown key id")
)
