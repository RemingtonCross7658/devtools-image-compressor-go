package compression

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompressImageReference(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Image struct {
				Base64 string `json:"base64"`
			} `json:"image"`
			Format string `json:"format"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if request.Image.Base64 != "c2FtcGxl" || request.Format != "webp" {
			t.Errorf("unexpected request: %+v", request)
		}
		_, _ = w.Write([]byte(`{"ok":true,"data":{},"metadata":{}}`))
	}))
	defer server.Close()
	client := Client{BaseURL: server.URL, HTTPClient: server.Client(), APIKey: "test"}
	if _, err := client.Compress([]byte("sample"), "webp"); err != nil {
		t.Fatal(err)
	}
}

func TestDecideRelease(t *testing.T) {
	tests := []struct {
		name          string
		before, after int64
		publish       bool
	}{
		{"smaller image", 1200, 800, true},
		{"same size", 1200, 1200, false},
		{"larger image", 1200, 1400, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DecideRelease(tt.before, tt.after).Publish; got != tt.publish {
				t.Fatalf("publish=%v, want %v", got, tt.publish)
			}
		})
	}
}
