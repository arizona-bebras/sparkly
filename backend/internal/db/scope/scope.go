package scope

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserParkScope filters parks belonging to the given user via the user_parks join table.
func UserParkScope(userID uuid.UUID) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("parks.id IN (SELECT user_parks.park_id FROM user_parks WHERE user_parks.user_id = ?)", userID)
	}
}