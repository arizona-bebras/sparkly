package domain

import "errors"

var (
	ErrDuplicatedUser = errors.New("this user already exists")
)