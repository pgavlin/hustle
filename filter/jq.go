package filter

import (
	"time"

	"github.com/itchyny/gojq"

	logpkg "github.com/pgavlin/hustle/log"
)

// JQFilter is a compiled jq expression that can test LogRecords.
type JQFilter struct {
	code *gojq.Code
}

// Compile parses and compiles a jq expression.
func Compile(expr string) (*JQFilter, error) {
	query, err := gojq.Parse(expr)
	if err != nil {
		return nil, err
	}
	code, err := gojq.Compile(query)
	if err != nil {
		return nil, err
	}
	return &JQFilter{code: code}, nil
}

// Match evaluates the jq expression against a LogRecord.
// Returns true if the expression produces any truthy (non-false, non-null) output.
// Returns false if the expression produces false, null, empty output, or errors.
func (f *JQFilter) Match(rec logpkg.LogRecord) bool {
	input := recordToMap(rec)
	iter := f.code.Run(input)
	v, ok := iter.Next()
	if !ok {
		return false
	}
	if _, isErr := v.(error); isErr {
		return false
	}
	switch v := v.(type) {
	case nil:
		return false
	case bool:
		return v
	default:
		return true
	}
}

func recordToMap(rec logpkg.LogRecord) map[string]any {
	m := make(map[string]any, 3+len(rec.Attrs))
	m["time"] = rec.Time.Format(time.RFC3339Nano)
	m["level"] = rec.Level
	m["msg"] = rec.Msg
	for k, v := range rec.Attrs {
		m[k] = v
	}
	return m
}
