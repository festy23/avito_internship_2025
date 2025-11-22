CREATE TABLE teams (
    team_name VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_team_name_length CHECK (LENGTH(team_name) BETWEEN 1 AND 255)
);

