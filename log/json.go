package log

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/segmentio/encoding/json"
)

var tokPool = sync.Pool{
	New: func() any { return json.NewTokenizer(nil) },
}

// JSONFormat parses JSON log lines, auto-detecting common field name variants
// for time, level, and message across slog, zap, zerolog, logrus, bunyan, and docker.
type JSONFormat struct{}

func (f *JSONFormat) Name() string { return "json" }

func (f *JSONFormat) ParseRecord(line string) (LogRecord, error) {
	b := unsafe.Slice(unsafe.StringData(line), len(line))

	rec := LogRecord{
		RawJSON: line,
		Attrs:   make(Attrs, 0, 4),
	}

	tok := tokPool.Get().(*json.Tokenizer)
	tok.Reset(b)
	defer tokPool.Put(tok)

	if !tok.Next() || tok.Delim != '{' {
		return LogRecord{}, fmt.Errorf("expected JSON object")
	}

	for tok.Next() {
		if tok.Delim == '}' {
			break
		}

		// Skip non-key tokens (commas, colons)
		if tok.Delim != 0 {
			continue
		}

		if !tok.IsKey {
			continue
		}

		// Get the key as a zero-copy string.
		keyBytes := tok.String()
		key := unsafe.String(unsafe.SliceData(keyBytes), len(keyBytes))

		// Advance to the value.
		if !tok.Next() {
			break
		}
		// Skip the colon.
		if tok.Delim == ':' {
			if !tok.Next() {
				break
			}
		}

		// Check if this is a known field.
		switch key {
		case "time", "ts", "timestamp", "@timestamp":
			if rec.Time.IsZero() {
				rec.Time = parseTokenTime(tok)
			} else {
				rec.Attrs.Set(key, tokenToAny(tok))
			}

		case "level", "severity", "lvl":
			if rec.Level == "" {
				rec.Level = parseTokenLevel(tok)
			} else {
				rec.Attrs.Set(key, tokenToAny(tok))
			}

		case "msg", "message", "log":
			if rec.Msg == "" {
				if tok.Delim == 0 && tok.Kind() >= json.String {
					s := tok.String()
					rec.Msg = unsafe.String(unsafe.SliceData(s), len(s))
				}
			} else {
				rec.Attrs.Set(key, tokenToAny(tok))
			}

		default:
			rec.Attrs.Set(key, tokenToAny(tok))
		}
	}

	if tok.Err != nil {
		return LogRecord{}, tok.Err
	}

	return rec, nil
}

// parseTokenTime extracts a time value from the current token.
func parseTokenTime(tok *json.Tokenizer) time.Time {
	switch {
	case tok.Kind() >= json.String:
		s := tok.String()
		return parseTimeBytes(s)
	case tok.Kind() >= json.Num:
		f := tok.Float()
		sec, frac := math.Modf(f)
		return time.Unix(int64(sec), int64(frac*1e9))
	}
	return time.Time{}
}

// parseTokenLevel extracts a level string from the current token.
func parseTokenLevel(tok *json.Tokenizer) string {
	switch {
	case tok.Kind() >= json.String:
		s := tok.String()
		return normalizeLevel(unsafe.String(unsafe.SliceData(s), len(s)))
	case tok.Kind() >= json.Num:
		f := tok.Float()
		// Bunyan numeric levels
		switch {
		case f <= 10:
			return "TRACE"
		case f <= 20:
			return "DEBUG"
		case f <= 30:
			return "INFO"
		case f <= 40:
			return "WARN"
		case f <= 50:
			return "ERROR"
		default:
			return "FATAL"
		}
	}
	return ""
}

