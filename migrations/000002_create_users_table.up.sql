CREATE TABLE users (
    user_id VARCHAR(255) PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    team_name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_users_team_name FOREIGN KEY (team_name) 
        REFERENCES teams(team_name) ON DELETE RESTRICT,
    CONSTRAINT chk_user_id_length CHECK (LENGTH(user_id) BETWEEN 1 AND 255),
    CONSTRAINT chk_username_length CHECK (LENGTH(username) BETWEEN 1 AND 255),
    CONSTRAINT chk_team_name_length CHECK (LENGTH(team_name) BETWEEN 1 AND 255)
);

CREATE INDEX idx_users_team_name ON users(team_name);
CREATE INDEX idx_users_team_active ON users(team_name, is_active);

