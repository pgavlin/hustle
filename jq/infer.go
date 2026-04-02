package jq

import (
	logpkg "github.com/pgavlin/hustle/log"
)

// InferShape builds a Shape from loaded LogRecords by scanning all records
// and merging their structures. This mirrors the recordToMap format used by
// the filter package: time, level, msg at top level plus all attr keys.
func InferShape(records []logpkg.LogRecord) Shape {
	base := ObjectShape{Fields: map[string]Shape{
		"time":  StringShape{},
		"level": StringShape{},
		"msg":   StringShape{},
	}}

	for _, rec := range records {
		for k, v := range rec.Attrs {
			s := inferValue(v)
			if existing, ok := base.Fields[k]; ok {
				base.Fields[k] = Merge(existing, s)
			} else {
				base.Fields[k] = s
			}
		}
	}

	return base
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
