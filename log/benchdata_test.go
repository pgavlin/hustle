package log

import (
	"bytes"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strings"
	"time"
)

// benchSeed is the fixed PRNG seed for deterministic benchmark data.
const benchSeed = 42

// generateGlogLines generates n deterministic glog-format log lines.
// Each line is ~200-500 bytes, producing roughly n*300 bytes total.
func generateGlogLines(n int) []string {
	rng := rand.New(rand.NewPCG(benchSeed, 0))
	ts := time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC)
	lines := make([]string, n)

	for i := range n {
		lvl := pickLevel(rng)
		ts = ts.Add(time.Duration(rng.IntN(500)+1) * time.Millisecond)
		tid := 1000 + rng.IntN(200)
		file, line := pickFileLocation(rng)
		msg := pickMessage(rng)

		// Build: I0615 08:00:00.123456 1042 server.go:123] message key=value ...
		var b strings.Builder
		fmt.Fprintf(&b, "%c%02d%02d %02d:%02d:%02d.%06d %5d %s:%d] %s",
			lvl, ts.Month(), ts.Day(),
			ts.Hour(), ts.Minute(), ts.Second(), ts.Nanosecond()/1000,
			tid, file, line, msg)

		// Add structured key=value attrs
		nAttrs := rng.IntN(6)
		for j := range nAttrs {
			_ = j
			k, v := pickAttr(rng)
			fmt.Fprintf(&b, " %s=%s", k, v)
		}

		lines[i] = b.String()
	}
	return lines
}

// generateJSONLines generates n deterministic slog JSON log lines
// using log/slog's JSONHandler for authentic output.
func generateJSONLines(n int) []string {
	rng := rand.New(rand.NewPCG(benchSeed, 0))
	ts := time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
		// Use a fixed time via ReplaceAttr so slog doesn't use wall clock.
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			return a
		},
	}))

	lines := make([]string, n)
	for i := range n {
		lvl := slogLevel(rng)
		ts = ts.Add(time.Duration(rng.IntN(500)+1) * time.Millisecond)
		msg := pickMessage(rng)

		attrs := make([]slog.Attr, 0, 6)
		nAttrs := rng.IntN(6)
		for range nAttrs {
			k, v := pickAttr(rng)
			attrs = append(attrs, slog.String(k, v))
		}

		buf.Reset()
		args := make([]any, len(attrs))
		for j, a := range attrs {
			args[j] = a
		}
		record := slog.NewRecord(ts, lvl, msg, 0)
		record.AddAttrs(attrs...)
		logger.Handler().Handle(nil, record)

		// slog writes a trailing newline; trim it
		line := strings.TrimRight(buf.String(), "\n")
		lines[i] = line
	}
	return lines
}

// generateLogfmtLines generates n deterministic logfmt log lines.
func generateLogfmtLines(n int) []string {
	rng := rand.New(rand.NewPCG(benchSeed, 0))
	ts := time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC)
	lines := make([]string, n)

	for i := range n {
		lvl := jsonLevel(rng)
		ts = ts.Add(time.Duration(rng.IntN(500)+1) * time.Millisecond)
		msg := pickMessage(rng)

		var b strings.Builder
		fmt.Fprintf(&b, `time=%s level=%s msg="%s"`,
			ts.Format(time.RFC3339Nano), strings.ToLower(lvl), msg)

		nAttrs := rng.IntN(6)
		for range nAttrs {
			k, v := pickAttr(rng)
			if strings.ContainsAny(v, " \"") {
				fmt.Fprintf(&b, ` %s="%s"`, k, v)
			} else {
				fmt.Fprintf(&b, ` %s=%s`, k, v)
			}
		}

		lines[i] = b.String()
	}
	return lines
}

// --- Pickers ---

var glogLevelChars = []byte{'I', 'I', 'I', 'I', 'I', 'I', 'W', 'W', 'E', 'F'}

func pickLevel(rng *rand.Rand) byte {
	return glogLevelChars[rng.IntN(len(glogLevelChars))]
}

var jsonLevels = []string{"DEBUG", "INFO", "INFO", "INFO", "INFO", "WARN", "WARN", "ERROR"}

