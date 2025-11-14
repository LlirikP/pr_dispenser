-- +goose Up

CREATE TABLE teams (
    id UUID PRIMARY KEY,
    teamname TEXT UNIQUE NOT NULL
);

CREATE TABLE users (
    id UUID PRIMARY KEY,
    username TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE
);

CREATE TABLE prs (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE RESTRICT,
    status TEXT NOT NULL CHECK (status IN ('OPEN', 'MERGED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    merged_at TIMESTAMPTZ NULL
);
CREATE TABLE pr_reviewers (
    pr_id UUID NOT NULL REFERENCES prs(id) ON DELETE CASCADE,
    reviewer_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    PRIMARY KEY (pr_id, reviewer_id)
);

-- +goose Down

DROP TABLE pr_reviewers;
DROP TABLE prs;
DROP TABLE users;
DROP TABLE teams;
