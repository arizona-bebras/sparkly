package domain

import "errors"

var (
	ErrInvalidID = errors.New("invalid id, expected UUID")

	ErrUserNotFound        = errors.New("user not found")
	ErrUnknownUser         = errors.New("user's ID not provided")
	ErrDuplicatedUser      = errors.New("this user already exists")
	ErrWrongPassword       = errors.New("wrong password")
	ErrTokenNotFound       = errors.New("token not found")
	ErrExpiredRefreshToken = errors.New("refresh token expired")
	ErrRevokedRefreshToken = errors.New("refresh token has been revoked")

	ErrKeyNotFound = errors.New("unknown key id")

	ErrParkNotFound     = errors.New("park with this ID or token doesnt exists")
	ErrParkAccessDenied = errors.New("you are not member of this park")

	ErrGroupNotFound = errors.New("group with this ID doesnt exists")

	ErrDuplicatedDevice = errors.New("this device is already registered")
	ErrDeviceNotFound   = errors.New("device not found")

	ErrBuildNotFound = errors.New("build with this ID doesnt exists")

	ErrReleaseNotFound = errors.New("release with this ID doesnt exists")

	ErrIssueNotFound        = errors.New("issue with this ID doesnt exists")
	ErrIssueAlreadyResolved = errors.New("this issue already resolved")
)
