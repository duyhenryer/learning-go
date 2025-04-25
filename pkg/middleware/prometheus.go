package middleware

import (
	"bytes"
	"io"
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// RequestLatency measures the latency of HTTP requests in seconds
	RequestLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_duration_seconds",
			Help:    "Latency of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	// RequestTotal counts the total number of HTTP requests
	RequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// RequestsInFlight tracks the number of requests currently being processed
	RequestsInFlight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "requests_in_flight",
			Help: "Number of requests currently being processed",
		},
		[]string{"method", "path"},
	)

	// RequestSize measures the size of HTTP request bodies in bytes
	RequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_size_bytes",
			Help:    "Size of HTTP request bodies in bytes",
			Buckets: []float64{0, 100, 500, 1000, 5000, 10000, 50000},
		},
		[]string{"method", "path"},
	)

	// ResponseSize measures the size of HTTP response bodies in bytes
	ResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "response_size_bytes",
			Help:    "Size of HTTP response bodies in bytes",
			Buckets: []float64{0, 100, 500, 1000, 5000, 10000, 50000},
		},
		[]string{"method", "path", "status"},
	)

	// ErrorRateTotal counts the total number of HTTP requests with errors (4xx, 5xx)
	ErrorRateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "error_rate_total",
			Help: "Total number of HTTP requests with errors (4xx, 5xx)",
		},
		[]string{"method", "path", "status"},
	)
)

func init() {
	log.Println("Registering Prometheus metrics...")
	prometheus.MustRegister(RequestLatency)
	prometheus.MustRegister(RequestTotal)
	prometheus.MustRegister(RequestsInFlight)
	prometheus.MustRegister(RequestSize)
	prometheus.MustRegister(ResponseSize)
	prometheus.MustRegister(ErrorRateTotal)
	log.Println("Metrics registered successfully")
}

// PrometheusMiddleware instruments Gin requests with Prometheus metrics
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("Middleware start: %s %s", c.Request.Method, c.Request.URL.Path)
		RequestsInFlight.WithLabelValues(c.Request.Method, c.Request.URL.Path).Inc()

		// Capture request body size
		var requestSize float64
		if c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				requestSize = float64(len(bodyBytes))
				// Restore body for downstream handlers
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}
		RequestSize.WithLabelValues(c.Request.Method, c.Request.URL.Path).Observe(requestSize)

		// Capture response size
		writer := &bodyWriter{ResponseWriter: c.Writer, body: bytes.NewBuffer(nil)}
		c.Writer = writer

		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		// Record metrics
		log.Printf("Recording latency: method=%s, path=%s, status=%s, duration=%fs",
			c.Request.Method, c.Request.URL.Path, status, duration)
		RequestLatency.WithLabelValues(c.Request.Method, c.Request.URL.Path, status).Observe(duration)

		log.Printf("Incrementing request total: method=%s, path=%s, status=%s",
			c.Request.Method, c.Request.URL.Path, status)
		RequestTotal.WithLabelValues(c.Request.Method, c.Request.URL.Path, status).Inc()

		// Increment error rate for 4xx and 5xx status codes
		if c.Writer.Status() >= 400 {
			log.Printf("Incrementing error rate: method=%s, path=%s, status=%s",
				c.Request.Method, c.Request.URL.Path, status)
			ErrorRateTotal.WithLabelValues(c.Request.Method, c.Request.URL.Path, status).Inc()
		}

		log.Printf("Recording response size: method=%s, path=%s, status=%s, size=%d bytes",
			c.Request.Method, c.Request.URL.Path, status, writer.body.Len())
		ResponseSize.WithLabelValues(c.Request.Method, c.Request.URL.Path, status).Observe(float64(writer.body.Len()))

		RequestsInFlight.WithLabelValues(c.Request.Method, c.Request.URL.Path).Dec()
		log.Printf("Middleware end: %s %s", c.Request.Method, c.Request.URL.Path)
	}
}

// bodyWriter captures response body size
type bodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// PrometheusHandler serves the /metrics endpoint for Prometheus scraping
func PrometheusHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Println("Serving /metrics endpoint")
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	}
}