package jq

import "sort"

// Shape represents the abstract structure of a jq value.
type Shape interface {
	shape()
}

type ObjectShape struct{ Fields map[string]Shape }
type ArrayShape struct{ Element Shape }
type StringShape struct{}
type NumberShape struct{}
type BoolShape struct{}
type NullShape struct{}
type AnyShape struct{}
type UnionShape struct{ Alternatives []Shape }

func (ObjectShape) shape() {}
func (ArrayShape) shape()  {}
func (StringShape) shape() {}
func (NumberShape) shape() {}
func (BoolShape) shape()   {}
func (NullShape) shape()   {}
func (AnyShape) shape()    {}
func (UnionShape) shape()  {}

// Value is the unified value type for both symbolic and concrete evaluation.
// When Unknown is true, only Shape is meaningful (symbolic mode).
// When Unknown is false, Data holds the concrete value.
type Value struct {
	Shape   Shape
	Data    any  // nil | bool | float64 | string | []Value | map[string]Value
	Unknown bool
}

// SymbolicValue creates an unknown Value with the given shape.
func SymbolicValue(s Shape) Value {
	return Value{Shape: s, Unknown: true}
}

// ConcreteValue creates a concrete Value from a Go value, inferring its shape.
func ConcreteValue(v any) Value {
	switch v := v.(type) {
	case nil:
		return Value{Shape: NullShape{}, Data: nil}
	case bool:
		return Value{Shape: BoolShape{}, Data: v}
	case float64:
		return Value{Shape: NumberShape{}, Data: v}
	case string:
		return Value{Shape: StringShape{}, Data: v}
	case map[string]any:
		fields := make(map[string]Shape, len(v))
		data := make(map[string]Value, len(v))
		for k, child := range v {
			cv := ConcreteValue(child)
			fields[k] = cv.Shape
			data[k] = cv
		}
		return Value{Shape: ObjectShape{Fields: fields}, Data: data}
	case []any:
		var elemShape Shape = AnyShape{}
		vals := make([]Value, len(v))
		for i, child := range v {
			cv := ConcreteValue(child)
			if i == 0 {
				elemShape = cv.Shape
			} else {
				elemShape = Merge(elemShape, cv.Shape)
			}
			vals[i] = cv
		}
		return Value{Shape: ArrayShape{Element: elemShape}, Data: vals}
	default:
		return Value{Shape: AnyShape{}, Data: v}
	}
}

// Truthy returns whether the value is truthy in jq semantics.
// false and null are falsy; everything else is truthy.
// Unknown values are considered truthy (for symbolic evaluation).
func (v Value) Truthy() bool {
	if v.Unknown {
		return true
	}
	switch d := v.Data.(type) {
	case nil:
		return false
	case bool:
		return d
	default:
		return true
	}
}

// FieldNames returns sorted field names from an ObjectShape, or nil for other shapes.
func FieldNames(s Shape) []string {
	s = unwrapUnion(s)
	switch s := s.(type) {
	case ObjectShape:
		names := make([]string, 0, len(s.Fields))
		for k := range s.Fields {
			names = append(names, k)
		}
		sort.Strings(names)
		return names
	case UnionShape:
		// Collect field names from all object alternatives
		seen := map[string]bool{}
		for _, alt := range s.Alternatives {
			for _, name := range FieldNames(alt) {
				seen[name] = true
			}
		}
		if len(seen) == 0 {
			return nil
		}
		names := make([]string, 0, len(seen))
		for k := range seen {
			names = append(names, k)
		}
		sort.Strings(names)
		return names
	default:
		return nil
	}
}

// LookupField looks up a field in a shape. Returns AnyShape if not found or non-object.
func LookupField(s Shape, name string) Shape {
	s = unwrapUnion(s)
	switch s := s.(type) {
	case ObjectShape:
		if f, ok := s.Fields[name]; ok {
			return f
		}
		return AnyShape{}
	case UnionShape:
		var alts []Shape
		for _, alt := range s.Alternatives {
			f := LookupField(alt, name)
			if _, ok := f.(AnyShape); !ok {
				alts = append(alts, f)
			}
		}
		if len(alts) == 0 {
			return AnyShape{}
		}
		if len(alts) == 1 {
			return alts[0]
		}
		return UnionShape{Alternatives: alts}
	case AnyShape:
		return AnyShape{}
	default:
		return AnyShape{}
	}
}

// Merge combines two shapes. For ObjectShapes, merges field maps.
// For mismatched types, produces a UnionShape.
func Merge(a, b Shape) Shape {
	if _, ok := a.(AnyShape); ok {
		return b
	}
	if _, ok := b.(AnyShape); ok {
		return a
	}

	switch a := a.(type) {
	case ObjectShape:
		if b, ok := b.(ObjectShape); ok {
			fields := make(map[string]Shape, len(a.Fields)+len(b.Fields))
			for k, v := range a.Fields {
				fields[k] = v
			}
			for k, v := range b.Fields {
				if existing, ok := fields[k]; ok {
					fields[k] = Merge(existing, v)
				} else {
					fields[k] = v
				}
			}
			return ObjectShape{Fields: fields}
		}
	case ArrayShape:
		if b, ok := b.(ArrayShape); ok {
			return ArrayShape{Element: Merge(a.Element, b.Element)}
		}
	case StringShape:
		if _, ok := b.(StringShape); ok {
			return StringShape{}
		}
	case NumberShape:
		if _, ok := b.(NumberShape); ok {
			return NumberShape{}
		}
	case BoolShape:
		if _, ok := b.(BoolShape); ok {
			return BoolShape{}
		}
	case NullShape:
		if _, ok := b.(NullShape); ok {
			return NullShape{}
		}
	}

	return union(a, b)
}

func union(shapes ...Shape) Shape {
	var flat []Shape
	for _, s := range shapes {
		if u, ok := s.(UnionShape); ok {
			flat = append(flat, u.Alternatives...)
		} else {
			flat = append(flat, s)
		}
	}
	if len(flat) == 1 {
		return flat[0]
	}
	return UnionShape{Alternatives: flat}
}

func unwrapUnion(s Shape) Shape {
	if u, ok := s.(UnionShape); ok && len(u.Alternatives) == 1 {
		return u.Alternatives[0]
	}
	return s
}
