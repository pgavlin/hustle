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
	pairs, err := parseLogfmt(line)
	if err != nil {
		return LogRecord{}, err
	}
	if len(pairs) == 0 {
		return LogRecord{}, fmt.Errorf("no key=value pairs found")
	}

	rec := LogRecord{
		RawJSON: line,
		Attrs:   make(map[string]any, len(pairs)),
	}

	for _, p := range pairs {
		switch p.key {
		case "time", "ts", "timestamp":
			if t, err := time.Parse(time.RFC3339Nano, p.value); err == nil {
				rec.Time = t
			} else if t, err := time.Parse(time.RFC3339, p.value); err == nil {
				rec.Time = t
			}
		case "level", "lvl", "severity":
			rec.Level = normalizeLevel(p.value)
		case "msg", "message":
			rec.Msg = p.value
		default:
			rec.Attrs[p.key] = inferValue(p.value)
		}
	}

	return rec, nil
}

type kvPair struct {
	key, value string
}

func parseLogfmt(line string) ([]kvPair, error) {
	var pairs []kvPair
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
			// Bare word without =, skip
			continue
		}
		i++ // skip '='

		// Read value
		var value string
		if i < len(line) && line[i] == '"' {
			// Quoted value
			i++ // skip opening quote
			var b strings.Builder
			for i < len(line) {
				if line[i] == '\\' && i+1 < len(line) {
					b.WriteByte(line[i+1])
					i += 2
				} else if line[i] == '"' {
					i++ // skip closing quote
					break
				} else {
					b.WriteByte(line[i])
					i++
				}
			}
			value = b.String()
		} else {
			// Unquoted value
			valStart := i
			for i < len(line) && line[i] != ' ' {
				i++
			}
			value = line[valStart:i]
		}

		pairs = append(pairs, kvPair{key: key, value: value})
	}
	return pairs, nil
}

// inferValue tries to parse a string as a number or bool.
func inferValue(s string) any {
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}
