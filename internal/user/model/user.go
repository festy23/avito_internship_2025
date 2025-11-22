package model

import (
	"time"

	"gorm.io/gorm"
)

// User represents a user entity in the system.
// Matches the users table schema.
type User struct {
	UserID    string    `gorm:"primaryKey;column:user_id;type:varchar(255)"                                                         json:"user_id"`
	Username  string    `gorm:"column:username;type:varchar(255);not null"                                                          json:"username"`
	TeamName  string    `gorm:"column:team_name;type:varchar(255);not null;index:idx_users_team_name"                               json:"team_name"`
	IsActive  bool      `gorm:"column:is_active;type:boolean;not null;default:true;index:idx_users_team_active,composite:team_name" json:"is_active"`
	CreatedAt time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()"                                           json:"-"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()"                                           json:"-"`
}

// TableName specifies the table name for GORM.
func (User) TableName() string {
	return "users"
}

// BeforeUpdate updates the UpdatedAt timestamp before saving.
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	u.UpdatedAt = time.Now()
	return nil
}
