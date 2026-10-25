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

	ErrKeyNotFound = errors.New("unknown key id")
	ErrParkNotFound = errors.New("park with this ID ro token doesnt exists")

	ErrDuplicatedDevice = errors.New("this device is already registered")
)
