package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"tomlord.io-backend/internal/database"
	db "tomlord.io-backend/internal/db_sqlc"
)

type BlogService struct {
	dbService database.DBService
}

type BlogInfo struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Slug        string   `json:"slug"`
	Date        string   `json:"date"`
	Lang        string   `json:"lang"`
	Duration    string   `json:"duration"`
	Tags        []string `json:"tags"`
	Description string   `json:"description,omitempty"`
	IsPublished bool     `json:"is_published"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type BlogWithMessageCount struct {
	BlogInfo
	MessageCount int64 `json:"message_count"`
}

type CreateBlogRequest struct {
	Title       string   `json:"title" binding:"required"`
	Slug        string   `json:"slug" binding:"required"`
	Date        string   `json:"date" binding:"required"` // Format: 2006-01-02
	Lang        string   `json:"lang"`
	Duration    string   `json:"duration"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
	IsPublished bool     `json:"is_published"`
}

type UpdateBlogRequest struct {
	Title       string   `json:"title" binding:"required"`
	Date        string   `json:"date" binding:"required"` // Format: 2006-01-02
	Lang        string   `json:"lang"`
	Duration    string   `json:"duration"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
	IsPublished bool     `json:"is_published"`
}

type ListBlogsRequest struct {
	Limit         int32  `json:"limit"`
	Offset        int32  `json:"offset"`
	Tag           string `json:"tag,omitempty"`
	Lang          string `json:"lang,omitempty"`
	PublishedOnly bool   `json:"published_only"`
}

// NewBlogService creates a new blog service
func NewBlogService(dbService database.DBService) *BlogService {
	return &BlogService{
		dbService: dbService,
	}
}

// CreateBlog creates a new blog entry
func (b *BlogService) CreateBlog(ctx context.Context, req CreateBlogRequest) (*BlogInfo, error) {
	queries := b.dbService.GetQueries()

	// Parse date
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	// Set defaults
	if req.Lang == "" {
		req.Lang = "zh-tw"
	}
	if req.Duration == "" {
		req.Duration = "5min"
	}

	// Convert description to pgtype.Text
	description := pgtype.Text{}
	if req.Description != "" {
		if err := description.Scan(req.Description); err != nil {
			return nil, fmt.Errorf("failed to scan description: %w", err)
		}
	}

	blog, err := queries.CreateBlog(ctx, db.CreateBlogParams{
		Title:       req.Title,
		Slug:        req.Slug,
		Date:        pgtype.Date{Time: date, Valid: true},
		Lang:        req.Lang,
		Duration:    req.Duration,
		Tags:        req.Tags,
		Description: description,
		IsPublished: pgtype.Bool{Bool: req.IsPublished, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create blog: %w", err)
	}

	return b.convertBlogToInfo(blog), nil
}

// GetBlogBySlug retrieves a blog by its slug
func (b *BlogService) GetBlogBySlug(ctx context.Context, slug string) (*BlogInfo, error) {
	queries := b.dbService.GetQueries()

	blog, err := queries.GetBlogBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to get blog: %w", err)
	}

	return b.convertBlogToInfo(blog), nil
}

// GetBlogWithMessageCountBySlug retrieves a blog with its message count
func (b *BlogService) GetBlogWithMessageCountBySlug(ctx context.Context, slug string) (*BlogWithMessageCount, error) {
	queries := b.dbService.GetQueries()

	result, err := queries.GetBlogWithMessageCountBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to get blog with message count: %w", err)
	}

	return &BlogWithMessageCount{
		BlogInfo:     *b.convertBlogToInfoFromRow(result.ID, result.Title, result.Slug, result.Date, result.Lang, result.Duration, result.Tags, result.Description, result.IsPublished, result.CreatedAt, result.UpdatedAt),
		MessageCount: result.MessageCount,
	}, nil
}

// ListBlogs retrieves a list of blogs based on criteria
func (b *BlogService) ListBlogs(ctx context.Context, req ListBlogsRequest) ([]BlogInfo, error) {
	queries := b.dbService.GetQueries()

	if req.Limit <= 0 {
		req.Limit = 10000 // Default limit
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	var blogs []db.Blog
	var err error

	if req.Tag != "" {
		blogs, err = queries.GetBlogsByTag(ctx, db.GetBlogsByTagParams{
			Tags:   []string{req.Tag},
			Limit:  req.Limit,
			Offset: req.Offset,
		})
	} else if req.Lang != "" {
		blogs, err = queries.GetBlogsByLang(ctx, db.GetBlogsByLangParams{
			Lang:   req.Lang,
			Limit:  req.Limit,
			Offset: req.Offset,
		})
	} else if req.PublishedOnly {
		blogs, err = queries.GetBlogs(ctx, db.GetBlogsParams{
			Limit:  req.Limit,
			Offset: req.Offset,
		})
	} else {
		blogs, err = queries.GetAllBlogs(ctx, db.GetAllBlogsParams{
			Limit:  req.Limit,
			Offset: req.Offset,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get blogs: %w", err)
	}

	result := make([]BlogInfo, len(blogs))
	for i, blog := range blogs {
		result[i] = *b.convertBlogToInfo(blog)
	}

	return result, nil
}

// UpdateBlogBySlug updates a blog by its slug
func (b *BlogService) UpdateBlogBySlug(ctx context.Context, slug string, req UpdateBlogRequest) (*BlogInfo, error) {
	queries := b.dbService.GetQueries()

	// Parse date
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	// Set defaults
	if req.Lang == "" {
		req.Lang = "zh-tw"
	}
	if req.Duration == "" {
		req.Duration = "5min"
	}

	// Convert description to pgtype.Text
	description := pgtype.Text{}
	if req.Description != "" {
		if err := description.Scan(req.Description); err != nil {
			return nil, fmt.Errorf("failed to scan description: %w", err)
		}
	}

	blog, err := queries.UpdateBlogBySlug(ctx, db.UpdateBlogBySlugParams{
		Slug:        slug,
		Title:       req.Title,
		Date:        pgtype.Date{Time: date, Valid: true},
		Lang:        req.Lang,
		Duration:    req.Duration,
		Tags:        req.Tags,
		Description: description,
		IsPublished: pgtype.Bool{Bool: req.IsPublished, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update blog: %w", err)
	}

	return b.convertBlogToInfo(blog), nil
}

// Helper function to convert db.Blog to BlogInfo
func (b *BlogService) convertBlogToInfo(blog db.Blog) *BlogInfo {
	return &BlogInfo{
		ID:          uuid.UUID(blog.ID.Bytes).String(),
		Title:       blog.Title,
		Slug:        blog.Slug,
		Date:        blog.Date.Time.Format("2006-01-02"),
		Lang:        blog.Lang,
		Duration:    blog.Duration,
		Tags:        blog.Tags,
		Description: blog.Description.String,
		IsPublished: blog.IsPublished.Bool,
		CreatedAt:   blog.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   blog.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// Helper function to convert individual fields to BlogInfo
func (b *BlogService) convertBlogToInfoFromRow(id pgtype.UUID, title, slug string, date pgtype.Date, lang, duration string, tags []string, description pgtype.Text, isPublished pgtype.Bool, createdAt, updatedAt pgtype.Timestamptz) *BlogInfo {
	return &BlogInfo{
		ID:          uuid.UUID(id.Bytes).String(),
		Title:       title,
		Slug:        slug,
		Date:        date.Time.Format("2006-01-02"),
		Lang:        lang,
		Duration:    duration,
		Tags:        tags,
		Description: description.String,
		IsPublished: isPublished.Bool,
		CreatedAt:   createdAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   updatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}