func jsonLevel(rng *rand.Rand) string {
	return jsonLevels[rng.IntN(len(jsonLevels))]
}

var slogLevels = []slog.Level{
	slog.LevelDebug,
	slog.LevelInfo, slog.LevelInfo, slog.LevelInfo, slog.LevelInfo,
	slog.LevelWarn, slog.LevelWarn,
	slog.LevelError,
}

func slogLevel(rng *rand.Rand) slog.Level {
	return slogLevels[rng.IntN(len(slogLevels))]
}

var messages = []string{
	"Starting server",
	"Request received",
	"Processing request",
	"Database query executed",
	"Cache hit",
	"Cache miss",
	"Connection established",
	"Connection closed",
	"Authentication successful",
	"Authentication failed",
	"Rate limit exceeded",
	"Retrying request",
	"Timeout waiting for response",
	"Configuration reloaded",
	"Health check passed",
	"Health check failed",
	"Deployment started",
	"Deployment completed",
	"Rolling back deployment",
	"Certificate renewal scheduled",
	"DNS resolution failed",
	"Upstream service unavailable",
	"Circuit breaker opened",
	"Circuit breaker closed",
	"Garbage collection completed",
	"Memory pressure detected",
	"Disk usage warning",
	"Log rotation triggered",
	"Backup completed successfully",
	"Scheduled job executed",
	"Message published to queue",
	"Message consumed from queue",
	"WebSocket connection opened",
	"WebSocket connection closed",
	"gRPC call completed",
	"Metrics exported",
	"Trace span recorded",
	"Feature flag evaluated",
	"A/B test variant assigned",
	"User session created",
}

func pickMessage(rng *rand.Rand) string {
	return messages[rng.IntN(len(messages))]
}

var sourceFiles = []string{
	"server.go", "handler.go", "middleware.go", "db.go", "cache.go",
	"auth.go", "router.go", "config.go", "metrics.go", "health.go",
	"worker.go", "queue.go", "grpc.go", "websocket.go", "retry.go",
}

func pickFileLocation(rng *rand.Rand) (string, int) {
	return sourceFiles[rng.IntN(len(sourceFiles))], rng.IntN(500) + 10
}

type attrGen struct {
	key    string
	values []string
}

var attrGens = []attrGen{
	{"method", []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"}},
	{"path", []string{"/api/v1/users", "/api/v1/orders", "/api/v1/products", "/health", "/metrics", "/api/v2/search"}},
	{"status", []string{"200", "201", "204", "301", "400", "401", "403", "404", "500", "502", "503"}},
	{"duration_ms", []string{"1", "5", "12", "42", "150", "500", "1200", "3500"}},
	{"user_id", []string{"usr_abc123", "usr_def456", "usr_ghi789", "usr_jkl012", "usr_mno345"}},
	{"trace_id", []string{"4bf92f3577b34da6a3ce929d0e0e4736", "7b3bf470f3b84e6ab2e8f1a0c4d5e6f7", "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6"}},
	{"service", []string{"api-gateway", "user-service", "order-service", "payment-service", "notification-service"}},
	{"region", []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}},
	{"host", []string{"web-01.prod", "web-02.prod", "web-03.prod", "worker-01.prod", "worker-02.prod"}},
	{"version", []string{"v1.2.3", "v1.2.4", "v1.3.0", "v2.0.0-rc1"}},
	{"cache_key", []string{"users:123:profile", "orders:456:summary", "products:789:detail", "sessions:abc:data"}},
	{"queue", []string{"email-notifications", "order-processing", "analytics-events", "audit-log"}},
	{"error_code", []string{"TIMEOUT", "CONNECTION_REFUSED", "INVALID_TOKEN", "RATE_LIMITED", "INTERNAL"}},
	{"retry_count", []string{"0", "1", "2", "3", "5"}},
	{"bytes", []string{"128", "512", "1024", "4096", "16384", "65536", "1048576"}},
}

func pickAttr(rng *rand.Rand) (string, string) {
	gen := attrGens[rng.IntN(len(attrGens))]
	return gen.key, gen.values[rng.IntN(len(gen.values))]
}

