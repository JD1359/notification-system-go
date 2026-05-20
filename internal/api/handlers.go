package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/JD1359/notification-system-go/internal/models"
	"github.com/JD1359/notification-system-go/internal/queue"
	"github.com/JD1359/notification-system-go/internal/storage"
)

type Handlers struct {
	Store *storage.Postgres
	Queue *queue.RedisStream
}

func NewHandlers(s *storage.Postgres, q *queue.RedisStream) *Handlers {
	return &Handlers{Store: s, Queue: q}
}

type enqueueRequest struct {
	Channel  models.Channel  `json:"channel"`
	To       string          `json:"to"`
	Subject  string          `json:"subject"`
	Body     string          `json:"body"`
	Metadata json.RawMessage `json:"metadata"`
}

func (r *enqueueRequest) validate() error {
	if r.Channel == "" {
		return errors.New("channel is required")
	}
	switch r.Channel {
	case models.ChannelEmail, models.ChannelSMS, models.ChannelPush:
	default:
		return errors.New("invalid channel")
	}
	if r.To == "" {
		return errors.New("to is required")
	}
	if r.Body == "" {
		return errors.New("body is required")
	}
	return nil
}

func (h *Handlers) Enqueue(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Idempotency-Key header required"})
		return
	}

	var req enqueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid json"})
		return
	}
	if err := req.validate(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	// Idempotency check
	existing, err := h.Store.Get(r.Context(), idempotencyKey)
	if err == nil && existing != nil {
		render.Status(r, http.StatusOK)
		render.JSON(w, r, existing)
		return
	}

	n := &models.Notification{
		ID:        idempotencyKey,
		Channel:   req.Channel,
		To:        req.To,
		Subject:   req.Subject,
		Body:      req.Body,
		Metadata:  req.Metadata,
		Status:    models.StatusQueued,
		QueuedAt:  time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := h.Store.Insert(r.Context(), n); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "persist failed"})
		return
	}

	if err := h.Queue.Enqueue(r.Context(), n); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "enqueue failed"})
		return
	}

	render.Status(r, http.StatusAccepted)
	render.JSON(w, r, n)
}

func (h *Handlers) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	n, err := h.Store.Get(r.Context(), id)
	if err != nil || n == nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "not found"})
		return
	}
	render.JSON(w, r, n)
}

func Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func Readyz(s *storage.Postgres, q *queue.RedisStream) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.Ping(r.Context()); err != nil {
			http.Error(w, "postgres unavailable", http.StatusServiceUnavailable)
			return
		}
		if err := q.Ping(r.Context()); err != nil {
			http.Error(w, "redis unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	}
}
