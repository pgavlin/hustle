package log

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// GlogFormat parses glog/klog format log lines.
// Format: Lmmdd hh:mm:ss.uuuuuu threadid file:line] message
// L is a single letter: I=INFO, W=WARNING, E=ERROR, F=FATAL
type GlogFormat struct{}

func (f *GlogFormat) Name() string { return "glog" }

var glogLevels = map[byte]string{
	'I': "INFO",
	'W': "WARNING",
	'E': "ERROR",
	'F': "FATAL",
}

// Matches both mmdd and yyyymmdd date variants.
var glogRegex = regexp.MustCompile(
	`^([IWEF])(\d{4,8}) (\d{2}:\d{2}:\d{2}\.\d{1,6})\s+(\d+) ([^:]+):(\d+)\] (.*)`,
)

func (f *GlogFormat) ParseRecord(line string) (LogRecord, error) {
	m := glogRegex.FindStringSubmatch(line)
	if m == nil {
		return LogRecord{}, fmt.Errorf("not a glog log line")
	}

	rec := LogRecord{
		RawJSON: line,
		Attrs:   make(map[string]any),
	}

	// Level
	rec.Level = glogLevels[m[1][0]]

	// Time: parse date (mmdd or yyyymmdd) + time
	rec.Time = parseGlogTime(m[2], m[3])

	// Thread ID
	if tid, err := strconv.ParseFloat(m[4], 64); err == nil {
		rec.Attrs["thread_id"] = tid
	}

	// Source location
	rec.Attrs["file"] = m[5]
	if lineNo, err := strconv.ParseFloat(m[6], 64); err == nil {
		rec.Attrs["line"] = lineNo
	}

	// Message — check for klog structured format: "quoted msg" key=val key=val
	rec.Msg = m[7]
	parseKlogStructuredMessage(&rec)

	return rec, nil
}

// parseKlogStructuredMessage checks if the message follows klog's InfoS/ErrorS
// format: a quoted string followed by logfmt-style key=value pairs.
// If so, extracts the message and merges the pairs into Attrs.
func parseKlogStructuredMessage(rec *LogRecord) {
	msg := rec.Msg
	if len(msg) == 0 || msg[0] != '"' {
		return
	}

	// Find the closing quote (handle escaped quotes)
	i := 1
	for i < len(msg) {
		if msg[i] == '\\' && i+1 < len(msg) {
			i += 2
			continue
		}
		if msg[i] == '"' {
			break
		}
		i++
	}
	if i >= len(msg) {
		return // no closing quote
	}

	// Extract the quoted message
	quotedMsg := msg[1:i]
	rest := msg[i+1:]

	// Parse the rest as logfmt key=value pairs
	if len(rest) == 0 {
		rec.Msg = quotedMsg
		return
	}

	pairs, err := parseLogfmt(rest)
	if err != nil || len(pairs) == 0 {
		return // not structured — keep original message
	}

	rec.Msg = quotedMsg
	for _, p := range pairs {
		rec.Attrs[p.key] = inferValue(p.value)
	}
}

func parseGlogTime(datePart, timePart string) time.Time {
	var year, month, day int

	switch len(datePart) {
	case 4: // mmdd — use current year
		month, _ = strconv.Atoi(datePart[:2])
		day, _ = strconv.Atoi(datePart[2:4])
		year = time.Now().Year()
	case 8: // yyyymmdd
		year, _ = strconv.Atoi(datePart[:4])
		month, _ = strconv.Atoi(datePart[4:6])
		day, _ = strconv.Atoi(datePart[6:8])
	default:
		return time.Time{}
	}

	// Parse time part: hh:mm:ss.uuuuuu
	t, err := time.Parse("15:04:05.000000", padMicroseconds(timePart))
	if err != nil {
		return time.Time{}
	}

	return time.Date(year, time.Month(month), day,
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond(),
		time.Local)
}

// padMicroseconds ensures the fractional part is exactly 6 digits.
func padMicroseconds(timePart string) string {
	dotIdx := -1
	for i, c := range timePart {
		if c == '.' {
			dotIdx = i
			break
		}
	}
	if dotIdx < 0 {
		return timePart + ".000000"
	}
	frac := timePart[dotIdx+1:]
	for len(frac) < 6 {
		frac += "0"
	}
	return timePart[:dotIdx+1] + frac[:6]
}
