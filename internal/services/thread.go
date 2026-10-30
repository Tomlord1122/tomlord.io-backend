package services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"tomlord.io-backend/internal/cache"
	"tomlord.io-backend/internal/database"
	db "tomlord.io-backend/internal/db_sqlc"
)

const (
	threadDefaultLimit = int32(20)
	threadMaxLimit     = int32(50)
	threadMaxRunes     = 2000

	cacheKeyThreadIDPrefix    = "thread:id:"
	cacheKeyThreadsListPrefix = "threads:list:"
	cacheTTLThread            = 10 * time.Minute
	cacheTTLThreadsList       = 5 * time.Minute
)

var (
	ErrThreadNotFound = errors.New("thread not found")
	ErrThreadInvalid  = errors.New("invalid thread")
	ErrThreadCursor   = errors.New("invalid cursor")
)

type ThreadService struct {
	dbService database.DBService
	cache     *cache.MemoryCache
}

type ThreadInfo struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type ThreadPage struct {
	Threads    []ThreadInfo `json:"threads"`
	NextCursor string       `json:"next_cursor"`
	HasMore    bool         `json:"has_more"`
}

type CreateThreadRequest struct {
	Content string `json:"content" binding:"required"`
}

type UpdateThreadRequest struct {
	Content string `json:"content" binding:"required"`
}

type ListThreadsRequest struct {
	Limit  int32
	Cursor string
}

func NewThreadService(dbService database.DBService) *ThreadService {
	return &ThreadService{dbService: dbService, cache: cache.GetInstance()}
}

func (t *ThreadService) CreateThread(ctx context.Context, req CreateThreadRequest) (*ThreadInfo, error) {
	content, err := normalizeThreadContent(req.Content)
	if err != nil {
		return nil, err
	}

	row, err := t.dbService.GetQueries().CreateThread(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("failed to create thread: %w", err)
	}
	info := threadFromRow(row)
	t.invalidateThreads()
	return &info, nil
}

func (t *ThreadService) GetThread(ctx context.Context, id string) (*ThreadInfo, error) {
	threadID, err := parseThreadID(id)
	if err != nil {
		return nil, err
	}

	cacheKey := cacheKeyThreadIDPrefix + id
	if cached, ok := t.cache.Get(cacheKey); ok {
		info := cached.(ThreadInfo)
		return &info, nil
	}

	row, err := t.dbService.GetQueries().GetThreadByID(ctx, threadID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrThreadNotFound
		}
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}
	info := threadFromRow(row)
	t.cache.Set(cacheKey, info, cacheTTLThread)
	return &info, nil
}

func (t *ThreadService) ListThreads(ctx context.Context, req ListThreadsRequest) (*ThreadPage, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = threadDefaultLimit
	}
	if limit > threadMaxLimit {
		limit = threadMaxLimit
	}

	cacheKey := fmt.Sprintf("%s%d:%s", cacheKeyThreadsListPrefix, limit, req.Cursor)
	if cached, ok := t.cache.Get(cacheKey); ok {
		page := cached.(ThreadPage)
		return &page, nil
	}

	params := db.ListThreadsParams{Limit: limit + 1}
	if req.Cursor != "" {
		createdAt, id, err := decodeThreadCursor(req.Cursor)
		if err != nil {
			return nil, err
		}
		params.CursorCreatedAt = pgtype.Timestamptz{Time: createdAt, Valid: true}
		params.CursorID = id
	}

	rows, err := t.dbService.GetQueries().ListThreads(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list threads: %w", err)
	}

	hasMore := int32(len(rows)) > limit
	if hasMore {
		rows = rows[:limit]
	}

	threads := make([]ThreadInfo, len(rows))
	for i, row := range rows {
		threads[i] = threadFromRow(row)
	}

	page := &ThreadPage{Threads: threads, HasMore: hasMore}
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		page.NextCursor = encodeThreadCursor(last.CreatedAt.Time.UTC(), uuid.UUID(last.ID.Bytes))
	}
	t.cache.Set(cacheKey, *page, cacheTTLThreadsList)
	return page, nil
}

func (t *ThreadService) UpdateThread(ctx context.Context, id string, req UpdateThreadRequest) (*ThreadInfo, error) {
	threadID, err := parseThreadID(id)
	if err != nil {
		return nil, err
	}
	content, err := normalizeThreadContent(req.Content)
	if err != nil {
		return nil, err
	}

	row, err := t.dbService.GetQueries().UpdateThread(ctx, db.UpdateThreadParams{
		ID:      threadID,
		Content: content,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrThreadNotFound
		}
		return nil, fmt.Errorf("failed to update thread: %w", err)
	}
	info := threadFromRow(row)
	t.invalidateThreads()
	return &info, nil
}

func (t *ThreadService) DeleteThread(ctx context.Context, id string) error {
	threadID, err := parseThreadID(id)
	if err != nil {
		return err
	}
	if err := t.dbService.GetQueries().DeleteThread(ctx, threadID); err != nil {
		return fmt.Errorf("failed to delete thread: %w", err)
	}
	t.invalidateThreads()
	return nil
}

func (t *ThreadService) invalidateThreads() {
	t.cache.DeletePrefix(cacheKeyThreadIDPrefix)
	t.cache.DeletePrefix(cacheKeyThreadsListPrefix)
}

func normalizeThreadContent(content string) (string, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "", fmt.Errorf("%w: content is required", ErrThreadInvalid)
	}
	if utf8.RuneCountInString(trimmed) > threadMaxRunes {
		return "", fmt.Errorf("%w: content must be %d characters or fewer", ErrThreadInvalid, threadMaxRunes)
	}
	return trimmed, nil
}

func parseThreadID(id string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("%w: invalid id", ErrThreadInvalid)
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

func threadFromRow(row db.Thread) ThreadInfo {
	return ThreadInfo{
		ID:        uuid.UUID(row.ID.Bytes).String(),
		Content:   row.Content,
		CreatedAt: row.CreatedAt.Time.UTC().Format(time.RFC3339Nano),
		UpdatedAt: row.UpdatedAt.Time.UTC().Format(time.RFC3339Nano),
	}
}

func encodeThreadCursor(createdAt time.Time, id uuid.UUID) string {
	raw := createdAt.UTC().Format(time.RFC3339Nano) + "|" + id.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeThreadCursor(cursor string) (time.Time, pgtype.UUID, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, pgtype.UUID{}, fmt.Errorf("%w: malformed cursor", ErrThreadCursor)
	}
	createdAtRaw, idRaw, ok := strings.Cut(string(decoded), "|")
	if !ok {
		return time.Time{}, pgtype.UUID{}, fmt.Errorf("%w: malformed cursor", ErrThreadCursor)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return time.Time{}, pgtype.UUID{}, fmt.Errorf("%w: malformed cursor", ErrThreadCursor)
	}
	id, err := parseThreadID(idRaw)
	if err != nil {
		return time.Time{}, pgtype.UUID{}, fmt.Errorf("%w: malformed cursor", ErrThreadCursor)
	}
	return createdAt.UTC(), id, nil
}
