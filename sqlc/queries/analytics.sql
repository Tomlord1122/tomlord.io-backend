-- name: GetTodayVisitCount :one
SELECT visit_count FROM daily_visits WHERE date = $1;

-- name: GetTotalVisitCount :one
SELECT COALESCE(SUM(visit_count), 0)::bigint AS total FROM daily_visits;

-- name: GetRecentVisitCounts :many
SELECT date, visit_count FROM daily_visits
WHERE date >= $1
ORDER BY date DESC;

-- name: UpsertDailyVisit :one
INSERT INTO daily_visits (date, visit_count) VALUES ($1, 1)
ON CONFLICT (date) DO UPDATE SET visit_count = daily_visits.visit_count + 1
RETURNING visit_count;

-- name: CheckVisitorHash :one
SELECT EXISTS(SELECT 1 FROM visitor_hashes WHERE date = $1 AND hash = $2) AS exists;

-- name: InsertVisitorHash :exec
INSERT INTO visitor_hashes (date, hash) VALUES ($1, $2);
