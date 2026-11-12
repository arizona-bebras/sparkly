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
	ctx = clickhouse.Context(ctx, clickhouse.WithAsync(false))
	if _, err := r.db.ExecContext(ctx, 
	`
		INSERT INTO group_membership (device_id, group_ids, version,)
		VALUES (?, ?, ?)
	`,
	params.DeviceID, params.GroupIDs, params.Version); err != nil {
		return fmt.Errorf("failed to write groups membership to db: %w", err)
	}

	return nil
}
