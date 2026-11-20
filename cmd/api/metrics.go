package main

import (
	"expvar"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type dbStats struct {
	AcquireCount            int64   `json:"acquire_count"`
	AcquireDurationMs       float64 `json:"acquire_duration_ms"`
	AcquiredConns           int32   `json:"acquired_conns"`
	CanceledAcquireCount    int64   `json:"canceled_acquire_count"`
	ConstructingConns       int32   `json:"constructing_conns"`
	EmptyAcquireCount       int64   `json:"empty_acquire_count"`
	EmptyAcquireWaitTimeMs  float64 `json:"empty_acquire_wait_time_ms"`
	IdleConns               int32   `json:"idle_conns"`
	MaxConns                int32   `json:"max_conns"`
	TotalConns              int32   `json:"total_conns"`
	NewConnsCount           int64   `json:"new_conns_count"`
	MaxLifetimeDestroyCount int64   `json:"max_lifetime_destroy_count"`
	MaxIdleDestroyCount     int64   `json:"max_idle_destroy_count"`
}

func newDBStats(s *pgxpool.Stat) dbStats {
	return dbStats{
		AcquireCount:            s.AcquireCount(),
		AcquireDurationMs:       toMs(s.AcquireDuration()),
		AcquiredConns:           s.AcquiredConns(),
		CanceledAcquireCount:    s.CanceledAcquireCount(),
		ConstructingConns:       s.ConstructingConns(),
		EmptyAcquireCount:       s.EmptyAcquireCount(),
		EmptyAcquireWaitTimeMs:  toMs(s.EmptyAcquireWaitTime()),
		IdleConns:               s.IdleConns(),
		MaxConns:                s.MaxConns(),
		TotalConns:              s.TotalConns(),
		NewConnsCount:           s.NewConnsCount(),
		MaxLifetimeDestroyCount: s.MaxLifetimeDestroyCount(),
		MaxIdleDestroyCount:     s.MaxIdleDestroyCount(),
	}
}

func toMs(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

func publishMetrics(db *pgxpool.Pool) {
	expvar.NewString("version").Set(version)
	expvar.Publish("goroutines", expvar.Func(func() any { return runtime.NumGoroutine() }))
	expvar.Publish("database", expvar.Func(func() any { return newDBStats(db.Stat()) }))
	expvar.Publish("timestamp", expvar.Func(func() any { return time.Now().UTC() }))
}

func (app *application) RecordRequestMetrics(next http.Handler) http.Handler {
	var (
		totalRequestsReceived           = expvar.NewInt("total_requests_received")
		totalResponsesSent              = expvar.NewInt("total_responses_sent")
		totalProcessingTimeMicroseconds = expvar.NewInt("total_processing_time_μs")
		totalResponsesSentByStatus      = expvar.NewMap("total_responses_sent_by_status")
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		totalRequestsReceived.Add(1)

		mw := newMetricsResponseWritter(w)

		next.ServeHTTP(mw, r)

		totalResponsesSent.Add(1)
		totalResponsesSentByStatus.Add(strconv.Itoa(mw.statusCode), 1)
		duration := time.Since(start).Microseconds()
		totalProcessingTimeMicroseconds.Add(duration)
	})
}

type metricsResponseWriter struct {
	wrapped       http.ResponseWriter
	statusCode    int
	headerWritten bool
}

func newMetricsResponseWritter(w http.ResponseWriter) *metricsResponseWriter {
	return &metricsResponseWriter{
		wrapped:    w,
		statusCode: 200,
	}
}

func (mw *metricsResponseWriter) Header() http.Header {
	return mw.wrapped.Header()
}

func (mw *metricsResponseWriter) Write(b []byte) (int, error) {
	mw.headerWritten = true
	return mw.wrapped.Write(b)
}

func (mw *metricsResponseWriter) WriteHeader(statusCode int) {
	mw.wrapped.WriteHeader(statusCode)

	if !mw.headerWritten {
		mw.statusCode = statusCode
		mw.headerWritten = true
	}
}

func (mw *metricsResponseWriter) Unwrap() http.ResponseWriter {
	return mw.wrapped
}
