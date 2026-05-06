package services

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"tomlord.io-backend/internal/cache"
	"tomlord.io-backend/internal/database"
	db "tomlord.io-backend/internal/db_sqlc"
)

const (
	cacheKeyVisitorStats = "visitors:stats"
	cacheTTLVisitorStats = 1 * time.Minute
)

type VisitStats struct {
	TodayCount int64   `json:"today_count"`
	TotalCount int64   `json:"total_count"`
	IsNewVisit bool    `json:"is_new_visit"`
	Recent     []DayStat `json:"recent"`
}

type DayStat struct {
	Date       string `json:"date"`
	VisitCount int32  `json:"visit_count"`
}

type AnalyticsService struct {
	dbService database.DBService
	cache     *cache.MemoryCache
}

func NewAnalyticsService(dbService database.DBService) *AnalyticsService {
	return &AnalyticsService{
		dbService: dbService,
		cache:     cache.GetInstance(),
	}
}

// RecordVisit checks if this visitor (identified by a stable client key) has been counted today.
// If not, it records the hash and increments the daily visit count.
func (s *AnalyticsService) RecordVisit(ctx context.Context, visitorKey string) (*VisitStats, error) {
	date := time.Now().UTC().Truncate(24 * time.Hour)
	pgDate := pgtype.Date{Time: date, Valid: true}

	// Store only a hash of the stable visitor key.
	hash := s.hashVisitor(visitorKey)

	var stats VisitStats
	err := s.dbService.RunTransaction(ctx, func(q *db.Queries) error {
		// Check if this visitor was already counted today
		exists, err := q.CheckVisitorHash(ctx, db.CheckVisitorHashParams{
			Date: pgDate,
			Hash: hash,
		})
		if err != nil {
			return fmt.Errorf("failed to check visitor hash: %w", err)
		}

		if exists {
			// Already counted today, just get stats
			stats.IsNewVisit = false
		} else {
			// New visitor for today
			err = q.InsertVisitorHash(ctx, db.InsertVisitorHashParams{
				Date: pgDate,
				Hash: hash,
			})
			if err != nil {
				return fmt.Errorf("failed to insert visitor hash: %w", err)
			}

			_, err = q.UpsertDailyVisit(ctx, pgDate)
			if err != nil {
				return fmt.Errorf("failed to upsert daily visit: %w", err)
			}
			stats.IsNewVisit = true
		}

		// Get today's count
		todayCount, err := q.GetTodayVisitCount(ctx, pgDate)
		if err != nil {
			// No row for today yet (shouldn't happen after upsert, but handle gracefully)
			todayCount = 0
		}
		stats.TodayCount = int64(todayCount)

		// Get total count
		totalCount, err := q.GetTotalVisitCount(ctx)
		if err != nil {
			return fmt.Errorf("failed to get total visit count: %w", err)
		}
		stats.TotalCount = totalCount

		// Get last 7 days for sparkline
		recent, err := q.GetRecentVisitCounts(ctx, pgtype.Date{Time: date.AddDate(0, 0, -6), Valid: true})
		if err != nil {
			return fmt.Errorf("failed to get recent visit counts: %w", err)
		}

		stats.Recent = buildRecentDayStats(date, recent)

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Invalidate cache since stats changed (if it was a new visit)
	if stats.IsNewVisit {
		s.cache.Delete(cacheKeyVisitorStats)
	}

	return &stats, nil
}

// GetVisitorStats returns current visitor stats without recording a new visit.
func (s *AnalyticsService) GetVisitorStats(ctx context.Context) (*VisitStats, error) {
	// Try cache first
	if cached, ok := s.cache.Get(cacheKeyVisitorStats); ok {
		if result, valid := cached.(*VisitStats); valid {
			return result, nil
		}
	}

	date := time.Now().UTC().Truncate(24 * time.Hour)
	pgDate := pgtype.Date{Time: date, Valid: true}

	queries := s.dbService.GetQueries()

	var stats VisitStats
	var err error

	todayCount, err := queries.GetTodayVisitCount(ctx, pgDate)
	if err != nil {
		stats.TodayCount = 0
	} else {
		stats.TodayCount = int64(todayCount)
	}

	stats.TotalCount, err = queries.GetTotalVisitCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get total visit count: %w", err)
	}

	// Get last 7 days for sparkline
	recent, err := queries.GetRecentVisitCounts(ctx, pgtype.Date{Time: date.AddDate(0, 0, -6), Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get recent visit counts: %w", err)
	}

	stats.Recent = buildRecentDayStats(date, recent)

	// Cache the result
	s.cache.Set(cacheKeyVisitorStats, &stats, cacheTTLVisitorStats)

	return &stats, nil
}

func (s *AnalyticsService) hashVisitor(visitorKey string) string {
	h := sha256.New()
	h.Write([]byte(visitorKey))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func buildRecentDayStats(today time.Time, rows []db.DailyVisit) []DayStat {
	countsByDate := make(map[string]int32, len(rows))
	for _, row := range rows {
		countsByDate[row.Date.Time.Format("2006-01-02")] = row.VisitCount
	}

	stats := make([]DayStat, 7)
	for i := 0; i < 7; i++ {
		day := today.AddDate(0, 0, -i)
		date := day.Format("2006-01-02")
		stats[i] = DayStat{
			Date:       date,
			VisitCount: countsByDate[date],
		}
	}

	return stats
}
