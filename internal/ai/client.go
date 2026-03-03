package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type vectorResponse struct {
	Vector    []float64 `json:"vector"`
	Embedding []float64 `json:"embedding"`
	Data      struct {
		Vector    []float64 `json:"vector"`
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

func NewClient(baseURL string) (*Client, error) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return nil, fmt.Errorf("AI tool base url is empty")
	}

	return &Client{
		baseURL: strings.TrimRight(trimmed, "/"),
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}, nil
}

func (c *Client) TextToVector(ctx context.Context, text string) ([]float64, error) {
	payload := map[string]string{"text": text}
	return c.requestVector(ctx, "/vector/text", payload)
}

func (c *Client) AudioURIToVector(ctx context.Context, uri string) ([]float64, error) {
	payload := map[string]string{"uri": uri}
	return c.requestVector(ctx, "/vector/audio", payload)
}

func (c *Client) ImageURIToVector(ctx context.Context, uri string) ([]float64, error) {
	payload := map[string]string{"uri": uri}
	return c.requestVector(ctx, "/vector/image", payload)
}

func (c *Client) requestVector(ctx context.Context, path string, payload any) ([]float64, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	log.Printf("Requesting vector from AI tool at %s", c.baseURL+path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call AI tool: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		rawBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("AI tool status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(rawBody)))
	}

	var out vectorResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	vector := out.Vector
	if len(vector) == 0 {
		vector = out.Embedding
	}
	if len(vector) == 0 {
		vector = out.Data.Vector
	}
	if len(vector) == 0 {
		vector = out.Data.Embedding
	}
	if len(vector) == 0 {
		return nil, fmt.Errorf("AI tool returned empty vector")
	}

	return vector, nil
}
