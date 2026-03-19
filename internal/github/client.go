package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Client wraps the GitHub API with token-aware error handling.
type Client struct {
	httpClient *http.Client
	token      string
	statusMu   sync.RWMutex
	status     RateLimitStatus
}

// NewClient constructs a GitHub API client.
func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 45 * time.Second},
		token:      token,
		status: RateLimitStatus{
			Authenticated: token != "",
		},
	}
}

// Request sends an authenticated or anonymous GitHub API request.
func (c *Client) Request(ctx context.Context, method, target string, body any) (*http.Response, error) {
	reader, err := encodeBody(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("User-Agent", "filerepo/1.0.0")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &APIError{Kind: ErrorAPI, Message: err.Error()}
	}
	c.captureRateLimit(resp)

	if apiErr := classifyResponse(resp, c.token != ""); apiErr != nil {
		return nil, apiErr
	}

	return resp, nil
}

// FetchContents loads a directory or file response from the contents API.
func (c *Client) FetchContents(ctx context.Context, target string) ([]RepoItem, error) {
	resp, err := c.Request(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read GitHub response: %w", err)
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty GitHub response")
	}

	if trimmed[0] == '[' {
		var items []RepoItem
		if err := json.Unmarshal(trimmed, &items); err != nil {
			return nil, fmt.Errorf("parse GitHub API response: %w", err)
		}
		return items, nil
	}

	var item RepoItem
	if err := json.Unmarshal(trimmed, &item); err != nil {
		return nil, fmt.Errorf("parse GitHub API response: %w", err)
	}

	return []RepoItem{item}, nil
}

// FetchRecursiveTree loads the full repository tree using the git trees API.
func (c *Client) FetchRecursiveTree(ctx context.Context, owner, repo, branch string) (GitTreeResponse, error) {
	target := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1",
		owner,
		repo,
		url.PathEscape(branch),
	)
	resp, err := c.Request(ctx, http.MethodGet, target, nil)
	if err != nil {
		return GitTreeResponse{}, err
	}
	defer resp.Body.Close()

	var tree GitTreeResponse
	if err := json.NewDecoder(resp.Body).Decode(&tree); err != nil {
		return GitTreeResponse{}, fmt.Errorf("parse Git tree response: %w", err)
	}

	return tree, nil
}

// FetchRawContent downloads textual response bodies.
func (c *Client) FetchRawContent(ctx context.Context, target string) (string, error) {
	data, err := c.DownloadBinary(ctx, target)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FetchJSON decodes a JSON payload into the provided target.
func (c *Client) FetchJSON(ctx context.Context, target string, value any) error {
	resp, err := c.Request(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(value); err != nil {
		return fmt.Errorf("parse GitHub JSON response: %w", err)
	}

	return nil
}

// DownloadBinary downloads raw bytes from a GitHub endpoint.
func (c *Client) DownloadBinary(ctx context.Context, target string) ([]byte, error) {
	resp, err := c.Request(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read binary response: %w", err)
	}

	return data, nil
}

// GetLFSDownloadURL resolves a git-lfs object into its real download URL.
func (c *Client) GetLFSDownloadURL(ctx context.Context, owner, repo, oid string, size uint64) (string, error) {
	target := fmt.Sprintf("https://github.com/%s/%s.git/info/lfs/objects/batch", owner, repo)
	payload := map[string]any{
		"operation": "download",
		"transfers": []string{"basic"},
		"objects": []map[string]any{
			{"oid": oid, "size": size},
		},
	}

	resp, err := c.Request(ctx, http.MethodPost, target, payload)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var batch lfsBatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
		return "", fmt.Errorf("parse LFS response: %w", err)
	}

	if len(batch.Objects) == 0 || batch.Objects[0].Actions == nil || batch.Objects[0].Actions.Download == nil {
		return "", fmt.Errorf("no download URL in LFS response")
	}

	return batch.Objects[0].Actions.Download.Href, nil
}

// DecodeBase64Content decodes a GitHub API base64 content field.
func DecodeBase64Content(encoded string) (string, error) {
	encoded = strings.ReplaceAll(encoded, "\n", "")
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode base64 content: %w", err)
	}
	return string(data), nil
}

// Status returns the most recently observed rate-limit/auth status.
func (c *Client) Status() RateLimitStatus {
	c.statusMu.RLock()
	defer c.statusMu.RUnlock()
	return c.status
}

func encodeBody(body any) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode request body: %w", err)
	}

	return bytes.NewReader(data), nil
}

func classifyResponse(resp *http.Response, hasToken bool) error {
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		resp.Body.Close()
		if hasToken {
			return &APIError{Kind: ErrorInvalidToken, Message: "invalid token; falling back to public API"}
		}
	case http.StatusForbidden:
		remaining := resp.Header.Get("X-RateLimit-Remaining")
		resp.Body.Close()
		switch {
		case remaining == "0":
			level := "unauthenticated user"
			if hasToken {
				level = "authenticated user"
			}
			return &APIError{
				Kind:    ErrorRateLimitReached,
				Message: fmt.Sprintf("rate limit reached for %s; add a token via `filerepo config set token YOUR_TOKEN`", level),
			}
		case hasToken:
			return &APIError{Kind: ErrorInvalidToken, Message: "invalid token; falling back to public API"}
		default:
			return &APIError{Kind: ErrorAPI, Message: "forbidden"}
		}
	case http.StatusNotFound:
		resp.Body.Close()
		return &APIError{Kind: ErrorNotFound, Message: resp.Request.URL.String()}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		status := resp.Status
		resp.Body.Close()
		return &APIError{Kind: ErrorAPI, Message: status}
	}

	return nil
}

func (c *Client) captureRateLimit(resp *http.Response) {
	status := RateLimitStatus{
		Authenticated: c.token != "",
	}

	if value, err := strconv.Atoi(resp.Header.Get("X-RateLimit-Limit")); err == nil {
		status.Limit = value
	}
	if value, err := strconv.Atoi(resp.Header.Get("X-RateLimit-Remaining")); err == nil {
		status.Remaining = value
	}
	if value, err := strconv.Atoi(resp.Header.Get("X-RateLimit-Used")); err == nil {
		status.Used = value
	}
	if unixValue, err := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64); err == nil && unixValue > 0 {
		status.ResetAt = time.Unix(unixValue, 0).UTC()
	}

	c.statusMu.Lock()
	c.status = status
	c.statusMu.Unlock()
}
