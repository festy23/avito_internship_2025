package model

import "errors"

var (
	// ErrPullRequestExists indicates that a pull request with the given ID already exists.
	ErrPullRequestExists = errors.New("pull request already exists")
	// ErrPullRequestNotFound indicates that the requested pull request does not exist.
	ErrPullRequestNotFound = errors.New("pull request not found")
	// ErrPullRequestMerged indicates that the pull request is already merged and cannot be modified.
	ErrPullRequestMerged = errors.New("pull request is merged")
	// ErrReviewerNotAssigned indicates that the user is not assigned as a reviewer for this PR.
	ErrReviewerNotAssigned = errors.New("reviewer is not assigned to this PR")
	// ErrNoCandidate indicates that there are no available candidates for assignment.
	ErrNoCandidate = errors.New("no active replacement candidate in team")
	// ErrAuthorNotFound indicates that the author user does not exist.
	ErrAuthorNotFound = errors.New("author not found")
	// ErrInvalidPullRequestID indicates that the provided pull request ID is invalid (e.g., empty).
	ErrInvalidPullRequestID = errors.New("invalid pull request ID")
	// ErrInvalidAuthorID indicates that the provided author ID is invalid (empty or too long).
	ErrInvalidAuthorID = errors.New("author_id must be between 1 and 255 characters")
	// ErrMaxReviewersExceeded indicates that the maximum number of reviewers (2) has been exceeded.
	ErrMaxReviewersExceeded = errors.New("maximum 2 reviewers allowed per pull request")
	// ErrReviewerAlreadyAssigned indicates that the reviewer is already assigned to this pull request.
	ErrReviewerAlreadyAssigned = errors.New("reviewer already assigned to this pull request")
	// ErrAuthorCannotBeReviewer indicates that the author cannot be assigned as a reviewer.
	ErrAuthorCannotBeReviewer = errors.New("author cannot be assigned as reviewer")
)