// tokenToAny converts the current token value to a Go value.
// For nested objects/arrays, it falls back to json.Parse for that subtree.
func tokenToAny(tok *json.Tokenizer) any {
	switch {
	case tok.Delim == '{' || tok.Delim == '[':
		// Nested object or array — find the extent and decode with json.Parse.
		return decodeNested(tok)
	case tok.Kind() >= json.String:
		s := tok.String()
		return unsafe.String(unsafe.SliceData(s), len(s))
	case tok.Kind() == json.True:
		return true
	case tok.Kind() == json.False:
		return false
	case tok.Kind() == json.Null:
		return nil
	case tok.Kind() >= json.Num:
		// Try integer first, fall back to float.
		s := unsafe.String(unsafe.SliceData(tok.Value), len(tok.Value))
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return float64(i)
		}
		return tok.Float()
	}
	return nil
}

// decodeNested decodes a nested object or array from the tokenizer.
// It captures the raw JSON bytes of the subtree and decodes them with json.Parse.
// Nested values in log records are uncommon, so the extra cost here is acceptable.
func decodeNested(tok *json.Tokenizer) any {
	// The Value field contains the opening delimiter. We need to find the
	// matching close by tracking depth.
	startDepth := tok.Depth

	// Capture the start position. The raw bytes from Value start through
	// the close delimiter give us the full subtree.
	// We'll collect by scanning until depth returns to startDepth.
	start := tok.Value

	var end json.RawValue
	for tok.Next() {
		end = tok.Value
		if tok.Depth == startDepth && (tok.Delim == '}' || tok.Delim == ']') {
			break
		}
	}

	if len(start) == 0 || len(end) == 0 {
		return nil
	}

	// Compute the subtree bytes from the backing array.
	// start and end both point into the same underlying input slice.
	startPtr := uintptr(unsafe.Pointer(unsafe.SliceData(start)))
	endPtr := uintptr(unsafe.Pointer(unsafe.SliceData(end))) + uintptr(len(end))
	length := int(endPtr - startPtr)
	subtree := unsafe.Slice((*byte)(unsafe.Pointer(startPtr)), length)

	var result any
	json.Parse(subtree, &result, json.DontCopyString|json.DontCopyNumber)
	return result
}

// parseTimeBytes parses common time formats from a byte slice.
func parseTimeBytes(s []byte) time.Time {
	str := unsafe.String(unsafe.SliceData(s), len(s))
	if t, err := time.Parse(time.RFC3339Nano, str); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, str); err == nil {
		return t
	}
	return time.Time{}
}

// parseTime handles both string (RFC3339) and numeric (unix timestamp) time values.
func parseTime(v any) time.Time {
	switch v := v.(type) {
	case string:
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			return t
		}
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t
		}
	case float64:
		sec, frac := math.Modf(v)
		return time.Unix(int64(sec), int64(frac*1e9))
	}
	return time.Time{}
}

func isUpperASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 'a' && s[i] <= 'z' {
			return false
		}
	}
	return true
}

// commonLevels maps lowercase (and mixed-case) level strings to their canonical
// uppercase forms. Avoids strings.ToUpper allocations for the common case.
var commonLevels = map[string]string{
	"trace":   "TRACE",
	"debug":   "DEBUG",
	"info":    "INFO",
	"warn":    "WARN",
	"warning": "WARNING",
	"error":   "ERROR",
	"fatal":   "FATAL",
	"panic":   "PANIC",
}

// normalizeLevel returns the canonical uppercase form of a level string,
// avoiding an allocation for the common case.
func normalizeLevel(s string) string {
	if isUpperASCII(s) {
		return s
	}
	if l, ok := commonLevels[s]; ok {
		return l
	}
	return strings.ToUpper(s)
}

// parseLevel normalizes level values to uppercase strings.
// Handles string levels and bunyan numeric levels.
func parseLevel(v any) string {
	switch v := v.(type) {
	case string:
		return normalizeLevel(v)
	case float64:
		switch {
		case v <= 10:
			return "TRACE"
		case v <= 20:
			return "DEBUG"
		case v <= 30:
			return "INFO"
		case v <= 40:
			return "WARN"
		case v <= 50:
			return "ERROR"
		default:
			return "FATAL"
		}
	}
	return ""
}
