package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"mortis/internal/logging"
)

const (
	groqURL      = "https://api.groq.com/openai/v1/chat/completions"
	defaultModel = "openai/gpt-oss-120b"
)

const (
	StateThinking = "thinking"
	StateIdle     = "idle"
)

type GroqClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	logger     *logging.Logger
	maxRetries int
}

type Option func(*GroqClient)

func WithModel(model string) Option {
	return func(c *GroqClient) { c.model = model }
}

func WithMaxRetries(n int) Option {
	return func(c *GroqClient) { c.maxRetries = n }
}

func WithHTTPClient(h *http.Client) Option {
	return func(c *GroqClient) { c.httpClient = h }
}

func NewGroqClient(apiKey string, logger *slog.Logger, opts ...Option) *GroqClient {
	c := &GroqClient{
		apiKey:     apiKey,
		model:      defaultModel,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logging.New(logger, "GroqClient"),
		maxRetries: 3,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Temperature float64       `json:"temperature"`
	Messages    []chatMessage `json:"messages"`
}

type ActivityOutput struct {
	ActivityName string         `json:"activityName"`
	Module       string         `json:"module"`
	Action       string         `json:"action"`
	Params       map[string]any `json:"params"`
	Response     string         `json:"response"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *GroqClient) Complete(ctx context.Context, systemPrompt, userInput string) (string, error) {
	req := chatRequest{
		Model:       c.model,
		Temperature: 0.2,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userInput},
		},
	}

	c.logger.Info("groq: sending request",
		"state", StateThinking,
		"input_len", len(userInput),
	)

	body, err := c.sendWithRetry(ctx, req)
	if err != nil {
		c.logger.Error("groq: request failed after retries",
			"state", StateIdle,
			"error", err,
		)
		return "", err
	}

	content, err := extractContent(body)
	if err != nil {
		c.logger.Error("groq: could not parse response", "error", err)
		return "", err
	}

	c.logger.Info("groq: received response",
		"state", StateIdle,
		"output_len", len(content),
	)
	return content, nil
}

func (c *GroqClient) sendWithRetry(ctx context.Context, payload chatRequest) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(500*(1<<attempt)) * time.Millisecond
			c.logger.Warn("groq: retrying request",
				"attempt", attempt,
				"backoff", backoff,
			)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		status, body, err := c.sendRequest(ctx, payload)
		if err != nil {
			return nil, err
		}
		lastErr = nil

		retryable := status == http.StatusTooManyRequests || status >= 500
		if retryable && attempt < c.maxRetries {
			c.logger.Warn("groq: retryable status", "status", status, "attempt", attempt)
			continue
		}

		if status < 200 || status >= 300 {
			return nil, fmt.Errorf("groq: unexpected status %d: %s", status, string(body))
		}

		return body, nil
	}

	return nil, lastErr
}

func (c *GroqClient) sendRequest(ctx context.Context, payload chatRequest) (status int, body []byte, err error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, fmt.Errorf("groq: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, groqURL, bytes.NewReader(data))
	if err != nil {
		return 0, nil, fmt.Errorf("groq: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("groq: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("groq: read body: %w", err)
	}

	return resp.StatusCode, body, nil
}

func extractContent(body []byte) (string, error) {
	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("groq: unmarshal response: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("groq: api error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("groq: no choices in response")
	}
	return parsed.Choices[0].Message.Content, nil
}