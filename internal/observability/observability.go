package observability

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

var (
	httpReqs = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests, labeled by method, path, and status.",
	}, []string{"method", "path", "status"})

	httpLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	}, []string{"method", "path"})
)

func init() {
	prometheus.MustRegister(httpReqs, httpLatency)
}

func PrometheusHandler() http.Handler { return promhttp.Handler() }

// RequestLogger emits a structured log line per request and records Prometheus metrics.
func RequestLogger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: 200}
			next.ServeHTTP(rec, r)
			dur := time.Since(start)
			httpReqs.WithLabelValues(r.Method, r.URL.Path, statusString(rec.status)).Inc()
			httpLatency.WithLabelValues(r.Method, r.URL.Path).Observe(dur.Seconds())
			log.Info().Str("method", r.Method).Str("path", r.URL.Path).Int("status", rec.status).Dur("dur", dur).Msg("http")
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func statusString(s int) string {
	switch {
	case s < 200:
		return "1xx"
	case s < 300:
		return "2xx"
	case s < 400:
		return "3xx"
	case s < 500:
		return "4xx"
	default:
		return "5xx"
	}
}
