package model

import (
	"time"
)

// PullRequest represents a pull request entity in the system.
// Matches the pull_requests table schema.
type PullRequest struct {
	PullRequestID   string     `gorm:"primaryKey;column:pull_request_id;type:varchar(255)"                           json:"pull_request_id"`
	PullRequestName string     `gorm:"column:pull_request_name;type:varchar(255);not null"                           json:"pull_request_name"`
	AuthorID        string     `gorm:"column:author_id;type:varchar(255);not null;index:idx_pull_requests_author_id" json:"author_id"`
	Status          string     `gorm:"column:status;type:pr_status_enum;not null;index:idx_pull_requests_status"     json:"status"`
	CreatedAt       time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()"                     json:"createdAt"`
	MergedAt        *time.Time `gorm:"column:merged_at;type:timestamptz"                                             json:"mergedAt,omitempty"`
}

// TableName specifies the table name for GORM.
func (PullRequest) TableName() string {
	return "pull_requests"
}

// PullRequestReviewer represents a reviewer assignment for a pull request.
// Matches the pull_request_reviewers table schema.
type PullRequestReviewer struct {
	ID            int64     `gorm:"primaryKey;column:id;type:bigserial"                                                   json:"id"`
	PullRequestID string    `gorm:"column:pull_request_id;type:varchar(255);not null;index:idx_reviewers_pull_request_id" json:"pull_request_id"`
	UserID        string    `gorm:"column:user_id;type:varchar(255);not null;index:idx_reviewers_user_id"                 json:"user_id"`
	AssignedAt    time.Time `gorm:"column:assigned_at;type:timestamptz;not null;default:now()"                            json:"assigned_at"`
}

// TableName specifies the table name for GORM.
func (PullRequestReviewer) TableName() string {
	return "pull_request_reviewers"
}
