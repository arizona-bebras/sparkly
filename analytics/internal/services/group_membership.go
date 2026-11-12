package services

import (
	"context"
	"fmt"
	"sparkly/analytics/internal/domain"
	"time"

	"github.com/google/uuid"
)

type GroupMembershipRepo interface {
	WriteAssociatedGroups(ctx context.Context, params domain.GroupsToDeviceAssociationParams) error
}

type GroupMembershipService struct {
	repo GroupMembershipRepo
}

func NewGroupMembershipService(repo GroupMembershipRepo) *GroupMembershipService {
	return &GroupMembershipService{
		repo: repo,
	}
}

func (c *GroupMembershipService) AssociateGroupsToDevice(ctx context.Context, deviceIDStr string, groupIDsStr []string, versionStr string) error {
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		return fmt.Errorf("failed to parse device id from message: %w", err)
	}

	groupIDs := make([]uuid.UUID, 0, len(groupIDsStr))
	for _, groupID := range groupIDsStr {
		id, err := uuid.Parse(groupID)
		if err != nil {
			return fmt.Errorf("failed to parse group id from message: %w", err)
		}
		groupIDs = append(groupIDs, id)
	}

	version, err := time.Parse(time.RFC1123, versionStr)
	if err != nil {
		return fmt.Errorf("failed to parse group membership version from message: %w", err)
	}

	err = c.repo.WriteAssociatedGroups(ctx, domain.GroupsToDeviceAssociationParams{
		DeviceID: deviceID,
		GroupIDs: groupIDs,
		Version:  version,
	})
	if err != nil {
		return fmt.Errorf("failed to write device group membership: %w", err)
	}

	return nil
}
