DROP TRIGGER IF EXISTS trigger_check_reviewer_not_author ON pull_request_reviewers;
DROP TRIGGER IF EXISTS trigger_check_max_reviewers ON pull_request_reviewers;
DROP FUNCTION IF EXISTS check_reviewer_not_author();
DROP FUNCTION IF EXISTS check_max_reviewers();
DROP TABLE IF EXISTS pull_request_reviewers;

