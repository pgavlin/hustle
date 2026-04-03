package log

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type CLFFormat struct{}

func (f *CLFFormat) Name() string { return "clf" }

// CLF time format: [02/Jan/2006:15:04:05 -0700]
const clfTimeLayout = "02/Jan/2006:15:04:05 -0700"

// Combined Log Format regex
// Groups: 1=remote_addr, 2=ident, 3=user, 4=time, 5=request, 6=status, 7=bytes, 8=referer, 9=user_agent
var clfRegex = regexp.MustCompile(
	`^(\S+) (\S+) (\S+) \[([^\]]+)\] "([^"]*)" (\d{3}) (\S+)(?: "([^"]*)" "([^"]*)")?`,
)

func (f *CLFFormat) ParseRecord(line string) (LogRecord, error) {
	idx := clfRegex.FindStringSubmatchIndex(line)
	if idx == nil {
		return LogRecord{}, fmt.Errorf("not a CLF log line")
	}

	sub := func(n int) string {
		start, end := idx[2*n], idx[2*n+1]
		if start < 0 {
			return ""
		}
		return line[start:end]
	}

	rec := LogRecord{
		RawJSON: line,
		Attrs:   make(map[string]any, 8),
	}

	// Parse time
	if t, err := time.Parse(clfTimeLayout, sub(4)); err == nil {
		rec.Time = t
	}

	// Request line is the message
	rec.Msg = sub(5)

	// Status code determines level
	status, _ := strconv.Atoi(sub(6))
	switch {
	case status >= 500:
		rec.Level = "ERROR"
	case status >= 400:
		rec.Level = "WARN"
	default:
		rec.Level = "INFO"
	}

	// Parse request line into method, path, protocol
	parseRequestLine(sub(5), rec.Attrs)

	// Attrs
	rec.Attrs["remote_addr"] = sub(1)
	if u := sub(3); u != "-" {
		rec.Attrs["user"] = u
	}
	rec.Attrs["status"] = float64(status)
	if bytes, err := strconv.ParseFloat(sub(7), 64); err == nil {
		rec.Attrs["bytes"] = bytes
	}
	if r := sub(8); r != "" && r != "-" {
		rec.Attrs["referer"] = r
	}
	if ua := sub(9); ua != "" {
		rec.Attrs["user_agent"] = ua
	}

	return rec, nil
}

// parseRequestLine extracts method, path, protocol, and query parameters
// from an HTTP request line like "GET /api/v1/users?id=123 HTTP/1.1".
func parseRequestLine(reqLine string, attrs map[string]any) {
	parts := strings.SplitN(reqLine, " ", 3)
	if len(parts) < 2 {
		return
	}

	attrs["method"] = parts[0]

	rawPath := parts[1]
	if len(parts) >= 3 {
		attrs["protocol"] = parts[2]
	}

	// Parse path and query string
	u, err := url.ParseRequestURI(rawPath)
	if err != nil {
		attrs["path"] = rawPath
		return
	}

	attrs["path"] = u.Path

	// Extract query parameters as individual attributes
	if u.RawQuery != "" {
		params := u.Query()
		for k, v := range params {
			if len(v) == 1 {
				attrs["query."+k] = v[0]
			} else {
				attrs["query."+k] = strings.Join(v, ",")
			}
		}
	}
}
