-- name: CreatePR :exec
INSERT INTO prs (id, title, author_id, status)
VALUES ($1, $2, $3, 'OPEN');

-- name: GetPRById :one
SELECT id, title, author_id, status, created_at, merged_at
FROM prs
WHERE id = $1;

-- name: AddReviewer :exec
INSERT INTO pr_reviewers (pr_id, reviewer_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: DeleteReviewer :exec
DELETE FROM pr_reviewers
WHERE pr_id = $1 AND reviewer_id = $2;

-- name: GetReviewersByPR :many
SELECT reviewer_id
FROM pr_reviewers
WHERE pr_id = $1;

-- name: GetActiveTeamMembersExceptAuthor :many
SELECT id
FROM users
WHERE team_id = $1
  AND is_active = TRUE
  AND id <> $2;

-- name: IsReviewerAssigned :one
SELECT COUNT(*) > 0 AS assigned
FROM pr_reviewers
WHERE pr_id = $1 AND reviewer_id = $2;

-- name: MergePR :exec
UPDATE prs
SET status = 'MERGED',
    merged_at = NOW()
WHERE id = $1;

-- name: CheckDuplicatePR :one
SELECT id
FROM prs
WHERE author_id = $1
  AND title = $2
  AND status = 'OPEN'
LIMIT 1;
