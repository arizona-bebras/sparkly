package domain

import "errors"

var (
	ErrWrongBrokerMsgContentType = errors.New("wrong message content type")
)