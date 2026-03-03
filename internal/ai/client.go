package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
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

func (c *Client) ImageUploadToVector(ctx context.Context, fileName string, data []byte) ([]float64, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	contentType := http.DetectContentType(data)
	if !strings.HasPrefix(contentType, "image/") {
		return nil, fmt.Errorf("uploaded file is not an image (detected: %s)", contentType)
	}

	finalName := normalizeImageFilename(fileName, contentType)
	headers := make(textproto.MIMEHeader)
	headers.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, finalName))
	headers.Set("Content-Type", contentType)

	part, err := writer.CreatePart(headers)
	if err != nil {
		return nil, fmt.Errorf("create image part: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return nil, fmt.Errorf("write image data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/vector/image", &body)
	if err != nil {
		return nil, fmt.Errorf("build upload request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload image to AI tool: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		rawBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("AI tool status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(rawBody)))
	}

	return decodeVectorResponse(resp.Body)
}

func (c *Client) requestVector(ctx context.Context, path string, payload any) ([]float64, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

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

	return decodeVectorResponse(resp.Body)
}

func decodeVectorResponse(body io.Reader) ([]float64, error) {
	var out vectorResponse
	if err := json.NewDecoder(body).Decode(&out); err != nil {
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

func sanitizeUploadFilename(name string) string {
	base := strings.TrimSpace(filepath.Base(name))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "image.bin"
	}
	return strings.ReplaceAll(base, " ", "_")
}

func normalizeImageFilename(name, contentType string) string {
	base := sanitizeUploadFilename(name)
	ext := strings.ToLower(filepath.Ext(base))
	if ext != "" {
		return base
	}

	switch contentType {
	case "image/jpeg":
		return base + ".jpg"
	case "image/png":
		return base + ".png"
	case "image/webp":
		return base + ".webp"
	case "image/gif":
		return base + ".gif"
	default:
		if exts, _ := mime.ExtensionsByType(contentType); len(exts) > 0 {
			return base + exts[0]
		}
		return base + ".img"
	}
}
