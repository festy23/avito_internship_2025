CREATE TYPE pr_status_enum AS ENUM ('OPEN', 'MERGED');

CREATE TABLE pull_requests (
    pull_request_id VARCHAR(255) PRIMARY KEY,
    pull_request_name VARCHAR(255) NOT NULL,
    author_id VARCHAR(255) NOT NULL,
    status pr_status_enum NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    merged_at TIMESTAMPTZ,
    CONSTRAINT fk_pull_requests_author_id FOREIGN KEY (author_id) 
        REFERENCES users(user_id) ON DELETE RESTRICT,
    CONSTRAINT chk_pull_request_id_length CHECK (LENGTH(pull_request_id) BETWEEN 1 AND 255),
    CONSTRAINT chk_pull_request_name_length CHECK (LENGTH(pull_request_name) BETWEEN 1 AND 255),
    CONSTRAINT chk_author_id_length CHECK (LENGTH(author_id) BETWEEN 1 AND 255)
);

CREATE INDEX idx_pull_requests_author_id ON pull_requests(author_id);
CREATE INDEX idx_pull_requests_status ON pull_requests(status);

