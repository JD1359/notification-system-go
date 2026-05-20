package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestEnqueueRejectsMissingIdempotencyKey(t *testing.T) {
	r := chi.NewRouter()
	h := &Handlers{}
	r.Post("/v1/notifications", h.Enqueue)

	req := httptest.NewRequest("POST", "/v1/notifications",
		strings.NewReader(`{"channel":"email","to":"a@b","body":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestEnqueueValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{"missing channel", `{"to":"a","body":"x"}`, http.StatusBadRequest},
		{"invalid channel", `{"channel":"fax","to":"a","body":"x"}`, http.StatusBadRequest},
		{"missing to",      `{"channel":"email","body":"x"}`, http.StatusBadRequest},
		{"missing body",    `{"channel":"email","to":"a"}`, http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := chi.NewRouter()
			h := &Handlers{}
			r.Post("/v1/notifications", h.Enqueue)

			req := httptest.NewRequest("POST", "/v1/notifications", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", "test-key")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.want {
				t.Fatalf("want %d, got %d", tc.want, w.Code)
			}
		})
	}
}
