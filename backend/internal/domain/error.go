package domain

import "errors"

var (
	ErrUserNotFound   = errors.New("user not found")
	ErrDuplicatedUser = errors.New("this user already exists")
	ErrWrongPassword  = errors.New("wrong password")

	ErrInvalidID   = errors.New("invalid id, expected UUID")
	ErrUnknownUser = errors.New("user's ID not provided")

	ErrParkNotFound = errors.New("park with this ID doesnt exists")
)
