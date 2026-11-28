package repos

import (
	"context"
	"fmt"
	"sparkly/analytics/internal/domain"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type GroupMembershipRepo struct {
	db driver.Conn
}

func NewGroupMembershipRepo(db driver.Conn) *GroupMembershipRepo {
	return &GroupMembershipRepo{
		db: db,
	}
}

func (r *GroupMembershipRepo) WriteAssociatedGroups(ctx context.Context, params domain.GroupsToDeviceAssociationParams) error {
	groupIDs := make([]string, 0, len(params.GroupIDs))
	for _, id := range params.GroupIDs {
		groupIDs = append(groupIDs, id.String())
	}

	ctx = clickhouse.Context(ctx)
	if err := r.db.Exec(ctx,
		`INSERT INTO group_membership (device_id, group_ids, version)
		VALUES (toUUID(?), CAST(? AS Array(UUID)), fromUnixTimestamp64Milli(?, 'UTC'))`,
		params.DeviceID.String(), groupIDs, params.Version.UTC().UnixMilli()); err != nil {
		return fmt.Errorf("failed to write groups membership to db: %w", err)
	}

	return nil
}
