package checker

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Clutz88/StatusGopher/internal/models"
)

func TestChecker(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
		url        string
		client     *http.Client
		slow       bool
	}{
		{"success response", http.StatusOK, false, "", &http.Client{}, false},
		{"internal server error response", http.StatusInternalServerError, false, "", &http.Client{}, false},
		{"no server", 0, true, "http://127.0.0.1:1", &http.Client{}, false},
		{"timeout", 0, true, "", &http.Client{Timeout: 5 * time.Millisecond}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.slow {
					time.Sleep(20 * time.Millisecond)
					return
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer ts.Close()
			if tt.url == "" {
				tt.url = ts.URL
			}

			res := Check(ctx, models.Site{URL: tt.url}, tt.client)

			if res.StatusCode != tt.statusCode {
				t.Errorf("Check(%q) statusCode = %v, wantStatusCode %v", ts.URL, res.StatusCode, tt.statusCode)
			}

			if (res.Err != "") != tt.wantErr {
				t.Errorf("Check(%q) err = %v, wantErr %v", ts.URL, res.Err, tt.wantErr)
			}
		})
	}
}
