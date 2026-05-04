package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
	"tomlord.io-backend/internal/cache"
)

const (
	cacheKeyPreviewPrefix = "preview:url:"
	cacheTTLPreview       = 1 * time.Hour
	maxBodySize           = 2 * 1024 * 1024 // 2 MB
	fetchTimeout          = 10 * time.Second
	maxRedirects          = 5
)

// LinkPreview holds the metadata extracted from a URL.
type LinkPreview struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	SiteName    string `json:"site_name"`
	Favicon     string `json:"favicon"`
}

// PreviewService fetches and caches OpenGraph metadata for URLs.
type PreviewService struct {
	cache  *cache.MemoryCache
	client *http.Client
}

// NewPreviewService creates a new PreviewService.
func NewPreviewService() *PreviewService {
	return &PreviewService{
		cache: cache.GetInstance(),
		client: &http.Client{
			Timeout: fetchTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= maxRedirects {
					return fmt.Errorf("too many redirects")
				}
				return validatePreviewURL(req.URL.String())
			},
		},
	}
}

// FetchPreview returns metadata for a URL, using the cache when available.
func (p *PreviewService) FetchPreview(ctx context.Context, urlStr string) (*LinkPreview, error) {
	if err := validatePreviewURL(urlStr); err != nil {
		return nil, err
	}

	cacheKey := previewCacheKey(urlStr)
	if cached, ok := p.cache.Get(cacheKey); ok {
		if preview, valid := cached.(*LinkPreview); valid {
			return preview, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "tomlord.io-preview-bot/1.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	preview, err := parsePreview(bytes.NewReader(body), urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse preview: %w", err)
	}

	p.cache.Set(cacheKey, preview, cacheTTLPreview)
	return preview, nil
}

// --- helpers ---

func previewCacheKey(urlStr string) string {
	h := sha256.Sum256([]byte(urlStr))
	return cacheKeyPreviewPrefix + hex.EncodeToString(h[:])
}

func validatePreviewURL(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid scheme: %s", u.Scheme)
	}

	hostname := u.Hostname()
	if hostname == "" {
		return fmt.Errorf("missing hostname")
	}

	// Block IP literals directly.
	if ip := net.ParseIP(hostname); ip != nil {
		if isPrivateIP(ip) {
			return fmt.Errorf("private IP not allowed")
		}
		return nil
	}

	// Block obvious internal hostnames.
	hostnameLower := strings.ToLower(hostname)
	if hostnameLower == "localhost" || strings.HasSuffix(hostnameLower, ".localhost") {
		return fmt.Errorf("localhost not allowed")
	}

	// Resolve hostname and ensure it doesn't point to private IPs.
	ips, err := net.LookupIP(hostname)
	if err != nil {
		// DNS failures are suppressed here so that broken external links don't
		// crash the preview endpoint; they will simply fail at fetch time.
		return nil
	}
	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("resolved to private IP")
		}
	}

	return nil
}

func isPrivateIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast()
}

func parsePreview(r io.Reader, baseURL string) (*LinkPreview, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	base, _ := url.Parse(baseURL)
	preview := &LinkPreview{URL: baseURL}

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type != html.ElementNode {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				traverse(c)
			}
			return
		}

		switch n.Data {
		case "meta":
			var property, content string
			for _, attr := range n.Attr {
				switch attr.Key {
				case "property", "name":
					property = attr.Val
				case "content":
					content = attr.Val
				}
			}
			switch property {
			case "og:title":
				if preview.Title == "" {
					preview.Title = content
				}
			case "og:description":
				if preview.Description == "" {
					preview.Description = content
				}
			case "og:image":
				if preview.Image == "" {
					preview.Image = resolveURL(base, content)
				}
			case "og:site_name":
				if preview.SiteName == "" {
					preview.SiteName = content
				}
			}
		case "title":
			if preview.Title == "" && n.FirstChild != nil {
				preview.Title = strings.TrimSpace(n.FirstChild.Data)
			}
		case "link":
			var rel, href string
			for _, attr := range n.Attr {
				switch attr.Key {
				case "rel":
					rel = attr.Val
				case "href":
					href = attr.Val
				}
			}
			if preview.Favicon == "" && (rel == "icon" || rel == "shortcut icon") {
				preview.Favicon = resolveURL(base, href)
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(doc)

	if preview.Favicon == "" && base != nil {
		preview.Favicon = base.Scheme + "://" + base.Host + "/favicon.ico"
	}
	if preview.SiteName == "" && base != nil {
		preview.SiteName = base.Hostname()
	}

	return preview, nil
}

func resolveURL(base *url.URL, ref string) string {
	if ref == "" {
		return ""
	}
	u, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	if base != nil {
		return base.ResolveReference(u).String()
	}
	return ref
}
