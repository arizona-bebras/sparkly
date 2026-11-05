package scope

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func UserParkScope(userID uuid.UUID) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("parks.id IN (SELECT user_parks.park_id FROM user_parks WHERE user_parks.user_id = ?)", userID)
	}
}

func UserGroupScope(userID uuid.UUID) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("groups.park_id IN (SELECT user_parks.park_id FROM user_parks WHERE user_parks.user_id = ?)", userID)
	}
}

func UserIssueScope(userID uuid.UUID) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("issues.park_id IN (SELECT user_parks.park_id FROM user_parks WHERE user_parks.user_id = ?)", userID)
	}
}
