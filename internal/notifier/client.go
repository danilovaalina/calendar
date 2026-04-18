package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/cockroachdb/errors"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(url string) *Client {
	return &Client{
		baseURL: url,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

type CreateRequest struct {
	Channel       string    `json:"channel"`
	Recipient     string    `json:"recipient"`
	Message       string    `json:"message"`
	ScheduledTime time.Time `json:"scheduled_time"`
}

func (c *Client) Send(ctx context.Context, req CreateRequest) error {
	u, err := url.JoinPath(c.baseURL, "/api/notify")
	if err != nil {
		return errors.Newf("invalid base URL: %w", err)
	}

	data, err := json.Marshal(req)
	if err != nil {
		return errors.Newf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewBuffer(data))
	if err != nil {
		return errors.Newf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return errors.Newf("notifier returned status: %d", resp.StatusCode)
	}
	return nil
}
