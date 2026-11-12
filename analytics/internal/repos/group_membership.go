package repos

import (
	"context"
	"database/sql"
	"fmt"
	"sparkly/analytics/internal/domain"

	"github.com/ClickHouse/clickhouse-go/v2"
)

type GroupMembershipRepo struct {
	db *sql.DB
}

func NewGroupMembershipRepo(db *sql.DB) *GroupMembershipRepo {
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
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO group_membership (device_id, group_ids, version)
		VALUES (toUUID(?), CAST(? AS Array(UUID)), fromUnixTimestamp64Milli(?, 'UTC'))`,
		params.DeviceID.String(), groupIDs, params.Version.UTC().UnixMilli()); err != nil {
		return fmt.Errorf("failed to write groups membership to db: %w", err)
	}

	return nil
}
