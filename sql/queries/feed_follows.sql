-- name: CreateFeedFollow :one
WITH inserted_feed_follow AS (
    INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
    VALUES ($1, $2, $3, $4, $5)
    RETURNING *
)
SELECT
    f.*,
    u.name AS user_name,
    fe.name AS feed_name
FROM inserted_feed_follow f
INNER JOIN users u ON f.user_id = u.id
INNER JOIN feeds fe ON f.feed_id = fe.id;

-- name: GetFeedFollowsForUser :many
SELECT
    ff.*,
    u.name AS user_name,
    f.name AS feed_name
FROM feed_follows ff 
INNER JOIN users u ON ff.user_id = u.id 
INNER JOIN feeds f ON ff.feed_id = f.id 
WHERE ff.user_id = $1;

-- name: DelFeedFollow :exec
DELETE FROM feed_follows
USING feeds
WHERE feed_follows.user_id = $1
    AND feed_follows.feed_id = feeds.id 
    AND feeds.url = $2;
