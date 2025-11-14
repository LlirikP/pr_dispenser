-- name: GetUserById :one
SELECT id, username, is_active, team_id
FROM users
WHERE id = $1;

-- name: SetUserIsActive :exec
UPDATE users
SET is_active = $2
WHERE id = $1;

-- name: GetReviewPRs :many
SELECT
    prs.id AS pr_id,
    prs.title AS pr_title,
    prs.author_id,
    prs.status
FROM pr_reviewers r
JOIN prs ON prs.id = r.pr_id
WHERE r.reviewer_id = $1
ORDER BY prs.id;
