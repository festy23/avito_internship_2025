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

