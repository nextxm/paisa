package server

import (
	"context"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const (
	perfHeaderLatencyMS = "X-Paisa-Perf-Latency-Ms"
	perfHeaderSQLCount  = "X-Paisa-Perf-SQL-Count"
	perfHeaderSQLTimeMS = "X-Paisa-Perf-SQL-Time-Ms"
)

type requestTelemetry struct {
	start    time.Time
	sqlCount atomic.Int64
	sqlNanos atomic.Int64
}

func beginRequestTelemetry(db *gorm.DB) (*gorm.DB, *requestTelemetry) {
	telemetry := &requestTelemetry{start: time.Now()}
	requestDB := db.Session(&gorm.Session{
		Logger: telemetryLogger{
			delegate: db.Logger,
			stats:    telemetry,
		},
	})
	return requestDB, telemetry
}

func (t *requestTelemetry) writeHeaders(c *gin.Context) {
	c.Header(perfHeaderLatencyMS, strconv.FormatInt(time.Since(t.start).Milliseconds(), 10))
	c.Header(perfHeaderSQLCount, strconv.FormatInt(t.sqlCount.Load(), 10))
	c.Header(perfHeaderSQLTimeMS, strconv.FormatFloat(float64(t.sqlNanos.Load())/float64(time.Millisecond), 'f', 3, 64))
}

type telemetryLogger struct {
	delegate gormlogger.Interface
	stats    *requestTelemetry
}

func (l telemetryLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return telemetryLogger{
		delegate: l.delegate.LogMode(level),
		stats:    l.stats,
	}
}

func (l telemetryLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.delegate.Info(ctx, msg, data...)
}

func (l telemetryLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.delegate.Warn(ctx, msg, data...)
}

func (l telemetryLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.delegate.Error(ctx, msg, data...)
}

func (l telemetryLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	l.stats.sqlCount.Add(1)
	l.stats.sqlNanos.Add(time.Since(begin).Nanoseconds())
	l.delegate.Trace(ctx, begin, fc, err)
}
