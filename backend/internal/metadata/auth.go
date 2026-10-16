package metadata

import (
	"context"
	"fmt"
	"sparkly/backend/internal/domain"

	"connectrpc.com/connect/v2"
	"github.com/google/uuid"
)

type contextKey string

const userIDContextKey contextKey = "userID"

func NewAuthInterceptor() connect.ServerInterceptor {
	return func(next connect.ServerFunc) connect.ServerFunc {
		return func(ctx context.Context, req connect.Spec, stream connect.ServerStream) error {
			info, ok := connect.CallInfoForServerContext(ctx)
			if !ok {
				return fmt.Errorf("call info not provided")
			}

			userID := info.RequestHeader().Get("X-User-Id")

			ctx = context.WithValue(ctx, userIDContextKey, userID)
			return next(ctx, req, stream)
		}
	}
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	val, ok := ctx.Value(userIDContextKey).(string)
	if !ok {
		return uuid.UUID{}, domain.ErrUnknownUser
	}
	id, err := uuid.Parse(val)
	if err != nil {
		return uuid.UUID{}, err
	}
	
	return id, nil
}