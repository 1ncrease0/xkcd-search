package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/1ncrease0/xkcd-search/frontend/core"
)

type Comic struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}

type SearchResponse struct {
	Comics []Comic `json:"comics"`
	Total  int     `json:"total"`
}

type LoginRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type Client struct {
	url  string
	http *http.Client
	log  *slog.Logger
}

func New(url string, timeout time.Duration, log *slog.Logger) *Client {
	return &Client{
		url: url,
		http: &http.Client{
			Timeout: timeout,
		},
		log: log,
	}
}

func makeURL(base, phrase string, limit int) string {
	return fmt.Sprintf("%s/api/isearch?phrase=%s&limit=%d", base, url.QueryEscape(phrase), limit)
}

func (c *Client) Search(ctx context.Context, phrase string, limit int) ([]core.Comic, error) {
	u := makeURL(c.url, phrase, limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.log.Error("close response body", "error", err)
		}
	}()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusServiceUnavailable:
		return nil, core.ErrRateLimited
	default:
		return nil, fmt.Errorf("api returned %d", resp.StatusCode)
	}

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	comics := make([]core.Comic, len(result.Comics))
	for i, comic := range result.Comics {
		comics[i] = core.Comic{
			ID:  comic.ID,
			URL: comic.URL,
		}
	}
	return comics, nil
}

func (c *Client) Login(ctx context.Context, username, password string) (string, error) {
	body, err := json.Marshal(LoginRequest{Name: username, Password: password})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/api/login", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.log.Error("close response body", "error", err)
		}
	}()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return "", core.ErrUnauthorized
	default:
		return "", fmt.Errorf("login api returned %d", resp.StatusCode)
	}

	tokenBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(tokenBytes)), nil
}

type dbStatsJSON struct {
	WordsTotal    int `json:"words_total"`
	WordsUnique   int `json:"words_unique"`
	ComicsFetched int `json:"comics_fetched"`
	ComicsTotal   int `json:"comics_total"`
}

type pingJSON struct {
	Replies map[string]string `json:"replies"`
}

func (c *Client) DBStats(ctx context.Context, token string) (core.AdminStats, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"/api/db/stats", nil)
	if err != nil {
		return core.AdminStats{}, err
	}
	req.Header.Set("Authorization", "Token "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return core.AdminStats{}, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.log.Error("close response body", "error", err)
		}
	}()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return core.AdminStats{}, core.ErrUnauthorized
	default:
		return core.AdminStats{}, fmt.Errorf("stats api returned %d", resp.StatusCode)
	}
	var raw dbStatsJSON
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return core.AdminStats{}, err
	}
	return core.AdminStats{
		WordsTotal:    raw.WordsTotal,
		WordsUnique:   raw.WordsUnique,
		ComicsFetched: raw.ComicsFetched,
		ComicsTotal:   raw.ComicsTotal,
	}, nil
}

func (c *Client) Ping(ctx context.Context, token string) (map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"/api/ping", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.log.Error("close response body", "error", err)
		}
	}()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return nil, core.ErrUnauthorized
	default:
		return nil, fmt.Errorf("ping api returned %d", resp.StatusCode)
	}
	var raw pingJSON
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	if raw.Replies == nil {
		return map[string]string{}, nil
	}
	return raw.Replies, nil
}

type dbJobStatusJSON struct {
	Status string `json:"status"`
}

func (c *Client) DBJobStatus(ctx context.Context, token string) (core.JobStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"/api/db/status", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Token "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.log.Error("close response body", "error", err)
		}
	}()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return "", core.ErrUnauthorized
	default:
		return "", fmt.Errorf("db status api returned %d", resp.StatusCode)
	}
	var raw dbJobStatusJSON
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return "", err
	}
	return core.JobStatus(raw.Status), nil
}

func (c *Client) DBUpdate(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/api/db/update", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.log.Error("close response body db update", "error", err)
		}
	}()
	switch resp.StatusCode {
	case http.StatusOK, http.StatusAccepted:
		return nil
	case http.StatusUnauthorized:
		return core.ErrUnauthorized
	default:
		return fmt.Errorf("update api returned %d", resp.StatusCode)
	}
}

func (c *Client) DBDrop(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.url+"/api/db", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.log.Error("close response body db drop", "error", err)
		}
	}()
	switch resp.StatusCode {
	case http.StatusOK, http.StatusConflict:
		return nil
	case http.StatusUnauthorized:
		return core.ErrUnauthorized
	default:
		return fmt.Errorf("drop api returned %d", resp.StatusCode)
	}
}
