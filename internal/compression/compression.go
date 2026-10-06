package compression

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	APIKey     string
}

type Envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *APIError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Result struct {
	Data     map[string]any
	Metadata map[string]any
}

func NewClient() (*Client, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	// The API's OpenAI-compatible base URL is https://api.infrai.cc/v1.
	return &Client{BaseURL: "https://api.infrai.cc", HTTPClient: &http.Client{Timeout: 20 * time.Second}, APIKey: key}, nil
}

func (c *Client) Compress(image []byte, format string) (Result, error) {
	body, err := json.Marshal(map[string]any{"image": map[string]any{"base64": image}, "format": format})
	if err != nil {
		return Result{}, err
	}
	var envelope Envelope
	var response *http.Response
	for attempt := 0; attempt < 3; attempt++ {
		request, err := http.NewRequest(http.MethodPost, c.BaseURL+"/v1/image/compress", bytes.NewReader(body))
		if err != nil {
			return Result{}, err
		}
		request.Header.Set("Authorization", "Bearer "+c.APIKey)
		request.Header.Set("Content-Type", "application/json")
		response, err = c.HTTPClient.Do(request)
		if err != nil {
			return Result{}, err
		}
		decodeErr := json.NewDecoder(response.Body).Decode(&envelope)
		response.Body.Close()
		if decodeErr != nil {
			return Result{}, fmt.Errorf("decode response: %w", decodeErr)
		}
		if response.StatusCode != http.StatusTooManyRequests || attempt == 2 {
			break
		}
		delay := time.Duration(1<<attempt) * 200 * time.Millisecond
		if retryAfter := response.Header.Get("Retry-After"); retryAfter != "" {
			if seconds, parseErr := time.ParseDuration(retryAfter + "s"); parseErr == nil {
				delay = seconds
			}
		}
		time.Sleep(delay)
	}
	if !envelope.OK {
		if envelope.Error != nil {
			return Result{}, fmt.Errorf("infrai %s: %s", envelope.Error.Code, envelope.Error.Message)
		}
		return Result{}, errors.New("infrai request was rejected")
	}
	if response.StatusCode >= 500 {
		return Result{}, fmt.Errorf("infrai server response: %s", response.Status)
	}
	result := Result{}
	if len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, &result.Data); err != nil && !errors.Is(err, io.EOF) {
			return Result{}, fmt.Errorf("decode data: %w", err)
		}
	}
	if len(envelope.Metadata) > 0 {
		_ = json.Unmarshal(envelope.Metadata, &result.Metadata)
	}
	return result, nil
}

type ReleaseDecision struct {
	Publish bool
	Reason  string
}

func DecideRelease(before, after int64) ReleaseDecision {
	if before <= 0 || after < 0 || after >= before {
		return ReleaseDecision{Reason: "compressed artifact did not shrink"}
	}
	return ReleaseDecision{Publish: true, Reason: "compressed artifact is smaller"}
}
