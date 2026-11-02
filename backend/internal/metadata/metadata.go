package metadata

import (
	"context"

	"sparkly/backend/internal/domain"
)

type userContextKey string
type tokenContextKey string

const UserIDContextKey userContextKey = "userID"
const TokenIDContextKey tokenContextKey = "tokenID"

func UserIDFromContext(ctx context.Context) (string, error) {
	id, ok := ctx.Value(UserIDContextKey).(string)
	if !ok {
		return "", domain.ErrUnknownUser
	}
	return id, nil
}

func TokenIDFromContext(ctx context.Context) (string, error) {
	id, ok := ctx.Value(TokenIDContextKey).(string)
	if !ok {
		return "", domain.ErrUnknownUser
	}
	return id, nil
}
