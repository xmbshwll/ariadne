package youtubemusic

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

func (a *Adapter) FetchAlbum(ctx context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	if parsed.Service != model.ServiceYouTubeMusic {
		return nil, fmt.Errorf("%w: %s", errUnexpectedYouTubeMusicService, parsed.Service)
	}
	body, err := a.fetchPage(ctx, parsed.CanonicalURL)
	if err != nil {
		return nil, fmt.Errorf("fetch youtube music page: %w", err)
	}
	return extractAlbum(body, parsed.CanonicalURL)
}

func (a *Adapter) fetchAlbumByBrowseID(ctx context.Context, browseID string) (*model.CanonicalAlbum, error) {
	browseURL := a.baseURL + "/browse/" + browseID
	body, err := a.fetchPage(ctx, browseURL)
	if err != nil {
		return nil, err
	}
	return extractAlbum(body, browseURL)
}

func (a *Adapter) fetchPage(ctx context.Context, requestURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build youtube music request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute youtube music request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("%w %d: %s", errUnexpectedYouTubeMusicStatus, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read youtube music response: %w", err)
	}
	return body, nil
}
