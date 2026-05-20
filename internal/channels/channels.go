package channels

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/JD1359/notification-system-go/internal/models"
)

type Adapter interface {
	Send(ctx context.Context, n *models.Notification) error
}

type Registry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
}

func NewRegistry() *Registry {
	return &Registry{adapters: map[string]Adapter{}}
}

func (r *Registry) Register(name string, a Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[name] = a
}

func (r *Registry) Get(name string) (Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.adapters[name]
	return a, ok
}

// rateLimited wraps an adapter with a token-bucket rate limit.
type rateLimited struct {
	inner Adapter
	lim   *rate.Limiter
}

func WithRateLimit(a Adapter, rps int, burst int) Adapter {
	return &rateLimited{inner: a, lim: rate.NewLimiter(rate.Limit(rps), burst)}
}

func (r *rateLimited) Send(ctx context.Context, n *models.Notification) error {
	if err := r.lim.Wait(ctx); err != nil {
		return err
	}
	return r.inner.Send(ctx, n)
}

// ---- Mock adapters (replace with SendGrid, Twilio, FCM in prod) ----

type mockEmail struct{}

func NewMockEmail() Adapter { return &mockEmail{} }
func (m *mockEmail) Send(ctx context.Context, n *models.Notification) error {
	time.Sleep(20 * time.Millisecond)
	return nil
}

type mockSMS struct{}

func NewMockSMS() Adapter { return &mockSMS{} }
func (m *mockSMS) Send(ctx context.Context, n *models.Notification) error {
	time.Sleep(30 * time.Millisecond)
	return nil
}

type mockPush struct{}

func NewMockPush() Adapter { return &mockPush{} }
func (m *mockPush) Send(ctx context.Context, n *models.Notification) error {
	time.Sleep(15 * time.Millisecond)
	return nil
}
