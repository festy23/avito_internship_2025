package model

import (
	"time"

	"gorm.io/gorm"
)

// Team represents a team entity in the system.
// Matches the teams table schema.
type Team struct {
	TeamName  string    `gorm:"primaryKey;column:team_name;type:varchar(255)" json:"team_name"`
	CreatedAt time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"-"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"-"`
}

// TableName specifies the table name for GORM.
func (Team) TableName() string {
	return "teams"
}

// BeforeUpdate updates the UpdatedAt timestamp before saving.
func (t *Team) BeforeUpdate(tx *gorm.DB) error {
	t.UpdatedAt = time.Now()
	return nil
}

