package scope

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func userScope(userID uuid.UUID) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("user_id = ?", userID)
	}
}