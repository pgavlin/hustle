package filter

import (
	"fmt"
	"time"

	"github.com/pgavlin/hustle/jq"
	logpkg "github.com/pgavlin/hustle/log"
)

// JQFilter is a compiled jq expression that can test LogRecords.
type JQFilter struct {
	node jq.Node
}

// Compile parses and compiles a jq expression.
func Compile(expr string) (*JQFilter, error) {
	node, errs := jq.Parse(expr)
	if len(errs) > 0 {
		return nil, fmt.Errorf("%s", errs[0])
	}
	return &JQFilter{node: node}, nil
}

// Match evaluates the jq expression against a LogRecord.
// Returns true if the expression produces any truthy (non-false, non-null) output.
// Returns false if the expression produces false, null, empty output, or errors.
func (f *JQFilter) Match(rec logpkg.LogRecord) bool {
	input := jq.ConcreteValue(recordToMap(rec))
	for v := range jq.Eval(f.node, input, jq.DefaultEnv()) {
		switch d := v.Data.(type) {
		case nil:
			return false
		case bool:
			return d
		default:
			return true
		}
	}
	return false
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
