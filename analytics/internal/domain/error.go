package domain

import "errors"

var (
	ErrWrongBrokerMsgContentType = errors.New("wrong message content type")
	ErrInvalidBrokerMsg          = errors.New("invalid broker message")
)
