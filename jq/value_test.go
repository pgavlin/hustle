package jq

import (
	"testing"
)

func TestFieldNames_ObjectShape(t *testing.T) {
	s := ObjectShape{Fields: map[string]Shape{
		"time": StringShape{}, "level": StringShape{}, "msg": StringShape{},
	}}
	names := FieldNames(s)
	want := []string{"level", "msg", "time"}
	if len(names) != len(want) {
		t.Fatalf("got %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("names[%d] = %q, want %q", i, names[i], want[i])
		}
	}
}

func TestFieldNames_NonObject(t *testing.T) {
	if names := FieldNames(StringShape{}); names != nil {
		t.Errorf("expected nil, got %v", names)
	}
	if names := FieldNames(AnyShape{}); names != nil {
		t.Errorf("expected nil for AnyShape, got %v", names)
	}
}

func TestFieldNames_UnionWithObjects(t *testing.T) {
	s := UnionShape{Alternatives: []Shape{
		ObjectShape{Fields: map[string]Shape{"a": StringShape{}}},
		ObjectShape{Fields: map[string]Shape{"b": NumberShape{}}},
	}}
	names := FieldNames(s)
	if len(names) != 2 || names[0] != "a" || names[1] != "b" {
		t.Errorf("got %v, want [a b]", names)
	}
}

func TestLookupField_Exists(t *testing.T) {
	s := ObjectShape{Fields: map[string]Shape{"foo": NumberShape{}}}
	if _, ok := LookupField(s, "foo").(NumberShape); !ok {
		t.Error("expected NumberShape for foo")
	}
}

func TestLookupField_Missing(t *testing.T) {
	s := ObjectShape{Fields: map[string]Shape{"foo": NumberShape{}}}
	if _, ok := LookupField(s, "bar").(AnyShape); !ok {
		t.Error("expected AnyShape for missing field")
	}
}

func TestLookupField_AnyShape(t *testing.T) {
	if _, ok := LookupField(AnyShape{}, "foo").(AnyShape); !ok {
		t.Error("expected AnyShape for lookup on AnyShape")
	}
}

func TestMerge_SameObjectShapes(t *testing.T) {
	a := ObjectShape{Fields: map[string]Shape{"x": StringShape{}}}
	b := ObjectShape{Fields: map[string]Shape{"y": NumberShape{}}}
	merged := Merge(a, b)
	obj, ok := merged.(ObjectShape)
	if !ok {
		t.Fatalf("expected ObjectShape, got %T", merged)
	}
	if len(obj.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(obj.Fields))
	}
}

func TestMerge_DifferentTypes(t *testing.T) {
	merged := Merge(StringShape{}, NumberShape{})
	if _, ok := merged.(UnionShape); !ok {
		t.Errorf("expected UnionShape, got %T", merged)
	}
}

func TestMerge_AnyAbsorbs(t *testing.T) {
	merged := Merge(AnyShape{}, StringShape{})
	if _, ok := merged.(StringShape); !ok {
		t.Errorf("expected StringShape when merging AnyShape, got %T", merged)
	}
	merged = Merge(NumberShape{}, AnyShape{})
	if _, ok := merged.(NumberShape); !ok {
		t.Errorf("expected NumberShape when merging with AnyShape, got %T", merged)
	}
}

func TestConcreteValue_Object(t *testing.T) {
	v := ConcreteValue(map[string]any{"foo": "bar", "n": float64(42)})
	obj, ok := v.Shape.(ObjectShape)
	if !ok {
		t.Fatalf("expected ObjectShape, got %T", v.Shape)
	}
	if _, ok := obj.Fields["foo"].(StringShape); !ok {
		t.Error("foo should be StringShape")
	}
	if _, ok := obj.Fields["n"].(NumberShape); !ok {
		t.Error("n should be NumberShape")
	}
}

func TestConcreteValue_Array(t *testing.T) {
	v := ConcreteValue([]any{"a", "b", "c"})
	arr, ok := v.Shape.(ArrayShape)
	if !ok {
		t.Fatalf("expected ArrayShape, got %T", v.Shape)
	}
	if _, ok := arr.Element.(StringShape); !ok {
		t.Error("element should be StringShape")
	}
}

func TestTruthy(t *testing.T) {
	tests := []struct {
		v    Value
		want bool
	}{
		{ConcreteValue(nil), false},
		{ConcreteValue(false), false},
		{ConcreteValue(true), true},
		{ConcreteValue(float64(0)), true},
		{ConcreteValue(""), true},
		{SymbolicValue(AnyShape{}), true},
	}
	for _, tt := range tests {
		if got := tt.v.Truthy(); got != tt.want {
			t.Errorf("Truthy(%v) = %v, want %v", tt.v.Data, got, tt.want)
		}
	}
}
