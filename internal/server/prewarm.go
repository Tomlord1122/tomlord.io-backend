package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/viper"

	"tomlord.io-backend/internal/services"
)

type isrPrewarmConfig struct {
	frontendURL    string
	token          string
	staticInterval time.Duration
	blogInterval   time.Duration
	requestJitter  time.Duration
	maxBlogs       int32
}

var staticISRPaths = []string{"/", "/project", "/blog"}

func (s *Server) startISRPrewarmer(ctx context.Context) {
	if !viper.GetBool("PREWARM_ENABLED") {
		return
	}

	cfg := isrPrewarmConfig{
		frontendURL:    strings.TrimRight(viper.GetString("PREWARM_FRONTEND_URL"), "/"),
		token:          viper.GetString("PREWARM_TOKEN"),
		staticInterval: viper.GetDuration("PREWARM_STATIC_INTERVAL"),
		blogInterval:   viper.GetDuration("PREWARM_BLOG_INTERVAL"),
		requestJitter:  viper.GetDuration("PREWARM_REQUEST_JITTER"),
		maxBlogs:       int32(viper.GetInt("PREWARM_MAX_BLOGS")),
	}

	if cfg.frontendURL == "" {
		cfg.frontendURL = strings.TrimRight(viper.GetString("FRONTEND_URL"), "/")
	}
	if cfg.token == "" {
		log.Println("ISR prewarmer disabled: PREWARM_TOKEN is empty")
		return
	}
	if cfg.staticInterval <= 0 {
		cfg.staticInterval = 10 * time.Minute
	}
	if cfg.blogInterval <= 0 {
		cfg.blogInterval = time.Hour
	}
	if cfg.requestJitter < 0 {
		cfg.requestJitter = 0
	}
	if cfg.maxBlogs <= 0 {
		cfg.maxBlogs = 50
	}

	go s.runISRPrewarmer(ctx, cfg)
}

func (s *Server) runISRPrewarmer(ctx context.Context, cfg isrPrewarmConfig) {
	client := &http.Client{Timeout: 10 * time.Second}
	staticTicker := time.NewTicker(cfg.staticInterval)
	blogTicker := time.NewTicker(cfg.blogInterval)
	defer staticTicker.Stop()
	defer blogTicker.Stop()

	log.Printf("ISR prewarmer started: frontend=%s static_interval=%s blog_interval=%s request_jitter=%s max_blogs=%d", cfg.frontendURL, cfg.staticInterval, cfg.blogInterval, cfg.requestJitter, cfg.maxBlogs)

	s.prewarmStaticISRPaths(ctx, client, cfg)
	s.prewarmBlogISRPaths(ctx, client, cfg)

	for {
		select {
		case <-ctx.Done():
			log.Println("ISR prewarmer stopped")
			return
		case <-staticTicker.C:
			s.prewarmStaticISRPaths(ctx, client, cfg)
		case <-blogTicker.C:
			s.prewarmBlogISRPaths(ctx, client, cfg)
		}
	}
}

func (s *Server) prewarmStaticISRPaths(ctx context.Context, client *http.Client, cfg isrPrewarmConfig) {
	for i, path := range staticISRPaths {
		if i > 0 && !waitForPrewarmJitter(ctx, cfg.requestJitter) {
			return
		}
		if err := prewarmISRPath(ctx, client, cfg, path); err != nil {
			log.Printf("ISR prewarm failed for %s: %v", path, err)
		}
	}
}

func (s *Server) prewarmBlogISRPaths(ctx context.Context, client *http.Client, cfg isrPrewarmConfig) {
	listCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	blogs, err := s.blogService.ListBlogs(listCtx, services.ListBlogsRequest{
		Limit:         cfg.maxBlogs,
		Offset:        0,
		PublishedOnly: true,
	})
	if err != nil {
		log.Printf("ISR prewarm failed to list blogs: %v", err)
		return
	}

	for i, blog := range blogs {
		if i > 0 && !waitForPrewarmJitter(ctx, cfg.requestJitter) {
			return
		}
		path := "/blog/" + url.PathEscape(blog.Slug)
		if err := prewarmISRPath(ctx, client, cfg, path); err != nil {
			log.Printf("ISR prewarm failed for %s: %v", path, err)
		}
	}
}

func prewarmISRPath(ctx context.Context, client *http.Client, cfg isrPrewarmConfig, path string) error {
	endpoint, err := url.Parse(cfg.frontendURL + "/api/revalidate")
	if err != nil {
		return fmt.Errorf("invalid revalidate URL: %w", err)
	}

	query := endpoint.Query()
	query.Set("path", path)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.token)

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("unexpected status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func waitForPrewarmJitter(ctx context.Context, maxJitter time.Duration) bool {
	if maxJitter <= 0 {
		return true
	}

	timer := time.NewTimer(time.Duration(rand.Int63n(int64(maxJitter))))
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
