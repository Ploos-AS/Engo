package botai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const APIVersion = "1.0.0"

type Message struct {
	Role string `json:"role"`
	Content string `json:"content"`
}

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string, timeout time.Duration) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("BotAI URL is required")
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("BotAI timeout must be positive")
	}
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: timeout}}, nil
}

func (c *Client) Compatible(ctx context.Context) error {
	var out struct {
		Version string `json:"api_version"`
	}
	if err := c.request(ctx, http.MethodGet, "/v1/version", nil, &out); err != nil {
		return err
	}
	if out.Version != APIVersion {
		return fmt.Errorf("unsupported BotAI API version %q", out.Version)
	}
	return nil
}

func (c *Client) Chat(ctx context.Context, expert, message string) (string, error) {
	return c.ChatWithHistory(ctx, expert, nil, message)
}

func (c *Client) ChatWithHistory(ctx context.Context, expert string, history []Message, message string) (string, error) {
	message = strings.TrimSpace(message)
	if message == "" { return "", fmt.Errorf("message is required") }
	if len(history) > 20 { return "", fmt.Errorf("history exceeds 20 messages") }
	in := struct {
		Expert string `json:"expert,omitempty"`
		History []Message `json:"history,omitempty"`
		Message string `json:"message"`
	}{Expert:strings.TrimSpace(expert), History:history, Message:message}
	var out struct{ Text string `json:"text"` }
	if err := c.request(ctx, http.MethodPost, "/v1/chat", in, &out); err != nil { return "", err }
	return out.Text, nil
}

func (c *Client) request(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
		return fmt.Errorf("BotAI HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
}
