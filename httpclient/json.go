package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
)

func GetJSON[T any](ctx context.Context, c *http.Client, url string, out *T) (*http.Response, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if out != nil {
		b, _ := io.ReadAll(resp.Body)
		if err := json.Unmarshal(b, out); err != nil {
			return resp, err
		}
	}
	return resp, nil
}

func PostJSON[T any](ctx context.Context, c *http.Client, url string, in any, out *T) (*http.Response, error) {
	var buf bytes.Buffer
	if in != nil {
		if err := json.NewEncoder(&buf).Encode(in); err != nil {
			return nil, err
		}
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if out != nil {
		b, _ := io.ReadAll(resp.Body)
		if err := json.Unmarshal(b, out); err != nil {
			return resp, err
		}
	}
	return resp, nil
}
