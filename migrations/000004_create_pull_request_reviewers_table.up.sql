CREATE TABLE pull_request_reviewers (
    id BIGSERIAL PRIMARY KEY,
    pull_request_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_reviewers_pull_request_id FOREIGN KEY (pull_request_id) 
        REFERENCES pull_requests(pull_request_id) ON DELETE RESTRICT,
    CONSTRAINT fk_reviewers_user_id FOREIGN KEY (user_id) 
        REFERENCES users(user_id) ON DELETE RESTRICT,
    CONSTRAINT uq_reviewers_pr_user UNIQUE (pull_request_id, user_id),
    CONSTRAINT chk_pull_request_id_length CHECK (LENGTH(pull_request_id) BETWEEN 1 AND 255),
    CONSTRAINT chk_user_id_length CHECK (LENGTH(user_id) BETWEEN 1 AND 255)
);

CREATE INDEX idx_reviewers_user_id ON pull_request_reviewers(user_id);
CREATE INDEX idx_reviewers_pull_request_id ON pull_request_reviewers(pull_request_id);

-- Constraint to limit max 2 reviewers per PR (with race condition protection)
CREATE OR REPLACE FUNCTION check_max_reviewers()
RETURNS TRIGGER AS $$
DECLARE
    reviewer_count INTEGER;
BEGIN
    -- Lock rows to prevent race condition
    SELECT COUNT(*) INTO reviewer_count
    FROM pull_request_reviewers
    WHERE pull_request_id = NEW.pull_request_id
    FOR UPDATE;
    
    IF reviewer_count >= 2 THEN
        RAISE EXCEPTION 'Maximum 2 reviewers allowed per pull request';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_check_max_reviewers
    BEFORE INSERT ON pull_request_reviewers
    FOR EACH ROW
    EXECUTE FUNCTION check_max_reviewers();

-- Constraint to prevent author from being reviewer
CREATE OR REPLACE FUNCTION check_reviewer_not_author()
RETURNS TRIGGER AS $$
DECLARE
    pr_author_id VARCHAR(255);
BEGIN
    SELECT author_id INTO pr_author_id
    FROM pull_requests
    WHERE pull_request_id = NEW.pull_request_id;
    
    IF pr_author_id = NEW.user_id THEN
        RAISE EXCEPTION 'Author cannot be assigned as reviewer';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_check_reviewer_not_author
    BEFORE INSERT OR UPDATE ON pull_request_reviewers
    FOR EACH ROW
    EXECUTE FUNCTION check_reviewer_not_author();

