package services

import (
	"context"
	"github.com/google/uuid"

	"sparkly/backend/internal/metadata"
	"sparkly/backend/internal/domain"
)

func parseCtxUserIDValue(ctx context.Context) (uuid.UUID, error) {
	userIDStr, err := metadata.UserIDFromContext(ctx)
	if err != nil {
		return uuid.UUID{}, err
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.UUID{}, domain.ErrInvalidID
	}

	return userID, nil
}