package xkcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/1ncrease0/xkcd-search/shared/closers"

	"github.com/1ncrease0/xkcd-search/update/core"
)

type Client struct {
	log    *slog.Logger
	client http.Client
	url    string
}

func NewClient(url string, timeout time.Duration, log *slog.Logger) (*Client, error) {
	if url == "" {
		return nil, fmt.Errorf("empty base url specified")
	}
	return &Client{
		client: http.Client{Timeout: timeout},
		log:    log,
		url:    url,
	}, nil
}

type xkcdResponse struct {
	Num        int    `json:"num"`
	SafeTitle  string `json:"safe_title"`
	Title      string `json:"title"`
	Img        string `json:"img"`
	Alt        string `json:"alt"`
	Transcript string `json:"transcript"`
}

func (c Client) Get(ctx context.Context, id int) (core.XKCDInfo, error) {
	url := fmt.Sprintf("%s/%d/info.0.json", c.url, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return core.XKCDInfo{}, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return core.XKCDInfo{}, fmt.Errorf("do request: %w", err)
	}
	defer closers.CloseOrLog(resp.Body, c.log)

	if resp.StatusCode == http.StatusNotFound {
		return core.XKCDInfo{}, fmt.Errorf("comic %d not found: %w", id, core.ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return core.XKCDInfo{}, fmt.Errorf("unexpected status %d for comic %d", resp.StatusCode, id)
	}

	var xkcd xkcdResponse
	if err := json.NewDecoder(resp.Body).Decode(&xkcd); err != nil {
		return core.XKCDInfo{}, fmt.Errorf("decode response: %w", err)
	}

	return xkcd.toInfo(), nil
}

func (c Client) LastID(ctx context.Context) (int, error) {
	url := fmt.Sprintf("%s/info.0.json", c.url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("do request: %w", err)
	}
	defer closers.CloseOrLog(resp.Body, c.log)

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var xkcd xkcdResponse
	if err := json.NewDecoder(resp.Body).Decode(&xkcd); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}

	return xkcd.Num, nil
}

func (r xkcdResponse) toInfo() core.XKCDInfo {
	description := r.Transcript
	if description == "" {
		description = r.Alt
	}

	title := r.SafeTitle
	if title == "" {
		title = r.Title
	}

	return core.XKCDInfo{
		ID:          r.Num,
		URL:         r.Img,
		Title:       title,
		Description: description,
	}
}
