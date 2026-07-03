package service

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"ppk/backend/internal/model"
)

const (
	platformLinkOpenModeOfficial = "official_link"
	platformLinkOpenModeApp      = "app_link"
)

type LandingPlatformLink struct {
	ID           uint   `json:"id"`
	PlatformCode string `json:"platformCode"`
	PlatformName string `json:"platformName"`
	ButtonText   string `json:"buttonText"`
	TargetURL    string `json:"targetUrl"`
	BackupURL    string `json:"backupUrl"`
	SortNo       int    `json:"sortNo"`
	Status       int    `json:"status"`
	OpenMode     string `json:"openMode"`
	OpenURL      string `json:"openUrl"`
}

type landingPlatformLinkOpen struct {
	OpenMode string
	OpenURL  string
}

type redirectURLResolver interface {
	Resolve(ctx context.Context, rawURL string) (string, error)
}

type httpRedirectURLResolver struct {
	client *http.Client
}

var defaultRedirectURLResolver redirectURLResolver = newHTTPRedirectURLResolver()

func newHTTPRedirectURLResolver() redirectURLResolver {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	return &httpRedirectURLResolver{
		client: &http.Client{
			Timeout:   3 * time.Second,
			Transport: transport,
		},
	}
}

func (r *httpRedirectURLResolver) Resolve(ctx context.Context, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
	if resp.Request == nil || resp.Request.URL == nil {
		return "", nil
	}
	return resp.Request.URL.String(), nil
}

func buildLandingPlatformLinks(links []model.StorePlatformLink, resolver redirectURLResolver) []LandingPlatformLink {
	if len(links) == 0 {
		return nil
	}
	out := make([]LandingPlatformLink, len(links))
	var wg sync.WaitGroup
	wg.Add(len(links))
	for i, link := range links {
		i := i
		link := link
		go func() {
			defer wg.Done()
			out[i] = landingPlatformLinkFromModel(link, resolver)
		}()
	}
	wg.Wait()
	return out
}

func landingPlatformLinkFromModel(link model.StorePlatformLink, resolver redirectURLResolver) LandingPlatformLink {
	resolvedURL, resolveErr := resolvePlatformLinkTargetURL(strings.TrimSpace(link.TargetURL), resolver)
	open := decideLandingPlatformLinkOpen(link, resolvedURL, resolveErr)
	return LandingPlatformLink{
		ID:           link.ID,
		PlatformCode: link.PlatformCode,
		PlatformName: link.PlatformName,
		ButtonText:   link.ButtonText,
		TargetURL:    link.TargetURL,
		BackupURL:    link.BackupURL,
		SortNo:       link.SortNo,
		Status:       link.Status,
		OpenMode:     open.OpenMode,
		OpenURL:      open.OpenURL,
	}
}

func resolvePlatformLinkTargetURL(targetURL string, resolver redirectURLResolver) (string, error) {
	targetURL = strings.TrimSpace(targetURL)
	if targetURL == "" || resolver == nil {
		return targetURL, nil
	}
	lower := strings.ToLower(targetURL)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return targetURL, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return resolver.Resolve(ctx, targetURL)
}

func decideLandingPlatformLinkOpen(link model.StorePlatformLink, resolvedURL string, resolveErr error) landingPlatformLinkOpen {
	targetURL := strings.TrimSpace(link.TargetURL)
	backupURL := strings.TrimSpace(link.BackupURL)
	if targetURL == "" {
		return landingPlatformLinkOpen{
			OpenMode: platformLinkOpenModeApp,
			OpenURL:  backupURL,
		}
	}
	if resolveErr != nil || strings.TrimSpace(resolvedURL) == "" {
		return landingPlatformLinkOpen{
			OpenMode: platformLinkOpenModeOfficial,
			OpenURL:  targetURL,
		}
	}
	if redirectURLHasParam(resolvedURL, "poiId") {
		return landingPlatformLinkOpen{
			OpenMode: platformLinkOpenModeOfficial,
			OpenURL:  targetURL,
		}
	}
	return landingPlatformLinkOpen{
		OpenMode: platformLinkOpenModeApp,
		OpenURL:  firstNonEmpty(strings.TrimSpace(resolvedURL), backupURL, targetURL),
	}
}

func redirectURLHasParam(rawURL string, param string) bool {
	param = strings.TrimSpace(param)
	if param == "" {
		return false
	}
	queue := []string{strings.TrimSpace(rawURL)}
	seen := map[string]bool{}
	for len(queue) > 0 {
		current := strings.TrimSpace(queue[0])
		queue = queue[1:]
		if current == "" || seen[current] {
			continue
		}
		seen[current] = true

		if queryHasParam(current, param) {
			return true
		}

		parsed, err := url.Parse(current)
		if err == nil {
			for key, values := range parsed.Query() {
				if strings.EqualFold(strings.TrimSpace(key), param) {
					return true
				}
				for _, value := range values {
					queue = appendDecoded(queue, value)
				}
			}
			if parsed.Fragment != "" {
				queue = appendDecoded(queue, parsed.Fragment)
			}
		}

		queue = appendDecoded(queue, current)
	}
	return false
}

func queryHasParam(raw string, param string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	query := raw
	if idx := strings.Index(query, "?"); idx >= 0 {
		query = query[idx+1:]
	}
	query = strings.TrimPrefix(query, "?")
	values, err := url.ParseQuery(query)
	if err != nil {
		return false
	}
	for key := range values {
		if strings.EqualFold(strings.TrimSpace(key), param) {
			return true
		}
	}
	return false
}

func appendDecoded(queue []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return queue
	}
	queue = append(queue, value)
	if decoded, err := url.QueryUnescape(value); err == nil && decoded != value {
		queue = append(queue, decoded)
	}
	return queue
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
