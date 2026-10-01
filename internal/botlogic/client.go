package botlogic

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

const APIVersion = "v1"

type Client struct {
	baseURL string
	http    *http.Client
}

type Solution map[string]string

type QueryResult struct {
	Ruleset   string     `json:"ruleset"`
	Revision  uint64     `json:"revision"`
	Solutions []Solution `json:"solutions"`
}

func New(baseURL string, timeout time.Duration) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("BotLogic URL is required")
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("BotLogic timeout must be positive")
	}
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: timeout}}, nil
}

func (c *Client) Compatible(ctx context.Context) error {
	var out struct {
		Service string `json:"service"`
		API     string `json:"api"`
	}
	if err := c.request(ctx, http.MethodGet, "/v1/version", nil, &out); err != nil {
		return err
	}
	if out.Service != "botlogic" {
		return fmt.Errorf("unexpected BotLogic service %q", out.Service)
	}
	if out.API != APIVersion {
		return fmt.Errorf("unsupported BotLogic API %q", out.API)
	}
	return nil
}

func (c *Client) Query(ctx context.Context, ruleset, query string) (QueryResult, error) {
	if strings.TrimSpace(ruleset) == "" || strings.TrimSpace(query) == "" {
		return QueryResult{}, fmt.Errorf("ruleset and query are required")
	}
	in := struct {
		Ruleset string `json:"ruleset"`
		Query   string `json:"query"`
	}{ruleset, query}
	var out QueryResult
	if err := c.request(ctx, http.MethodPost, "/v1/query", in, &out); err != nil {
		return QueryResult{}, err
	}
	return out, nil
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
		return fmt.Errorf("BotLogic HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
}

type Fact struct {
	Predicate string   `json:"predicate"`
	Args      []string `json:"args"`
}
type FactOperation struct {
	Op   string `json:"op"`
	Fact Fact   `json:"fact"`
}
type MutationResult struct {
	OK       bool   `json:"ok"`
	Ruleset  string `json:"ruleset"`
	Revision uint64 `json:"revision"`
}

func (c *Client) ApplyFacts(ctx context.Context, ruleset string, operations []FactOperation) (MutationResult, error) {
	if strings.TrimSpace(ruleset) == "" || len(operations) == 0 {
		return MutationResult{}, fmt.Errorf("ruleset and operations are required")
	}
	in := struct {
		Ruleset    string          `json:"ruleset"`
		Operations []FactOperation `json:"operations"`
	}{ruleset, operations}
	var out MutationResult
	if err := c.request(ctx, http.MethodPost, "/v1/facts/batch", in, &out); err != nil {
		return MutationResult{}, err
	}
	return out, nil
}
