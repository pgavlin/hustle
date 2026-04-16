package jq

import (
	"testing"
	"time"

	logpkg "github.com/pgavlin/hustle/log"
)

func TestInferShape_BasicFields(t *testing.T) {
	records := []logpkg.LogRecord{
		{Time: time.Now(), Level: "INFO", Msg: "hello", Attrs: logpkg.Attrs{{Key: "port", Value: float64(8080)}}},
	}
	shape := InferShape(records)
	obj, ok := shape.(ObjectShape)
	if !ok {
		t.Fatalf("expected ObjectShape, got %T", shape)
	}

	// Standard fields (may be wrapped in EnumShape)
	if _, ok := InnerShape(obj.Fields["time"]).(StringShape); !ok {
		t.Errorf("time should be StringShape, got %T", obj.Fields["time"])
	}
	if _, ok := InnerShape(obj.Fields["level"]).(StringShape); !ok {
		t.Errorf("level should be StringShape, got %T", obj.Fields["level"])
	}
	if _, ok := InnerShape(obj.Fields["msg"]).(StringShape); !ok {
		t.Errorf("msg should be StringShape, got %T", obj.Fields["msg"])
	}

	// Attr field
	if _, ok := InnerShape(obj.Fields["port"]).(NumberShape); !ok {
		t.Errorf("port should be NumberShape, got %T", obj.Fields["port"])
	}
}

func TestInferShape_NestedObject(t *testing.T) {
	records := []logpkg.LogRecord{
		{Attrs: logpkg.Attrs{{Key: "headers", Value: map[string]any{"content-type": "application/json"}}}},
	}
	shape := InferShape(records)
	obj := shape.(ObjectShape)
	headers, ok := InnerShape(obj.Fields["headers"]).(ObjectShape)
	if !ok {
		t.Fatalf("headers should be ObjectShape, got %T", InnerShape(obj.Fields["headers"]))
	}
	if _, ok := headers.Fields["content-type"].(StringShape); !ok {
		t.Error("content-type should be StringShape")
	}
}

func TestInferShape_MixedTypes(t *testing.T) {
	records := []logpkg.LogRecord{
		{Attrs: logpkg.Attrs{{Key: "value", Value: "hello"}}},
		{Attrs: logpkg.Attrs{{Key: "value", Value: float64(42)}}},
	}
	shape := InferShape(records)
	obj := shape.(ObjectShape)
	inner := InnerShape(obj.Fields["value"])
	if _, ok := inner.(UnionShape); !ok {
		t.Errorf("value inner should be UnionShape, got %T", inner)
	}
}

func TestInferShape_ArrayField(t *testing.T) {
	records := []logpkg.LogRecord{
		{Attrs: logpkg.Attrs{{Key: "tags", Value: []any{"a", "b", "c"}}}},
	}
	shape := InferShape(records)
	obj := shape.(ObjectShape)
	arr, ok := InnerShape(obj.Fields["tags"]).(ArrayShape)
	if !ok {
		t.Fatalf("tags should be ArrayShape, got %T", InnerShape(obj.Fields["tags"]))
	}
	if _, ok := arr.Element.(StringShape); !ok {
		t.Error("tags element should be StringShape")
	}
}

func TestInferShape_EmptyRecords(t *testing.T) {
	shape := InferShape(nil)
	obj, ok := shape.(ObjectShape)
	if !ok {
		t.Fatalf("expected ObjectShape, got %T", shape)
	}
	if len(obj.Fields) != 3 {
		t.Errorf("expected 3 fields, got %d", len(obj.Fields))
	}
}

func TestInferShape_MergesAcrossRecords(t *testing.T) {
	records := []logpkg.LogRecord{
		{Attrs: logpkg.Attrs{{Key: "port", Value: float64(8080)}}},
		{Attrs: logpkg.Attrs{{Key: "host", Value: "localhost"}}},
	}
	shape := InferShape(records)
	obj := shape.(ObjectShape)
	if _, ok := obj.Fields["port"]; !ok {
		t.Error("missing port field")
	}
	if _, ok := obj.Fields["host"]; !ok {
		t.Error("missing host field")
	}
}

func TestInferShape_EnumValues(t *testing.T) {
	records := []logpkg.LogRecord{
		{Level: "INFO", Msg: "a"},
		{Level: "ERROR", Msg: "b"},
		{Level: "WARN", Msg: "c"},
		{Level: "INFO", Msg: "d"},
	}
	shape := InferShape(records)
	obj := shape.(ObjectShape)

	levelShape := obj.Fields["level"]
	vals := EnumValues(levelShape)
	if vals == nil {
		t.Fatal("expected enum values for level")
	}

	valSet := map[any]bool{}
	for _, v := range vals {
		valSet[v] = true
	}
	for _, want := range []string{"INFO", "ERROR", "WARN"} {
		if !valSet[want] {
			t.Errorf("missing value %q in level enum", want)
		}
	}
	if len(vals) != 3 {
		t.Errorf("expected 3 distinct values, got %d: %v", len(vals), vals)
	}
}
