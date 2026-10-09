package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type AIEngineClient interface {
	ExtractFace(ctx context.Context, filename string, fileData []byte) ([]float32, error)
	EnrollMerge(ctx context.Context, files []UploadedFile) ([]float32, error)
	CompareEmbeddings(ctx context.Context, v1, v2 []float32) (float64, bool, error)
	CheckHealth(ctx context.Context) (string, error)
}

type UploadedFile struct {
	Filename string
	Data     []byte
}

type aiEngineClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAIEngineClient(baseURL string) AIEngineClient {
	return &aiEngineClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type ExtractResponse struct {
	Success       bool      `json:"success"`
	DetectedFaces int       `json:"detected_faces"`
	Embedding     []float32 `json:"embedding"`
	Detail        any       `json:"detail,omitempty"`
}

type EnrollMergeResponse struct {
	Success           bool      `json:"success"`
	ValidSamples      int       `json:"valid_samples"`
	AveragedEmbedding []float32 `json:"averaged_embedding"`
	Detail            any       `json:"detail,omitempty"`
}

type CompareResponse struct {
	Success    bool    `json:"success"`
	Similarity float64 `json:"similarity"`
	IsMatch    bool    `json:"is_match"`
	Detail     any     `json:"detail,omitempty"`
}

func (c *aiEngineClient) ExtractFace(ctx context.Context, filename string, fileData []byte) ([]float32, error) {
	url := fmt.Sprintf("%s/extract", c.baseURL)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("image", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create multipart form file: %w", err)
	}
	if _, err := part.Write(fileData); err != nil {
		return nil, fmt.Errorf("failed to write file data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to AI engine (%s): %w", url, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read AI engine response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Detail any `json:"detail"`
		}
		if jsonErr := json.Unmarshal(respBytes, &errResp); jsonErr == nil && errResp.Detail != nil {
			return nil, fmt.Errorf("AI engine error (%d): %v", resp.StatusCode, errResp.Detail)
		}
		return nil, fmt.Errorf("AI engine returned HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	var extractResp ExtractResponse
	if err := json.Unmarshal(respBytes, &extractResp); err != nil {
		return nil, fmt.Errorf("failed to parse AI engine response: %w", err)
	}

	if len(extractResp.Embedding) != 512 {
		return nil, fmt.Errorf("unexpected embedding dimension: %d (expected 512)", len(extractResp.Embedding))
	}

	return extractResp.Embedding, nil
}

func (c *aiEngineClient) EnrollMerge(ctx context.Context, files []UploadedFile) ([]float32, error) {
	url := fmt.Sprintf("%s/enroll-merge", c.baseURL)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for _, f := range files {
		part, err := writer.CreateFormFile("images", f.Filename)
		if err != nil {
			return nil, fmt.Errorf("failed to create multipart part: %w", err)
		}
		if _, err := part.Write(f.Data); err != nil {
			return nil, fmt.Errorf("failed to write multipart data: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to AI engine (%s): %w", url, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read AI engine response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Detail any `json:"detail"`
		}
		if jsonErr := json.Unmarshal(respBytes, &errResp); jsonErr == nil && errResp.Detail != nil {
			return nil, fmt.Errorf("AI engine error (%d): %v", resp.StatusCode, errResp.Detail)
		}
		return nil, fmt.Errorf("AI engine returned HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	var mergeResp EnrollMergeResponse
	if err := json.Unmarshal(respBytes, &mergeResp); err != nil {
		return nil, fmt.Errorf("failed to parse AI engine response: %w", err)
	}

	if len(mergeResp.AveragedEmbedding) != 512 {
		return nil, fmt.Errorf("unexpected averaged embedding dimension: %d (expected 512)", len(mergeResp.AveragedEmbedding))
	}

	return mergeResp.AveragedEmbedding, nil
}

func (c *aiEngineClient) CompareEmbeddings(ctx context.Context, v1, v2 []float32) (float64, bool, error) {
	url := fmt.Sprintf("%s/compare", c.baseURL)

	payload := map[string][]float32{
		"vector1": v1,
		"vector2": v2,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, false, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return 0, false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, false, fmt.Errorf("failed to connect to AI engine (%s): %w", url, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, false, err
	}

	if resp.StatusCode != http.StatusOK {
		return 0, false, fmt.Errorf("AI engine compare failed with HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	var compResp CompareResponse
	if err := json.Unmarshal(respBytes, &compResp); err != nil {
		return 0, false, err
	}

	return compResp.Similarity, compResp.IsMatch, nil
}

func (c *aiEngineClient) CheckHealth(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/health", c.baseURL)

	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return "unreachable", fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "unreachable", fmt.Errorf("failed to connect to AI engine health endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "unhealthy", fmt.Errorf("AI engine health check returned HTTP %d", resp.StatusCode)
	}

	var healthResp struct {
		Status      string `json:"status"`
		Service     string `json:"service"`
		ModelLoaded bool   `json:"model_loaded"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err == nil {
		if !healthResp.ModelLoaded {
			return "model_not_ready", nil
		}
	}

	return "healthy", nil
}

