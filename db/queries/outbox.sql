-- name: InsertOutbox :exec
INSERT INTO outbox (topic, payload) VALUES ($1, $2);

-- name: ClaimOutbox :many
-- FOR UPDATE SKIP LOCKED: a second worker skips rows the first one holds instead of waiting or sending twice.
SELECT id, topic, payload, attempts FROM outbox
WHERE sent_at IS NULL AND attempts < $1 AND next_attempt_at <= now()
ORDER BY id
LIMIT $2
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxSent :exec
UPDATE outbox SET sent_at = now(), attempts = attempts + 1, last_error = NULL WHERE id = $1;

-- name: MarkOutboxFailed :exec
UPDATE outbox
SET attempts = attempts + 1,
    last_error = $2,
    next_attempt_at = now() + make_interval(secs => @backoff_secs::float8)
WHERE id = $1;
