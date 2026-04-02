package jq

import (
	"testing"
	"time"

	logpkg "github.com/pgavlin/hustle/log"
)

func TestInferShape_BasicFields(t *testing.T) {
	records := []logpkg.LogRecord{
		{Time: time.Now(), Level: "INFO", Msg: "hello", Attrs: map[string]any{"port": float64(8080)}},
	}
	shape := InferShape(records)
	obj, ok := shape.(ObjectShape)
	if !ok {
		t.Fatalf("expected ObjectShape, got %T", shape)
	}

	// Standard fields
	if _, ok := obj.Fields["time"].(StringShape); !ok {
		t.Error("time should be StringShape")
	}
	if _, ok := obj.Fields["level"].(StringShape); !ok {
		t.Error("level should be StringShape")
	}
	if _, ok := obj.Fields["msg"].(StringShape); !ok {
		t.Error("msg should be StringShape")
	}

	// Attr field
	if _, ok := obj.Fields["port"].(NumberShape); !ok {
		t.Error("port should be NumberShape")
	}
}

func TestInferShape_NestedObject(t *testing.T) {
	records := []logpkg.LogRecord{
		{Attrs: map[string]any{
			"headers": map[string]any{"content-type": "application/json"},
		}},
	}
	shape := InferShape(records)
	obj := shape.(ObjectShape)
	headers, ok := obj.Fields["headers"].(ObjectShape)
	if !ok {
		t.Fatalf("headers should be ObjectShape, got %T", obj.Fields["headers"])
	}
	if _, ok := headers.Fields["content-type"].(StringShape); !ok {
		t.Error("content-type should be StringShape")
	}
}

func TestInferShape_MixedTypes(t *testing.T) {
	records := []logpkg.LogRecord{
		{Attrs: map[string]any{"value": "hello"}},
		{Attrs: map[string]any{"value": float64(42)}},
	}
	shape := InferShape(records)
	obj := shape.(ObjectShape)
	if _, ok := obj.Fields["value"].(UnionShape); !ok {
		t.Errorf("value should be UnionShape, got %T", obj.Fields["value"])
	}
}

func TestInferShape_ArrayField(t *testing.T) {
	records := []logpkg.LogRecord{
		{Attrs: map[string]any{"tags": []any{"a", "b", "c"}}},
	}
	shape := InferShape(records)
	obj := shape.(ObjectShape)
	arr, ok := obj.Fields["tags"].(ArrayShape)
	if !ok {
		t.Fatalf("tags should be ArrayShape, got %T", obj.Fields["tags"])
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
	// Should still have time, level, msg
	if len(obj.Fields) != 3 {
		t.Errorf("expected 3 fields, got %d", len(obj.Fields))
	}
}

func TestInferShape_MergesAcrossRecords(t *testing.T) {
	records := []logpkg.LogRecord{
		{Attrs: map[string]any{"port": float64(8080)}},
		{Attrs: map[string]any{"host": "localhost"}},
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
