package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"tomlord.io-backend/internal/database"
	db "tomlord.io-backend/internal/db_sqlc"
)

type PageService struct {
	dbService database.DBService
}

type PageInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type UpsertPageRequest struct {
	Name    string `json:"name" binding:"required"`
	Content string `json:"content" binding:"required"`
}

type ListPagesRequest struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

// NewPageService creates a new page service
func NewPageService(dbService database.DBService) *PageService {
	return &PageService{
		dbService: dbService,
	}
}

// GetPageByName retrieves a page by its name
func (p *PageService) GetPageByName(ctx context.Context, name string) (*PageInfo, error) {
	queries := p.dbService.GetQueries()

	page, err := queries.GetPageByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get page: %w", err)
	}

	return p.convertPageToInfo(page), nil
}

// UpsertPage creates or updates a page
func (p *PageService) UpsertPage(ctx context.Context, req UpsertPageRequest) (*PageInfo, error) {
	queries := p.dbService.GetQueries()

	page, err := queries.UpsertPage(ctx, db.UpsertPageParams{
		Name:    req.Name,
		Content: req.Content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upsert page: %w", err)
	}

	return p.convertPageToInfo(page), nil
}

// ListPages retrieves a list of pages
func (p *PageService) ListPages(ctx context.Context, req ListPagesRequest) ([]PageInfo, error) {
	queries := p.dbService.GetQueries()

	if req.Limit <= 0 {
		req.Limit = 100 // Default limit
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	pages, err := queries.ListPages(ctx, db.ListPagesParams{
		Limit:  req.Limit,
		Offset: req.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pages: %w", err)
	}

	result := make([]PageInfo, len(pages))
	for i, page := range pages {
		result[i] = *p.convertPageToInfo(page)
	}

	return result, nil
}

// Helper function to convert db.Page to PageInfo
func (p *PageService) convertPageToInfo(page db.Page) *PageInfo {
	return &PageInfo{
		ID:        uuid.UUID(page.ID.Bytes).String(),
		Name:      page.Name,
		Content:   page.Content,
		CreatedAt: page.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: page.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}
