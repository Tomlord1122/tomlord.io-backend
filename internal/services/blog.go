package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"tomlord.io-backend/internal/cache"
	"tomlord.io-backend/internal/database"
	db "tomlord.io-backend/internal/db_sqlc"
)

const (
	cacheKeyBlogSlugPrefix  = "blog:slug:"
	cacheKeyBlogsListPrefix = "blogs:list:"
	cacheTTLBlog            = 10 * time.Minute
	cacheTTLBlogsList       = 5 * time.Minute
)

type BlogService struct {
	dbService database.DBService
	cache     *cache.MemoryCache
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
	Content     string   `json:"content,omitempty"`
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
	Content     string   `json:"content"`
	IsPublished bool     `json:"is_published"`
}

type UpdateBlogRequest struct {
	Title       string   `json:"title" binding:"required"`
	Date        string   `json:"date" binding:"required"` // Format: 2006-01-02
	Lang        string   `json:"lang"`
	Duration    string   `json:"duration"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
	Content     string   `json:"content"`
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
		cache:     cache.GetInstance(),
	}
}

// listCacheKey builds a cache key for list queries.
func listCacheKey(tag, lang string, published bool, limit, offset int32) string {
	return fmt.Sprintf("%s%s:%s:%v:%d:%d", cacheKeyBlogsListPrefix, tag, lang, published, limit, offset)
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

	// Convert content to pgtype.Text
	content := pgtype.Text{}
	if req.Content != "" {
		if err := content.Scan(req.Content); err != nil {
			return nil, fmt.Errorf("failed to scan content: %w", err)
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
		Content:     content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create blog: %w", err)
	}

	// Invalidate list caches since a new blog was added
	b.cache.DeletePrefix(cacheKeyBlogsListPrefix)

	return b.convertBlogMetadataToInfoFromRow(blog.ID, blog.Title, blog.Slug, blog.Date, blog.Lang, blog.Duration, blog.Tags, blog.Description, blog.IsPublished, blog.CreatedAt, blog.UpdatedAt), nil
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
	cacheKey := cacheKeyBlogSlugPrefix + slug
	if cached, ok := b.cache.Get(cacheKey); ok {
		if result, valid := cached.(*BlogWithMessageCount); valid {
			return result, nil
		}
	}

	queries := b.dbService.GetQueries()

	result, err := queries.GetBlogWithMessageCountBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to get blog with message count: %w", err)
	}

	blog := &BlogWithMessageCount{
		BlogInfo:     *b.convertBlogToInfoFromRow(result.ID, result.Title, result.Slug, result.Date, result.Lang, result.Duration, result.Tags, result.Description, result.Content, result.IsPublished, result.CreatedAt, result.UpdatedAt),
		MessageCount: result.MessageCount,
	}

	b.cache.Set(cacheKey, blog, cacheTTLBlog)
	return blog, nil
}

// ListBlogs retrieves a list of blogs based on criteria
func (b *BlogService) ListBlogs(ctx context.Context, req ListBlogsRequest) ([]BlogInfo, error) {
	if req.Limit <= 0 {
		req.Limit = 10000 // Default limit
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	cacheKey := listCacheKey(req.Tag, req.Lang, req.PublishedOnly, req.Limit, req.Offset)
	if cached, ok := b.cache.Get(cacheKey); ok {
		if result, valid := cached.([]BlogInfo); valid {
			return result, nil
		}
	}

	queries := b.dbService.GetQueries()

	var result []BlogInfo

	if req.Tag != "" {
		blogs, err := queries.GetBlogsByTag(ctx, db.GetBlogsByTagParams{
			Tags:   []string{req.Tag},
			Limit:  req.Limit,
			Offset: req.Offset,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get blogs: %w", err)
		}
		result = make([]BlogInfo, len(blogs))
		for i, blog := range blogs {
			result[i] = *b.convertBlogMetadataToInfoFromRow(blog.ID, blog.Title, blog.Slug, blog.Date, blog.Lang, blog.Duration, blog.Tags, blog.Description, blog.IsPublished, blog.CreatedAt, blog.UpdatedAt)
		}
	} else if req.Lang != "" {
		blogs, err := queries.GetBlogsByLang(ctx, db.GetBlogsByLangParams{
			Lang:   req.Lang,
			Limit:  req.Limit,
			Offset: req.Offset,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get blogs: %w", err)
		}
		result = make([]BlogInfo, len(blogs))
		for i, blog := range blogs {
			result[i] = *b.convertBlogMetadataToInfoFromRow(blog.ID, blog.Title, blog.Slug, blog.Date, blog.Lang, blog.Duration, blog.Tags, blog.Description, blog.IsPublished, blog.CreatedAt, blog.UpdatedAt)
		}
	} else if req.PublishedOnly {
		blogs, err := queries.GetBlogs(ctx, db.GetBlogsParams{
			Limit:  req.Limit,
			Offset: req.Offset,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get blogs: %w", err)
		}
		result = make([]BlogInfo, len(blogs))
		for i, blog := range blogs {
			result[i] = *b.convertBlogMetadataToInfoFromRow(blog.ID, blog.Title, blog.Slug, blog.Date, blog.Lang, blog.Duration, blog.Tags, blog.Description, blog.IsPublished, blog.CreatedAt, blog.UpdatedAt)
		}
	} else {
		blogs, err := queries.GetAllBlogs(ctx, db.GetAllBlogsParams{
			Limit:  req.Limit,
			Offset: req.Offset,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get blogs: %w", err)
		}
		result = make([]BlogInfo, len(blogs))
		for i, blog := range blogs {
			result[i] = *b.convertBlogMetadataToInfoFromRow(blog.ID, blog.Title, blog.Slug, blog.Date, blog.Lang, blog.Duration, blog.Tags, blog.Description, blog.IsPublished, blog.CreatedAt, blog.UpdatedAt)
		}
	}

	b.cache.Set(cacheKey, result, cacheTTLBlogsList)
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

	// Convert content to pgtype.Text
	content := pgtype.Text{}
	if req.Content != "" {
		if err := content.Scan(req.Content); err != nil {
			return nil, fmt.Errorf("failed to scan content: %w", err)
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
		Content:     content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update blog: %w", err)
	}

	// Invalidate caches for this blog and all list caches
	b.cache.Delete(cacheKeyBlogSlugPrefix + slug)
	b.cache.DeletePrefix(cacheKeyBlogsListPrefix)

	return b.convertBlogMetadataToInfoFromRow(blog.ID, blog.Title, blog.Slug, blog.Date, blog.Lang, blog.Duration, blog.Tags, blog.Description, blog.IsPublished, blog.CreatedAt, blog.UpdatedAt), nil
}

// DeleteBlogBySlug deletes a blog by its slug
func (b *BlogService) DeleteBlogBySlug(ctx context.Context, slug string) error {
	queries := b.dbService.GetQueries()

	err := queries.DeleteBlogBySlug(ctx, slug)
	if err != nil {
		return fmt.Errorf("failed to delete blog: %w", err)
	}

	// Invalidate caches for this blog and all list caches
	b.cache.Delete(cacheKeyBlogSlugPrefix + slug)
	b.cache.DeletePrefix(cacheKeyBlogsListPrefix)

	return nil
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
		Content:     blog.Content.String,
		IsPublished: blog.IsPublished.Bool,
		CreatedAt:   blog.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   blog.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (b *BlogService) convertBlogMetadataToInfoFromRow(id pgtype.UUID, title, slug string, date pgtype.Date, lang, duration string, tags []string, description pgtype.Text, isPublished pgtype.Bool, createdAt, updatedAt pgtype.Timestamptz) *BlogInfo {
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

// Helper function to convert individual fields to BlogInfo
func (b *BlogService) convertBlogToInfoFromRow(id pgtype.UUID, title, slug string, date pgtype.Date, lang, duration string, tags []string, description, content pgtype.Text, isPublished pgtype.Bool, createdAt, updatedAt pgtype.Timestamptz) *BlogInfo {
	return &BlogInfo{
		ID:          uuid.UUID(id.Bytes).String(),
		Title:       title,
		Slug:        slug,
		Date:        date.Time.Format("2006-01-02"),
		Lang:        lang,
		Duration:    duration,
		Tags:        tags,
		Description: description.String,
		Content:     content.String,
		IsPublished: isPublished.Bool,
		CreatedAt:   createdAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   updatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}
