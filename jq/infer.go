package jq

import (
	"fmt"

	logpkg "github.com/pgavlin/hustle/log"
)

// maxEnumValues is the maximum number of distinct values to track for enum detection.
const maxEnumValues = 20

// InferShape builds a Shape from loaded LogRecords by scanning all records
// and merging their structures. This mirrors the recordToMap format used by
// the filter package: time, level, msg at top level plus all attr keys.
// Fields with a small number of distinct values are wrapped in EnumShape.
func InferShape(records []logpkg.LogRecord) Shape {
	// Track distinct values per top-level field.
	// nil means cardinality exceeded the threshold.
	fieldValues := map[string]map[string]bool{}

	base := ObjectShape{Fields: map[string]Shape{
		"time":  StringShape{},
		"level": StringShape{},
		"msg":   StringShape{},
	}}

	for _, rec := range records {
		// Track level values (small cardinality, useful for enum).
		// Skip time and msg — always high cardinality.
		trackValue(fieldValues, "level", rec.Level)

		for k, v := range rec.Attrs {
			s := inferValue(v)
			if existing, ok := base.Fields[k]; ok {
				base.Fields[k] = Merge(InnerShape(existing), s)
			} else {
				base.Fields[k] = s
			}
			trackValue(fieldValues, k, fmt.Sprint(v))
		}
	}

	// Wrap fields with small value sets in EnumShape
	for k, vals := range fieldValues {
		if vals == nil {
			continue // cardinality exceeded threshold
		}
		inner := base.Fields[k]
		values := make([]any, 0, len(vals))
		for v := range vals {
			values = append(values, v)
		}
		base.Fields[k] = EnumShape{Inner: inner, Values: values}
	}

	return base
}

func trackValue(fieldValues map[string]map[string]bool, key, value string) {
	if fieldValues[key] == nil {
		fieldValues[key] = map[string]bool{}
	}
	vals := fieldValues[key]
	if vals == nil {
		return // already exceeded threshold
	}
	vals[value] = true
	if len(vals) > maxEnumValues {
		fieldValues[key] = nil // too many — stop tracking
	}
}

func inferValue(v any) Shape {
	switch v := v.(type) {
	case string:
		return StringShape{}
	case float64:
		return NumberShape{}
	case bool:
		return BoolShape{}
	case nil:
		return NullShape{}
	case map[string]any:
		fields := make(map[string]Shape, len(v))
		for k, child := range v {
			fields[k] = inferValue(child)
		}
		return ObjectShape{Fields: fields}
	case []any:
		if len(v) == 0 {
			return ArrayShape{Element: AnyShape{}}
		}
		elem := inferValue(v[0])
		for _, child := range v[1:] {
			elem = Merge(elem, inferValue(child))
		}
		return ArrayShape{Element: elem}
	default:
		return AnyShape{}
	}
}
