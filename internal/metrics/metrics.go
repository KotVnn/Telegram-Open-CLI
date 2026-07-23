package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

// Collector holds all application metrics.
type Collector struct {
	registry *prometheus.Registry
	server   *http.Server
	logger   zerolog.Logger

	// Session metrics
	sessionsActive  prometheus.Gauge
	sessionsCreated prometheus.Counter

	// Message metrics
	messagesReceived prometheus.Counter
	messagesSent     prometheus.Counter

	// Backend metrics
	backendRequestsTotal *prometheus.CounterVec
	backendErrorsTotal   *prometheus.CounterVec
	backendResponseTime  *prometheus.HistogramVec

	// User metrics
	usersActive prometheus.Gauge
	usersTotal  prometheus.Gauge

	// Startup time
	startTime prometheus.Gauge
}

// NewCollector creates a new metrics collector.
func NewCollector(logger zerolog.Logger) *Collector {
	registry := prometheus.NewRegistry()

	c := &Collector{
		registry: registry,
		logger:   logger,
	}

	// Session metrics
	c.sessionsActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "toc_sessions_active",
		Help: "Number of currently active sessions",
	})

	c.sessionsCreated = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "toc_sessions_created_total",
		Help: "Total number of sessions created",
	})

	// Message metrics
	c.messagesReceived = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "toc_messages_received_total",
		Help: "Total number of messages received",
	})

	c.messagesSent = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "toc_messages_sent_total",
		Help: "Total number of messages sent",
	})

	// Backend metrics
	c.backendRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "toc_backend_requests_total",
		Help: "Total number of backend requests",
	}, []string{"backend", "operation"})

	c.backendErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "toc_backend_errors_total",
		Help: "Total number of backend errors",
	}, []string{"backend", "error_type"})

	c.backendResponseTime = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "toc_backend_response_time_seconds",
		Help:    "Backend response time in seconds",
		Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
	}, []string{"backend", "operation"})

	// User metrics
	c.usersActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "toc_users_active",
		Help: "Number of active users",
	})

	c.usersTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "toc_users_total",
		Help: "Total number of users",
	})

	// Startup time
	c.startTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "toc_start_time_seconds",
		Help: "Application start time in Unix timestamp",
	})

	// Register all metrics
	registry.MustRegister(
		c.sessionsActive,
		c.sessionsCreated,
		c.messagesReceived,
		c.messagesSent,
		c.backendRequestsTotal,
		c.backendErrorsTotal,
		c.backendResponseTime,
		c.usersActive,
		c.usersTotal,
		c.startTime,
	)

	return c
}

// Start begins the metrics HTTP server.
func (c *Collector) Start(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(c.registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	c.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	c.startTime.Set(float64(time.Now().Unix()))

	c.logger.Info().Str("addr", addr).Msg("starting metrics server")

	go func() {
		if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			c.logger.Error().Err(err).Msg("metrics server error")
		}
	}()

	return nil
}

// Stop gracefully stops the metrics server.
func (c *Collector) Stop(ctx context.Context) error {
	if c.server != nil {
		return c.server.Shutdown(ctx)
	}
	return nil
}

// IncSessionsCreated increments the sessions created counter and active gauge.
func (c *Collector) IncSessionsCreated() {
	c.sessionsCreated.Inc()
	c.sessionsActive.Inc()
}

// DecSessionsActive decrements the active sessions gauge.
func (c *Collector) DecSessionsActive() {
	c.sessionsActive.Dec()
}

// IncMessagesReceived increments the messages received counter.
func (c *Collector) IncMessagesReceived() {
	c.messagesReceived.Inc()
}

// IncMessagesSent increments the messages sent counter.
func (c *Collector) IncMessagesSent() {
	c.messagesSent.Inc()
}

// IncBackendRequests increments the backend requests counter.
func (c *Collector) IncBackendRequests(backend, operation string) {
	c.backendRequestsTotal.WithLabelValues(backend, operation).Inc()
}

// IncBackendErrors increments the backend errors counter.
func (c *Collector) IncBackendErrors(backend, errorType string) {
	c.backendErrorsTotal.WithLabelValues(backend, errorType).Inc()
}

// ObserveBackendResponseTime records a backend response time observation.
func (c *Collector) ObserveBackendResponseTime(backend, operation string, duration time.Duration) {
	c.backendResponseTime.WithLabelValues(backend, operation).Observe(duration.Seconds())
}

// SetUsersActive sets the active users gauge.
func (c *Collector) SetUsersActive(count float64) {
	c.usersActive.Set(count)
}

// SetUsersTotal sets the total users gauge.
func (c *Collector) SetUsersTotal(count float64) {
	c.usersTotal.Set(count)
}
