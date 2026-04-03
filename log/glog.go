package log

import (
	"fmt"
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

func (f *GlogFormat) ParseRecord(line string) (LogRecord, error) {
	if len(line) < 2 {
		return LogRecord{}, fmt.Errorf("not a glog log line")
	}

	// Level: first character must be I, W, E, or F
	level, ok := glogLevels[line[0]]
	if !ok {
		return LogRecord{}, fmt.Errorf("not a glog log line")
	}

	// Date: digits immediately after level char
	i := 1
	dateStart := i
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	dateLen := i - dateStart
	if dateLen != 4 && dateLen != 8 {
		return LogRecord{}, fmt.Errorf("not a glog log line")
	}
	datePart := line[dateStart:i]

	// Space
	if i >= len(line) || line[i] != ' ' {
		return LogRecord{}, fmt.Errorf("not a glog log line")
	}
	i++

	// Time: hh:mm:ss.uuuuuu
	timeStart := i
	for i < len(line) && line[i] != ' ' {
		i++
	}
	timePart := line[timeStart:i]
	if len(timePart) < 8 { // at least hh:mm:ss
		return LogRecord{}, fmt.Errorf("not a glog log line")
	}

	// Skip whitespace before thread ID
	for i < len(line) && line[i] == ' ' {
		i++
	}

	// Thread ID: digits
	tidStart := i
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	tidStr := line[tidStart:i]
	if len(tidStr) == 0 {
		return LogRecord{}, fmt.Errorf("not a glog log line")
	}

	// Space
	if i >= len(line) || line[i] != ' ' {
		return LogRecord{}, fmt.Errorf("not a glog log line")
	}
	i++

	// File: everything up to ':'
	fileStart := i
	for i < len(line) && line[i] != ':' {
		i++
	}
	if i >= len(line) {
		return LogRecord{}, fmt.Errorf("not a glog log line")
	}
	file := line[fileStart:i]
	i++ // skip ':'

	// Line number: digits up to ']'
	lineNoStart := i
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	lineNoStr := line[lineNoStart:i]

	// Expect '] '
	if i >= len(line) || line[i] != ']' {
		return LogRecord{}, fmt.Errorf("not a glog log line")
	}
	i++ // skip ']'
	if i < len(line) && line[i] == ' ' {
		i++ // skip space after ]
	}

	// Rest is the message
	msg := line[i:]

	// Build record — glog always has 3 base attrs
	rec := LogRecord{
		RawJSON: line,
		Level:   level,
		Msg:     msg,
		Time:    parseGlogTime(datePart, timePart),
		Attrs:   make(map[string]any, 4),
	}

	if tid, err := strconv.ParseFloat(tidStr, 64); err == nil {
		rec.Attrs["thread_id"] = tid
	}
	rec.Attrs["file"] = file
	if lineNo, err := strconv.ParseFloat(lineNoStr, 64); err == nil {
		rec.Attrs["line"] = lineNo
	}

	// Check for klog structured message
	parseKlogStructuredMessage(&rec)

	return rec, nil
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
