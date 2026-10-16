package domain

import "errors"

var (
	ErrUserNotFound   = errors.New("user not found")
	ErrDuplicatedUser = errors.New("this user already exists")
	ErrWrongPassword  = errors.New("wrong password")
)
