-- name: CreateTeam :exec
INSERT INTO teams (id, teamname)
VALUES ($1, $2);

-- name: GetTeamByName :one
SELECT id, teamname
FROM teams
WHERE teamname = $1;

-- name: UpsertUser :exec
INSERT INTO users (id, username, is_active, team_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE
SET username = EXCLUDED.username,
    is_active = EXCLUDED.is_active,
    team_id = EXCLUDED.team_id;

-- name: GetUsersByTeam :many
SELECT id, username, is_active, team_id
FROM users
WHERE team_id = $1
ORDER BY username;
