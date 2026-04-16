package log

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type LogfmtFormat struct{}

func (f *LogfmtFormat) Name() string { return "logfmt" }

func (f *LogfmtFormat) ParseRecord(line string) (LogRecord, error) {
	rec := LogRecord{
		RawJSON: line,
		Attrs:   make(Attrs, 0, 4),
	}

	found := false
	i := 0
	for i < len(line) {
		// Skip whitespace
		for i < len(line) && line[i] == ' ' {
			i++
		}
		if i >= len(line) {
			break
		}

		// Read key
		keyStart := i
		for i < len(line) && line[i] != '=' && line[i] != ' ' {
			i++
		}
		key := line[keyStart:i]

		if i >= len(line) || line[i] != '=' {
			continue
		}
		i++ // skip '='

		// Read value
		var value string
		if i < len(line) && line[i] == '"' {
			value, i = parseQuotedValue(line, i)
		} else {
			valStart := i
			for i < len(line) && line[i] != ' ' {
				i++
			}
			value = line[valStart:i]
		}

		found = true

		switch key {
		case "time", "ts", "timestamp":
			if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
				rec.Time = t
			} else if t, err := time.Parse(time.RFC3339, value); err == nil {
				rec.Time = t
			}
		case "level", "lvl", "severity":
			rec.Level = normalizeLevel(value)
		case "msg", "message":
			rec.Msg = value
		default:
			rec.Attrs.Set(key, inferValue(value))
		}
	}

	if !found {
		return LogRecord{}, fmt.Errorf("no key=value pairs found")
	}

	return rec, nil
}

// parseQuotedValue parses a quoted value starting at line[i] (which must be '"').
// Returns the unescaped value and the position after the closing quote.
// If there are no escape sequences, the returned string is a zero-copy slice
// of the input line.
func parseQuotedValue(line string, i int) (string, int) {
	i++ // skip opening quote
	start := i

	// Fast path: scan for closing quote without backslashes.
	for i < len(line) {
		if line[i] == '\\' {
			// Has escape sequences — fall through to slow path.
			goto slow
		}
		if line[i] == '"' {
			// No escapes — return a slice of the original string.
			value := line[start:i]
			return value, i + 1
		}
		i++
	}
	// Unterminated quote — return what we have.
	return line[start:i], i

slow:
	// Slow path: build unescaped string.
	var b strings.Builder
	b.WriteString(line[start:i]) // write everything before the first backslash
	for i < len(line) {
		if line[i] == '\\' && i+1 < len(line) {
			b.WriteByte(line[i+1])
			i += 2
		} else if line[i] == '"' {
			i++ // skip closing quote
			return b.String(), i
		} else {
			b.WriteByte(line[i])
			i++
		}
	}
	return b.String(), i
}

// inferValue tries to parse a string as a number or bool.
func inferValue(s string) any {
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	// Quick check: only attempt ParseFloat if the first byte looks numeric.
	// This avoids allocating an error object for non-numeric strings like
	// "GET", "usr_abc123", etc.
	if len(s) > 0 && looksNumeric(s[0]) {
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
	}
	return s
}

// looksNumeric returns true if c could be the start of a number.
func looksNumeric(c byte) bool {
	return (c >= '0' && c <= '9') || c == '-' || c == '+' || c == '.'
}

// parseLogfmtAttrs parses key=value pairs from s and adds them to attrs.
// Returns true if at least one pair was found.
func parseLogfmtAttrs(s string, attrs *Attrs) bool {
	found := false
	i := 0
	for i < len(s) {
		for i < len(s) && s[i] == ' ' {
			i++
		}
		if i >= len(s) {
			break
		}

		keyStart := i
		for i < len(s) && s[i] != '=' && s[i] != ' ' {
			i++
		}
		key := s[keyStart:i]

		if i >= len(s) || s[i] != '=' {
			continue
		}
		i++

		var value string
		if i < len(s) && s[i] == '"' {
			value, i = parseQuotedValue(s, i)
		} else {
			valStart := i
			for i < len(s) && s[i] != ' ' {
				i++
			}
			value = s[valStart:i]
		}

		attrs.Set(key, inferValue(value))
		found = true
	}
	return found
}
