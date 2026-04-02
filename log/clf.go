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
	m := clfRegex.FindStringSubmatch(line)
	if m == nil {
		return LogRecord{}, fmt.Errorf("not a CLF log line")
	}

	rec := LogRecord{
		RawJSON: line,
		Attrs:   make(map[string]any),
	}

	// Parse time
	if t, err := time.Parse(clfTimeLayout, m[4]); err == nil {
		rec.Time = t
	}

	// Request line is the message
	rec.Msg = m[5]

	// Status code determines level
	status, _ := strconv.Atoi(m[6])
	switch {
	case status >= 500:
		rec.Level = "ERROR"
	case status >= 400:
		rec.Level = "WARN"
	default:
		rec.Level = "INFO"
	}

	// Parse request line into method, path, protocol
	parseRequestLine(m[5], rec.Attrs)

	// Attrs
	rec.Attrs["remote_addr"] = m[1]
	if m[3] != "-" {
		rec.Attrs["user"] = m[3]
	}
	rec.Attrs["status"] = float64(status)
	if bytes, err := strconv.ParseFloat(m[7], 64); err == nil {
		rec.Attrs["bytes"] = bytes
	}
	if m[8] != "" && m[8] != "-" {
		rec.Attrs["referer"] = m[8]
	}
	if m[9] != "" {
		rec.Attrs["user_agent"] = m[9]
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
