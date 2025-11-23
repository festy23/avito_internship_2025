// Package model provides data transfer objects for statistics module.
package model

// ReviewerStatistics represents statistics for a reviewer.
type ReviewerStatistics struct {
	UserID          string `json:"user_id"`
	Username        string `json:"username"`
	TeamName        string `json:"team_name"`
	AssignmentCount int    `json:"assignment_count"`
	IsActive        bool   `json:"is_active"`
}

// ReviewersStatisticsResponse represents response for reviewers statistics.
type ReviewersStatisticsResponse struct {
	Reviewers []ReviewerStatistics `json:"reviewers"`
	Total     int                  `json:"total"`
}

// PullRequestStatistics represents statistics for pull requests.
type PullRequestStatistics struct {
	TotalPRs              int     `json:"total_prs"`
	OpenPRs               int     `json:"open_prs"`
	MergedPRs             int     `json:"merged_prs"`
	AverageReviewersPerPR float64 `json:"average_reviewers_per_pr"`
	PRsWith0Reviewers     int     `json:"prs_with_0_reviewers"`
	PRsWith1Reviewer      int     `json:"prs_with_1_reviewer"`
	PRsWith2Reviewers     int     `json:"prs_with_2_reviewers"`
}

// PullRequestStatisticsResponse represents response for pull request statistics.
type PullRequestStatisticsResponse struct {
	Statistics PullRequestStatistics `json:"statistics"`
}
