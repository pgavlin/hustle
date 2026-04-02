package jq

import (
	"encoding/json"
	"fmt"
	"iter"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var regexpCache sync.Map // string → *regexp.Regexp

func cachedCompileRegexp(pattern string) (*regexp.Regexp, error) {
	if v, ok := regexpCache.Load(pattern); ok {
		return v.(*regexp.Regexp), nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	regexpCache.Store(pattern, re)
	return re, nil
}

func registerBuiltins(env *Environment) {
	// Filters
	env.builtins["select"] = Builtin{Eval: builtinSelect}
	env.builtins["empty"] = Builtin{Eval: builtinEmpty}

	// Type functions
	env.builtins["type"] = Builtin{Eval: builtinType}
	env.builtins["length"] = Builtin{Eval: builtinLength}
	env.builtins["keys"] = Builtin{Eval: builtinKeys}
	env.builtins["keys_unsorted"] = Builtin{Eval: builtinKeysUnsorted}
	env.builtins["values"] = Builtin{Eval: builtinValues}
	env.builtins["has"] = Builtin{Eval: builtinHas}
	env.builtins["contains"] = Builtin{Eval: builtinContains}
	env.builtins["to_entries"] = Builtin{Eval: builtinToEntries}
	env.builtins["from_entries"] = Builtin{Eval: builtinFromEntries}
	env.builtins["with_entries"] = Builtin{Eval: builtinWithEntries}

	// String functions
	env.builtins["test"] = Builtin{Eval: builtinTest}
	env.builtins["startswith"] = Builtin{Eval: builtinStartswith}
	env.builtins["endswith"] = Builtin{Eval: builtinEndswith}
	env.builtins["split"] = Builtin{Eval: builtinSplit}
	env.builtins["join"] = Builtin{Eval: builtinJoin}
	env.builtins["ascii_downcase"] = Builtin{Eval: builtinAsciiDowncase}
	env.builtins["ascii_upcase"] = Builtin{Eval: builtinAsciiUpcase}
	env.builtins["ltrimstr"] = Builtin{Eval: builtinLtrimstr}
	env.builtins["rtrimstr"] = Builtin{Eval: builtinRtrimstr}
	env.builtins["tostring"] = Builtin{Eval: builtinTostring}
	env.builtins["tonumber"] = Builtin{Eval: builtinTonumber}

	// Array functions
	env.builtins["map"] = Builtin{Eval: builtinMap}
	env.builtins["map_values"] = Builtin{Eval: builtinMapValues}
	env.builtins["sort"] = Builtin{Eval: builtinSort}
	env.builtins["sort_by"] = Builtin{Eval: builtinSortBy}
	env.builtins["reverse"] = Builtin{Eval: builtinReverse}
	env.builtins["group_by"] = Builtin{Eval: builtinGroupBy}
	env.builtins["unique"] = Builtin{Eval: builtinUnique}
	env.builtins["unique_by"] = Builtin{Eval: builtinUniqueBy}
	env.builtins["flatten"] = Builtin{Eval: builtinFlatten}
	env.builtins["first"] = Builtin{Eval: builtinFirst}
	env.builtins["last"] = Builtin{Eval: builtinLast}
	env.builtins["range"] = Builtin{Eval: builtinRange}
	env.builtins["limit"] = Builtin{Eval: builtinLimit}
	env.builtins["add"] = Builtin{Eval: builtinAdd}
	env.builtins["any"] = Builtin{Eval: builtinAny}
	env.builtins["all"] = Builtin{Eval: builtinAll}
	env.builtins["min"] = Builtin{Eval: builtinMin}
	env.builtins["max"] = Builtin{Eval: builtinMax}
	env.builtins["min_by"] = Builtin{Eval: builtinMinBy}
	env.builtins["max_by"] = Builtin{Eval: builtinMaxBy}
	env.builtins["indices"] = Builtin{Eval: builtinIndices}
	env.builtins["index"] = Builtin{Eval: builtinIndex}
	env.builtins["rindex"] = Builtin{Eval: builtinRindex}
	env.builtins["inside"] = Builtin{Eval: builtinInside}
	env.builtins["nth"] = Builtin{Eval: builtinNth}

	// Conversion
	env.builtins["tojson"] = Builtin{Eval: builtinTojson}
	env.builtins["fromjson"] = Builtin{Eval: builtinFromjson}

	// Other
	env.builtins["not"] = Builtin{Eval: builtinNot}
	env.builtins["null"] = Builtin{Eval: builtinNull}
	env.builtins["true"] = Builtin{Eval: builtinTrue}
	env.builtins["false"] = Builtin{Eval: builtinFalse}
	env.builtins["debug"] = Builtin{Eval: builtinDebug}
	env.builtins["error"] = Builtin{Eval: builtinError}
	env.builtins["path"] = Builtin{Eval: builtinPath}
	env.builtins["getpath"] = Builtin{Eval: builtinGetpath}
	env.builtins["env"] = Builtin{Eval: builtinEnv}
	env.builtins["input"] = Builtin{Eval: builtinInput}
	env.builtins["recurse"] = Builtin{Eval: builtinRecurse}
	env.builtins["recurse_down"] = Builtin{Eval: builtinRecurse}
	env.builtins["objects"] = Builtin{Eval: builtinObjects}
	env.builtins["arrays"] = Builtin{Eval: builtinArrays}
	env.builtins["strings"] = Builtin{Eval: builtinStrings}
	env.builtins["numbers"] = Builtin{Eval: builtinNumbers}
	env.builtins["booleans"] = Builtin{Eval: builtinBooleans}
	env.builtins["nulls"] = Builtin{Eval: builtinNulls}
	env.builtins["iterables"] = Builtin{Eval: builtinIterables}
	env.builtins["scalars"] = Builtin{Eval: builtinScalars}
	env.builtins["leaf_paths"] = Builtin{Eval: builtinLeafPaths}
	env.builtins["infinite"] = Builtin{Eval: builtinInfinite}
	env.builtins["nan"] = Builtin{Eval: builtinNan}
	env.builtins["isinfinite"] = Builtin{Eval: builtinIsinfinite}
	env.builtins["isnan"] = Builtin{Eval: builtinIsnan}
	env.builtins["isnormal"] = Builtin{Eval: builtinIsnormal}
	env.builtins["in"] = Builtin{Eval: builtinIn}
	env.builtins["ascii"] = Builtin{Eval: builtinAscii}
	env.builtins["explode"] = Builtin{Eval: builtinExplode}
	env.builtins["implode"] = Builtin{Eval: builtinImplode}
	env.builtins["abs"] = Builtin{Eval: builtinAbs}
	env.builtins["floor"] = Builtin{Eval: builtinFloor}
	env.builtins["ceil"] = Builtin{Eval: builtinCeil}
	env.builtins["round"] = Builtin{Eval: builtinRound}
	env.builtins["sqrt"] = Builtin{Eval: builtinSqrt}
	env.builtins["pow"] = Builtin{Eval: builtinPow}
	env.builtins["log"] = Builtin{Eval: builtinLog}
	env.builtins["log2"] = Builtin{Eval: builtinLog2}
	env.builtins["fabs"] = Builtin{Eval: builtinAbs}
}

// ---- Filters ----

func builtinSelect(input Value, args []Node, env *Environment) iter.Seq[Value] {
	return func(yield func(Value) bool) {
		if len(args) == 0 {
			yield(input)
			return
		}
		if input.Unknown {
			// Symbolic: always yield input (shape preserved).
			yield(input)
			return
		}
		for v := range Eval(args[0], input, env) {
			if v.Truthy() {
				yield(input)
			}
			return
		}
	}
}

func builtinEmpty(_ Value, _ []Node, _ *Environment) iter.Seq[Value] {
	return none()
}

// ---- Type functions ----

func builtinType(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: StringShape{}, Unknown: true})
	}
	var typeName string
	switch input.Data.(type) {
	case nil:
		typeName = "null"
	case bool:
		typeName = "boolean"
	case float64:
		typeName = "number"
	case string:
		typeName = "string"
	case []Value:
		typeName = "array"
	case map[string]Value:
		typeName = "object"
	default:
		typeName = "unknown"
	}
	return one(Value{Shape: StringShape{}, Data: typeName})
}

func builtinLength(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: NumberShape{}, Unknown: true})
	}
	var length float64
	switch v := input.Data.(type) {
	case nil:
		length = 0
	case string:
		length = float64(len([]rune(v)))
	case []Value:
		length = float64(len(v))
	case map[string]Value:
		length = float64(len(v))
	case bool:
		// jq: length of booleans is an error, but we return null for robustness.
		return one(Value{Shape: NullShape{}, Data: nil})
	case float64:
		length = math.Abs(v)
	default:
		return one(Value{Shape: NullShape{}, Data: nil})
	}
	return one(Value{Shape: NumberShape{}, Data: length})
}

func builtinKeys(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		switch s := input.Shape.(type) {
		case ObjectShape:
			// Return known keys as an array.
			return one(Value{Shape: ArrayShape{Element: StringShape{}}, Unknown: true, Data: sortedKeys(s)})
		case ArrayShape:
			return one(Value{Shape: ArrayShape{Element: NumberShape{}}, Unknown: true})
		default:
			return one(Value{Shape: ArrayShape{Element: StringShape{}}, Unknown: true})
		}
	}

	switch v := input.Data.(type) {
	case map[string]Value:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		vals := make([]Value, len(keys))
		for i, k := range keys {
			vals[i] = Value{Shape: StringShape{}, Data: k}
		}
		return one(Value{Shape: ArrayShape{Element: StringShape{}}, Data: vals})
	case []Value:
		vals := make([]Value, len(v))
		for i := range v {
			vals[i] = Value{Shape: NumberShape{}, Data: float64(i)}
		}
		return one(Value{Shape: ArrayShape{Element: NumberShape{}}, Data: vals})
	default:
		return none()
	}
}

func sortedKeys(s ObjectShape) []Value {
	keys := make([]string, 0, len(s.Fields))
	for k := range s.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	vals := make([]Value, len(keys))
	for i, k := range keys {
		vals[i] = Value{Shape: StringShape{}, Data: k}
	}
	return vals
}

func builtinKeysUnsorted(input Value, args []Node, env *Environment) iter.Seq[Value] {
	// Same as keys but without sorting for objects.
	if input.Unknown {
		return builtinKeys(input, args, env)
	}
	switch v := input.Data.(type) {
	case map[string]Value:
		vals := make([]Value, 0, len(v))
		for k := range v {
			vals = append(vals, Value{Shape: StringShape{}, Data: k})
		}
		return one(Value{Shape: ArrayShape{Element: StringShape{}}, Data: vals})
	default:
		return builtinKeys(input, args, env)
	}
}

func builtinValues(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		switch s := input.Shape.(type) {
		case ObjectShape:
			var shapes []Shape
			for _, fs := range s.Fields {
				shapes = append(shapes, fs)
			}
			var elemShape Shape = AnyShape{}
			if len(shapes) == 1 {
				elemShape = shapes[0]
			} else if len(shapes) > 1 {
				elemShape = union(shapes...)
			}
			return one(Value{Shape: ArrayShape{Element: elemShape}, Unknown: true})
		case ArrayShape:
			return one(Value{Shape: s, Unknown: true})
		default:
			return one(Value{Shape: ArrayShape{Element: AnyShape{}}, Unknown: true})
		}
	}

	switch v := input.Data.(type) {
	case map[string]Value:
		// Sort keys for deterministic output.
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		vals := make([]Value, len(keys))
		var elemShape Shape = AnyShape{}
		for i, k := range keys {
			vals[i] = v[k]
			if i == 0 {
				elemShape = vals[i].Shape
			} else {
				elemShape = Merge(elemShape, vals[i].Shape)
			}
		}
		return one(Value{Shape: ArrayShape{Element: elemShape}, Data: vals})
	case []Value:
		return one(input)
	default:
		return none()
	}
}

func builtinHas(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: BoolShape{}, Unknown: true})
	}
	return func(yield func(Value) bool) {
		if len(args) == 0 {
			return
		}
		for kv := range Eval(args[0], input, env) {
			switch data := input.Data.(type) {
			case map[string]Value:
				if k, ok := kv.Data.(string); ok {
					_, exists := data[k]
					yield(Value{Shape: BoolShape{}, Data: exists})
				}
			case []Value:
				if idx, ok := kv.Data.(float64); ok {
					i := int(idx)
					yield(Value{Shape: BoolShape{}, Data: i >= 0 && i < len(data)})
				}
			default:
				yield(Value{Shape: BoolShape{}, Data: false})
			}
			return
		}
	}
}

func builtinContains(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: BoolShape{}, Unknown: true})
	}
	return func(yield func(Value) bool) {
		if len(args) == 0 {
			return
		}
		for other := range Eval(args[0], input, env) {
			yield(Value{Shape: BoolShape{}, Data: valueContains(input, other)})
			return
		}
	}
}

func valueContains(a, b Value) bool {
	switch av := a.Data.(type) {
	case string:
		if bv, ok := b.Data.(string); ok {
			return strings.Contains(av, bv)
		}
	case []Value:
		if bv, ok := b.Data.([]Value); ok {
			for _, be := range bv {
				found := false
				for _, ae := range av {
					if valueContains(ae, be) {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
			return true
		}
	case map[string]Value:
		if bv, ok := b.Data.(map[string]Value); ok {
			for k, bval := range bv {
				aval, ok := av[k]
				if !ok || !valueContains(aval, bval) {
					return false
				}
			}
			return true
		}
	default:
		return valueEqual(a, b)
	}
	return valueEqual(a, b)
}

func builtinToEntries(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		switch s := input.Shape.(type) {
		case ObjectShape:
			// Compute value shape as union of all field shapes.
			var valShapes []Shape
			for _, fs := range s.Fields {
				valShapes = append(valShapes, fs)
			}
			var valShape Shape = AnyShape{}
			if len(valShapes) == 1 {
				valShape = valShapes[0]
			} else if len(valShapes) > 1 {
				valShape = union(valShapes...)
			}
			entryShape := ObjectShape{Fields: map[string]Shape{
				"key":   StringShape{},
				"value": valShape,
			}}
			return one(Value{Shape: ArrayShape{Element: entryShape}, Unknown: true})
		default:
			entryShape := ObjectShape{Fields: map[string]Shape{
				"key":   StringShape{},
				"value": AnyShape{},
			}}
			return one(Value{Shape: ArrayShape{Element: entryShape}, Unknown: true})
		}
	}

	data, ok := input.Data.(map[string]Value)
	if !ok {
		return none()
	}

	// Sort keys for deterministic output.
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	entries := make([]Value, len(keys))
	for i, k := range keys {
		entryData := map[string]Value{
			"key":   {Shape: StringShape{}, Data: k},
			"value": data[k],
		}
		entryShape := ObjectShape{Fields: map[string]Shape{
			"key":   StringShape{},
			"value": data[k].Shape,
		}}
		entries[i] = Value{Shape: entryShape, Data: entryData}
	}

	var elemShape Shape
	if len(entries) > 0 {
		elemShape = entries[0].Shape
		for _, e := range entries[1:] {
			elemShape = Merge(elemShape, e.Shape)
		}
	} else {
		elemShape = ObjectShape{Fields: map[string]Shape{
			"key":   StringShape{},
			"value": AnyShape{},
		}}
	}

	return one(Value{Shape: ArrayShape{Element: elemShape}, Data: entries})
}

func builtinFromEntries(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: AnyShape{}, Unknown: true})
	}

	arr, ok := input.Data.([]Value)
	if !ok {
		return none()
	}

	result := make(map[string]Value)
	fields := make(map[string]Shape)
	for _, entry := range arr {
		entryData, ok := entry.Data.(map[string]Value)
		if !ok {
			continue
		}
		var key string
		if kv, ok := entryData["key"]; ok {
			if ks, ok := kv.Data.(string); ok {
				key = ks
			} else if kn, ok := kv.Data.(float64); ok {
				key = fmt.Sprintf("%g", kn)
			}
		} else if kv, ok := entryData["name"]; ok {
			if ks, ok := kv.Data.(string); ok {
				key = ks
			}
		}
		if key == "" {
			continue
		}
		val := entryData["value"]
		result[key] = val
		fields[key] = val.Shape
	}

	return one(Value{Shape: ObjectShape{Fields: fields}, Data: result})
}

func builtinWithEntries(input Value, args []Node, env *Environment) iter.Seq[Value] {
	// with_entries(f) is equivalent to [to_entries[] | f] | from_entries
	return func(yield func(Value) bool) {
		if len(args) == 0 {
			yield(input)
			return
		}

		// First get to_entries result.
		for entries := range builtinToEntries(input, nil, env) {
			if entries.Unknown {
				yield(Value{Shape: AnyShape{}, Unknown: true})
				return
			}

			arr, ok := entries.Data.([]Value)
			if !ok {
				return
			}

			// Apply f to each entry.
			var mapped []Value
			for _, entry := range arr {
				for v := range Eval(args[0], entry, env) {
					mapped = append(mapped, v)
				}
			}

			// from_entries on the result.
			mappedVal := Value{
				Shape: ArrayShape{Element: AnyShape{}},
				Data:  mapped,
			}
			for v := range builtinFromEntries(mappedVal, nil, env) {
				if !yield(v) {
					return
				}
			}
			return
		}
	}
}

// ---- String functions ----

func builtinTest(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: BoolShape{}, Unknown: true})
	}
	return func(yield func(Value) bool) {
		if len(args) == 0 {
			return
		}
		s, ok := input.Data.(string)
		if !ok {
			return
		}
		for pv := range Eval(args[0], input, env) {
			pat, ok := pv.Data.(string)
			if !ok {
				return
			}
			re, err := cachedCompileRegexp(pat)
			if err != nil {
				yield(Value{Shape: BoolShape{}, Data: false})
				return
			}
			yield(Value{Shape: BoolShape{}, Data: re.MatchString(s)})
			return
		}
	}
}

func builtinStartswith(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: BoolShape{}, Unknown: true})
	}
	return func(yield func(Value) bool) {
		if len(args) == 0 {
			return
		}
		s, ok := input.Data.(string)
		if !ok {
			return
		}
		for pv := range Eval(args[0], input, env) {
			prefix, ok := pv.Data.(string)
			if !ok {
				return
			}
			yield(Value{Shape: BoolShape{}, Data: strings.HasPrefix(s, prefix)})
			return
		}
	}
}

func builtinEndswith(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: BoolShape{}, Unknown: true})
	}
	return func(yield func(Value) bool) {
		if len(args) == 0 {
			return
		}
		s, ok := input.Data.(string)
		if !ok {
			return
		}
		for pv := range Eval(args[0], input, env) {
			suffix, ok := pv.Data.(string)
			if !ok {
				return
			}
			yield(Value{Shape: BoolShape{}, Data: strings.HasSuffix(s, suffix)})
			return
		}
	}
}

func builtinSplit(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: ArrayShape{Element: StringShape{}}, Unknown: true})
	}
	return func(yield func(Value) bool) {
		if len(args) == 0 {
			return
		}
		s, ok := input.Data.(string)
		if !ok {
			return
		}
		for sv := range Eval(args[0], input, env) {
			sep, ok := sv.Data.(string)
			if !ok {
				return
			}
			parts := strings.Split(s, sep)
			vals := make([]Value, len(parts))
			for i, p := range parts {
				vals[i] = Value{Shape: StringShape{}, Data: p}
			}
			yield(Value{Shape: ArrayShape{Element: StringShape{}}, Data: vals})
			return
		}
	}
}

func builtinJoin(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: StringShape{}, Unknown: true})
	}
	return func(yield func(Value) bool) {
		if len(args) == 0 {
			return
		}
		arr, ok := input.Data.([]Value)
		if !ok {
			return
		}
		for sv := range Eval(args[0], input, env) {
			sep, ok := sv.Data.(string)
			if !ok {
				return
			}
			parts := make([]string, len(arr))
			for i, v := range arr {
				switch d := v.Data.(type) {
				case string:
					parts[i] = d
				case nil:
					parts[i] = ""
				default:
					parts[i] = fmt.Sprintf("%v", d)
				}
			}
			yield(Value{Shape: StringShape{}, Data: strings.Join(parts, sep)})
			return
		}
	}
}

func builtinAsciiDowncase(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: StringShape{}, Unknown: true})
	}
	if s, ok := input.Data.(string); ok {
		return one(Value{Shape: StringShape{}, Data: strings.ToLower(s)})
	}
	return none()
}

func builtinAsciiUpcase(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: StringShape{}, Unknown: true})
	}
	if s, ok := input.Data.(string); ok {
		return one(Value{Shape: StringShape{}, Data: strings.ToUpper(s)})
	}
	return none()
}

func builtinLtrimstr(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: StringShape{}, Unknown: true})
	}
	return func(yield func(Value) bool) {
		if len(args) == 0 {
			yield(input)
			return
		}
		s, ok := input.Data.(string)
		if !ok {
			yield(input)
			return
		}
		for pv := range Eval(args[0], input, env) {
			prefix, ok := pv.Data.(string)
			if !ok {
				yield(input)
				return
			}
			yield(Value{Shape: StringShape{}, Data: strings.TrimPrefix(s, prefix)})
			return
		}
	}
}

func builtinRtrimstr(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: StringShape{}, Unknown: true})
	}
	return func(yield func(Value) bool) {
		if len(args) == 0 {
			yield(input)
			return
		}
		s, ok := input.Data.(string)
		if !ok {
			yield(input)
			return
		}
		for pv := range Eval(args[0], input, env) {
			suffix, ok := pv.Data.(string)
			if !ok {
				yield(input)
				return
			}
			yield(Value{Shape: StringShape{}, Data: strings.TrimSuffix(s, suffix)})
			return
		}
	}
}

func builtinTostring(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: StringShape{}, Unknown: true})
	}
	if s, ok := input.Data.(string); ok {
		return one(Value{Shape: StringShape{}, Data: s})
	}
	return one(Value{Shape: StringShape{}, Data: valueToJSON(input)})
}

func builtinTonumber(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: NumberShape{}, Unknown: true})
	}
	switch v := input.Data.(type) {
	case float64:
		return one(input)
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return none()
		}
		return one(Value{Shape: NumberShape{}, Data: f})
	default:
		return none()
	}
}

// ---- Array functions ----

func builtinMap(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if len(args) == 0 {
		return one(input)
	}

	if input.Unknown {
		// Symbolic: evaluate f on the element shape.
		switch s := input.Shape.(type) {
		case ArrayShape:
			elemInput := Value{Shape: s.Element, Unknown: true}
			var resultShape Shape = AnyShape{}
			for v := range Eval(args[0], elemInput, env) {
				resultShape = v.Shape
				break
			}
			return one(Value{Shape: ArrayShape{Element: resultShape}, Unknown: true})
		default:
			return one(Value{Shape: ArrayShape{Element: AnyShape{}}, Unknown: true})
		}
	}

	return func(yield func(Value) bool) {
		arr, ok := input.Data.([]Value)
		if !ok {
			return
		}
		var results []Value
		var elemShape Shape = AnyShape{}
		first := true
		for _, elem := range arr {
			for v := range Eval(args[0], elem, env) {
				results = append(results, v)
				if first {
					elemShape = v.Shape
					first = false
				} else {
					elemShape = Merge(elemShape, v.Shape)
				}
			}
		}
		if results == nil {
			results = []Value{}
		}
		yield(Value{Shape: ArrayShape{Element: elemShape}, Data: results})
	}
}

func builtinMapValues(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if len(args) == 0 {
		return one(input)
	}

	if input.Unknown {
		switch s := input.Shape.(type) {
		case ObjectShape:
			fields := make(map[string]Shape, len(s.Fields))
			for k, fs := range s.Fields {
				elemInput := Value{Shape: fs, Unknown: true}
				for v := range Eval(args[0], elemInput, env) {
					fields[k] = v.Shape
					break
				}
			}
			return one(Value{Shape: ObjectShape{Fields: fields}, Unknown: true})
		case ArrayShape:
			return builtinMap(input, args, env)
		default:
			return one(Value{Shape: AnyShape{}, Unknown: true})
		}
	}

	return func(yield func(Value) bool) {
		switch data := input.Data.(type) {
		case map[string]Value:
			result := make(map[string]Value, len(data))
			fields := make(map[string]Shape, len(data))
			for k, v := range data {
				for out := range Eval(args[0], v, env) {
					result[k] = out
					fields[k] = out.Shape
					break
				}
			}
			yield(Value{Shape: ObjectShape{Fields: fields}, Data: result})
		case []Value:
			var results []Value
			var elemShape Shape = AnyShape{}
			first := true
			for _, elem := range data {
				for v := range Eval(args[0], elem, env) {
					results = append(results, v)
					if first {
						elemShape = v.Shape
						first = false
					} else {
						elemShape = Merge(elemShape, v.Shape)
					}
					break
				}
			}
			if results == nil {
				results = []Value{}
			}
			yield(Value{Shape: ArrayShape{Element: elemShape}, Data: results})
		}
	}
}

func builtinSort(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: input.Shape, Unknown: true})
	}
	arr, ok := input.Data.([]Value)
	if !ok {
		return none()
	}
	sorted := make([]Value, len(arr))
	copy(sorted, arr)
	sort.SliceStable(sorted, func(i, j int) bool {
		return valueCompare(sorted[i], sorted[j]) < 0
	})
	return one(Value{Shape: input.Shape, Data: sorted})
}

func builtinSortBy(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: input.Shape, Unknown: true})
	}
	if len(args) == 0 {
		return builtinSort(input, nil, env)
	}
	arr, ok := input.Data.([]Value)
	if !ok {
		return none()
	}
	// Compute sort keys.
	type keyed struct {
		val Value
		key Value
	}
	items := make([]keyed, len(arr))
	for i, v := range arr {
		items[i].val = v
		for k := range Eval(args[0], v, env) {
			items[i].key = k
			break
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return valueCompare(items[i].key, items[j].key) < 0
	})
	sorted := make([]Value, len(items))
	for i, item := range items {
		sorted[i] = item.val
	}
	return one(Value{Shape: input.Shape, Data: sorted})
}

func builtinReverse(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: input.Shape, Unknown: true})
	}
	arr, ok := input.Data.([]Value)
	if !ok {
		if s, ok := input.Data.(string); ok {
			runes := []rune(s)
			for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
				runes[i], runes[j] = runes[j], runes[i]
			}
			return one(Value{Shape: StringShape{}, Data: string(runes)})
		}
		return none()
	}
	reversed := make([]Value, len(arr))
	for i, v := range arr {
		reversed[len(arr)-1-i] = v
	}
	return one(Value{Shape: input.Shape, Data: reversed})
}

func builtinGroupBy(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown {
		var elemShape Shape = AnyShape{}
		if a, ok := input.Shape.(ArrayShape); ok {
			elemShape = a.Element
		}
		return one(Value{Shape: ArrayShape{Element: ArrayShape{Element: elemShape}}, Unknown: true})
	}
	if len(args) == 0 {
		return none()
	}
	arr, ok := input.Data.([]Value)
	if !ok {
		return none()
	}

	type keyed struct {
		val Value
		key Value
	}
	items := make([]keyed, len(arr))
	for i, v := range arr {
		items[i].val = v
		for k := range Eval(args[0], v, env) {
			items[i].key = k
			break
		}
	}
	// Sort by key.
	sort.SliceStable(items, func(i, j int) bool {
		return valueCompare(items[i].key, items[j].key) < 0
	})
	// Group consecutive equal keys.
	var groups []Value
	var elemShape Shape = AnyShape{}
	if a, ok := input.Shape.(ArrayShape); ok {
		elemShape = a.Element
	}
	for i := 0; i < len(items); {
		j := i + 1
		for j < len(items) && valueEqual(items[i].key, items[j].key) {
			j++
		}
		group := make([]Value, j-i)
		for k := i; k < j; k++ {
			group[k-i] = items[k].val
		}
		groups = append(groups, Value{Shape: ArrayShape{Element: elemShape}, Data: group})
		i = j
	}

	return one(Value{Shape: ArrayShape{Element: ArrayShape{Element: elemShape}}, Data: groups})
}

func builtinUnique(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: input.Shape, Unknown: true})
	}
	arr, ok := input.Data.([]Value)
	if !ok {
		return none()
	}
	sorted := make([]Value, len(arr))
	copy(sorted, arr)
	sort.SliceStable(sorted, func(i, j int) bool {
		return valueCompare(sorted[i], sorted[j]) < 0
	})
	var result []Value
	for i, v := range sorted {
		if i == 0 || !valueEqual(v, sorted[i-1]) {
			result = append(result, v)
		}
	}
	return one(Value{Shape: input.Shape, Data: result})
}

func builtinUniqueBy(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: input.Shape, Unknown: true})
	}
	if len(args) == 0 {
		return builtinUnique(input, nil, env)
	}
	arr, ok := input.Data.([]Value)
	if !ok {
		return none()
	}
	type keyed struct {
		val Value
		key Value
	}
	items := make([]keyed, len(arr))
	for i, v := range arr {
		items[i].val = v
		for k := range Eval(args[0], v, env) {
			items[i].key = k
			break
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return valueCompare(items[i].key, items[j].key) < 0
	})
	var result []Value
	for i, item := range items {
		if i == 0 || !valueEqual(item.key, items[i-1].key) {
			result = append(result, item.val)
		}
	}
	return one(Value{Shape: input.Shape, Data: result})
}

func builtinFlatten(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: ArrayShape{Element: AnyShape{}}, Unknown: true})
	}
	arr, ok := input.Data.([]Value)
	if !ok {
		return none()
	}
	var result []Value
	flattenInto(&result, arr, -1)
	return one(Value{Shape: ArrayShape{Element: AnyShape{}}, Data: result})
}

func flattenInto(result *[]Value, arr []Value, depth int) {
	for _, v := range arr {
		if inner, ok := v.Data.([]Value); ok && depth != 0 {
			flattenInto(result, inner, depth-1)
		} else {
			*result = append(*result, v)
		}
	}
}

func builtinFirst(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if len(args) > 0 {
		// first(expr): yield first result of expr.
		return func(yield func(Value) bool) {
			for v := range Eval(args[0], input, env) {
				yield(v)
				return
			}
		}
	}

	if input.Unknown {
		switch s := input.Shape.(type) {
		case ArrayShape:
			return one(Value{Shape: s.Element, Unknown: true})
		default:
			return one(Value{Shape: AnyShape{}, Unknown: true})
		}
	}

	arr, ok := input.Data.([]Value)
	if !ok || len(arr) == 0 {
		return none()
	}
	return one(arr[0])
}

func builtinLast(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if len(args) > 0 {
		// last(expr): yield last result of expr.
		return func(yield func(Value) bool) {
			var last Value
			found := false
			for v := range Eval(args[0], input, env) {
				last = v
				found = true
			}
			if found {
				yield(last)
			}
		}
	}

	if input.Unknown {
		switch s := input.Shape.(type) {
		case ArrayShape:
			return one(Value{Shape: s.Element, Unknown: true})
		default:
			return one(Value{Shape: AnyShape{}, Unknown: true})
		}
	}

	arr, ok := input.Data.([]Value)
	if !ok || len(arr) == 0 {
		return none()
	}
	return one(arr[len(arr)-1])
}

func builtinRange(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: NumberShape{}, Unknown: true})
	}
	return func(yield func(Value) bool) {
		if len(args) == 0 {
			return
		}
		var from, to float64
		if len(args) == 1 {
			// range(n): 0..n-1
			for v := range Eval(args[0], input, env) {
				if f, ok := v.Data.(float64); ok {
					to = f
				}
				break
			}
		} else {
			// range(m; n): m..n-1
			for v := range Eval(args[0], input, env) {
				if f, ok := v.Data.(float64); ok {
					from = f
				}
				break
			}
			for v := range Eval(args[1], input, env) {
				if f, ok := v.Data.(float64); ok {
					to = f
				}
				break
			}
		}
		for i := from; i < to; i++ {
			if !yield(Value{Shape: NumberShape{}, Data: i}) {
				return
			}
		}
	}
}

func builtinLimit(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if len(args) < 2 {
		return none()
	}
	return func(yield func(Value) bool) {
		var n float64
		for v := range Eval(args[0], input, env) {
			if f, ok := v.Data.(float64); ok {
				n = f
			}
			break
		}
		count := 0
		for v := range Eval(args[1], input, env) {
			if count >= int(n) {
				return
			}
			if !yield(v) {
				return
			}
			count++
		}
	}
}

func builtinAdd(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		switch s := input.Shape.(type) {
		case ArrayShape:
			return one(Value{Shape: s.Element, Unknown: true})
		default:
			return one(Value{Shape: AnyShape{}, Unknown: true})
		}
	}

	arr, ok := input.Data.([]Value)
	if !ok || len(arr) == 0 {
		return one(Value{Shape: NullShape{}, Data: nil})
	}

	result := arr[0]
	for _, v := range arr[1:] {
		result = concreteAdd(result, v)
	}
	return one(result)
}

func builtinAny(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: BoolShape{}, Unknown: true})
	}

	arr, ok := input.Data.([]Value)
	if !ok {
		return one(Value{Shape: BoolShape{}, Data: false})
	}

	if len(args) > 0 {
		// any(f): check if any element satisfies f.
		for _, elem := range arr {
			for v := range Eval(args[0], elem, env) {
				if v.Truthy() {
					return one(Value{Shape: BoolShape{}, Data: true})
				}
				break
			}
		}
		return one(Value{Shape: BoolShape{}, Data: false})
	}

	// any: check if any element is truthy.
	for _, elem := range arr {
		if elem.Truthy() {
			return one(Value{Shape: BoolShape{}, Data: true})
		}
	}
	return one(Value{Shape: BoolShape{}, Data: false})
}

func builtinAll(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: BoolShape{}, Unknown: true})
	}

	arr, ok := input.Data.([]Value)
	if !ok {
		return one(Value{Shape: BoolShape{}, Data: true})
	}

	if len(args) > 0 {
		for _, elem := range arr {
			for v := range Eval(args[0], elem, env) {
				if !v.Truthy() {
					return one(Value{Shape: BoolShape{}, Data: false})
				}
				break
			}
		}
		return one(Value{Shape: BoolShape{}, Data: true})
	}

	for _, elem := range arr {
		if !elem.Truthy() {
			return one(Value{Shape: BoolShape{}, Data: false})
		}
	}
	return one(Value{Shape: BoolShape{}, Data: true})
}

func builtinMin(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		switch s := input.Shape.(type) {
		case ArrayShape:
			return one(Value{Shape: s.Element, Unknown: true})
		default:
			return one(Value{Shape: AnyShape{}, Unknown: true})
		}
	}
	arr, ok := input.Data.([]Value)
	if !ok || len(arr) == 0 {
		return one(Value{Shape: NullShape{}, Data: nil})
	}
	m := arr[0]
	for _, v := range arr[1:] {
		if valueCompare(v, m) < 0 {
			m = v
		}
	}
	return one(m)
}

func builtinMax(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		switch s := input.Shape.(type) {
		case ArrayShape:
			return one(Value{Shape: s.Element, Unknown: true})
		default:
			return one(Value{Shape: AnyShape{}, Unknown: true})
		}
	}
	arr, ok := input.Data.([]Value)
	if !ok || len(arr) == 0 {
		return one(Value{Shape: NullShape{}, Data: nil})
	}
	m := arr[0]
	for _, v := range arr[1:] {
		if valueCompare(v, m) > 0 {
			m = v
		}
	}
	return one(m)
}

func builtinMinBy(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown || len(args) == 0 {
		return builtinMin(input, nil, env)
	}
	arr, ok := input.Data.([]Value)
	if !ok || len(arr) == 0 {
		return one(Value{Shape: NullShape{}, Data: nil})
	}
	minVal := arr[0]
	var minKey Value
	for k := range Eval(args[0], arr[0], env) {
		minKey = k
		break
	}
	for _, v := range arr[1:] {
		for k := range Eval(args[0], v, env) {
			if valueCompare(k, minKey) < 0 {
				minVal = v
				minKey = k
			}
			break
		}
	}
	return one(minVal)
}

func builtinMaxBy(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown || len(args) == 0 {
		return builtinMax(input, nil, env)
	}
	arr, ok := input.Data.([]Value)
	if !ok || len(arr) == 0 {
		return one(Value{Shape: NullShape{}, Data: nil})
	}
	maxVal := arr[0]
	var maxKey Value
	for k := range Eval(args[0], arr[0], env) {
		maxKey = k
		break
	}
	for _, v := range arr[1:] {
		for k := range Eval(args[0], v, env) {
			if valueCompare(k, maxKey) > 0 {
				maxVal = v
				maxKey = k
			}
			break
		}
	}
	return one(maxVal)
}

func builtinIndices(_ Value, _ []Node, _ *Environment) iter.Seq[Value] {
	// Simplified: return empty array.
	return one(Value{Shape: ArrayShape{Element: NumberShape{}}, Data: []Value{}})
}

func builtinIndex(_ Value, _ []Node, _ *Environment) iter.Seq[Value] {
	return one(Value{Shape: NullShape{}, Data: nil})
}

func builtinRindex(_ Value, _ []Node, _ *Environment) iter.Seq[Value] {
	return one(Value{Shape: NullShape{}, Data: nil})
}

func builtinInside(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: BoolShape{}, Unknown: true})
	}
	return func(yield func(Value) bool) {
		if len(args) == 0 {
			return
		}
		for other := range Eval(args[0], input, env) {
			yield(Value{Shape: BoolShape{}, Data: valueContains(other, input)})
			return
		}
	}
}

func builtinNth(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if len(args) < 2 {
		return none()
	}
	return func(yield func(Value) bool) {
		var n float64
		for v := range Eval(args[0], input, env) {
			if f, ok := v.Data.(float64); ok {
				n = f
			}
			break
		}
		count := 0
		for v := range Eval(args[1], input, env) {
			if count == int(n) {
				yield(v)
				return
			}
			count++
		}
	}
}

// ---- Conversion ----

func builtinTojson(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: StringShape{}, Unknown: true})
	}
	return one(Value{Shape: StringShape{}, Data: valueToJSON(input)})
}

func builtinFromjson(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: AnyShape{}, Unknown: true})
	}
	s, ok := input.Data.(string)
	if !ok {
		return none()
	}
	var raw any
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		return none()
	}
	return one(ConcreteValue(raw))
}

// ---- Other ----

func builtinNot(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: BoolShape{}, Unknown: true})
	}
	return one(Value{Shape: BoolShape{}, Data: !input.Truthy()})
}

func builtinNull(_ Value, _ []Node, _ *Environment) iter.Seq[Value] {
	return one(Value{Shape: NullShape{}, Data: nil})
}

func builtinTrue(_ Value, _ []Node, _ *Environment) iter.Seq[Value] {
	return one(Value{Shape: BoolShape{}, Data: true})
}

func builtinFalse(_ Value, _ []Node, _ *Environment) iter.Seq[Value] {
	return one(Value{Shape: BoolShape{}, Data: false})
}

func builtinDebug(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	// Pass through; in concrete mode would log to stderr.
	return one(input)
}

func builtinError(_ Value, _ []Node, _ *Environment) iter.Seq[Value] {
	return none()
}

func builtinPath(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: ArrayShape{Element: AnyShape{}}, Unknown: true})
	}
	// Simplified: evaluate args but return path array.
	_ = args
	_ = env
	return one(Value{Shape: ArrayShape{Element: AnyShape{}}, Data: []Value{}})
}

func builtinGetpath(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: AnyShape{}, Unknown: true})
	}
	return func(yield func(Value) bool) {
		if len(args) == 0 {
			return
		}
		for pv := range Eval(args[0], input, env) {
			pathArr, ok := pv.Data.([]Value)
			if !ok {
				yield(Value{Shape: NullShape{}, Data: nil})
				return
			}
			current := input
			for _, seg := range pathArr {
				switch s := seg.Data.(type) {
				case string:
					current = fieldAccess(current, s)
				case float64:
					if arr, ok := current.Data.([]Value); ok {
						idx := int(s)
						if idx >= 0 && idx < len(arr) {
							current = arr[idx]
						} else {
							current = Value{Shape: NullShape{}, Data: nil}
						}
					} else {
						current = Value{Shape: NullShape{}, Data: nil}
					}
				default:
					current = Value{Shape: NullShape{}, Data: nil}
				}
			}
			yield(current)
			return
		}
	}
}

func builtinEnv(_ Value, _ []Node, _ *Environment) iter.Seq[Value] {
	return one(Value{Shape: ObjectShape{Fields: map[string]Shape{}}, Data: map[string]Value{}})
}

func builtinInput(_ Value, _ []Node, _ *Environment) iter.Seq[Value] {
	return none()
}

func builtinRecurse(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: AnyShape{}, Unknown: true})
	}
	if len(args) > 0 {
		// recurse(f): apply f recursively.
		return func(yield func(Value) bool) {
			recurseWithFunc(input, args[0], env, yield)
		}
	}
	return evalRecurse(input)
}

func recurseWithFunc(v Value, f Node, env *Environment, yield func(Value) bool) bool {
	if !yield(v) {
		return false
	}
	for next := range Eval(f, v, env) {
		if next.Data == nil {
			continue
		}
		if !recurseWithFunc(next, f, env, yield) {
			return false
		}
	}
	return true
}

// Type-filter builtins.

func builtinObjects(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(input)
	}
	if _, ok := input.Data.(map[string]Value); ok {
		return one(input)
	}
	return none()
}

func builtinArrays(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(input)
	}
	if _, ok := input.Data.([]Value); ok {
		return one(input)
	}
	return none()
}

func builtinStrings(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(input)
	}
	if _, ok := input.Data.(string); ok {
		return one(input)
	}
	return none()
}

func builtinNumbers(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(input)
	}
	if _, ok := input.Data.(float64); ok {
		return one(input)
	}
	return none()
}

func builtinBooleans(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(input)
	}
	if _, ok := input.Data.(bool); ok {
		return one(input)
	}
	return none()
}

func builtinNulls(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(input)
	}
	if input.Data == nil {
		return one(input)
	}
	return none()
}

func builtinIterables(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(input)
	}
	switch input.Data.(type) {
	case []Value, map[string]Value:
		return one(input)
	}
	return none()
}

func builtinScalars(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(input)
	}
	switch input.Data.(type) {
	case []Value, map[string]Value:
		return none()
	}
	return one(input)
}

func builtinLeafPaths(_ Value, _ []Node, _ *Environment) iter.Seq[Value] {
	// Simplified stub.
	return none()
}

func builtinInfinite(_ Value, _ []Node, _ *Environment) iter.Seq[Value] {
	return one(Value{Shape: NumberShape{}, Data: math.Inf(1)})
}

func builtinNan(_ Value, _ []Node, _ *Environment) iter.Seq[Value] {
	return one(Value{Shape: NumberShape{}, Data: math.NaN()})
}

func builtinIsinfinite(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: BoolShape{}, Unknown: true})
	}
	if f, ok := input.Data.(float64); ok {
		return one(Value{Shape: BoolShape{}, Data: math.IsInf(f, 0)})
	}
	return one(Value{Shape: BoolShape{}, Data: false})
}

func builtinIsnan(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: BoolShape{}, Unknown: true})
	}
	if f, ok := input.Data.(float64); ok {
		return one(Value{Shape: BoolShape{}, Data: math.IsNaN(f)})
	}
	return one(Value{Shape: BoolShape{}, Data: false})
}

func builtinIsnormal(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: BoolShape{}, Unknown: true})
	}
	if f, ok := input.Data.(float64); ok {
		return one(Value{Shape: BoolShape{}, Data: !math.IsInf(f, 0) && !math.IsNaN(f) && f != 0})
	}
	return one(Value{Shape: BoolShape{}, Data: false})
}

func builtinIn(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: BoolShape{}, Unknown: true})
	}
	return func(yield func(Value) bool) {
		if len(args) == 0 {
			return
		}
		for obj := range Eval(args[0], input, env) {
			switch data := obj.Data.(type) {
			case map[string]Value:
				if k, ok := input.Data.(string); ok {
					_, exists := data[k]
					yield(Value{Shape: BoolShape{}, Data: exists})
				} else {
					yield(Value{Shape: BoolShape{}, Data: false})
				}
			case []Value:
				if idx, ok := input.Data.(float64); ok {
					i := int(idx)
					yield(Value{Shape: BoolShape{}, Data: i >= 0 && i < len(data)})
				} else {
					yield(Value{Shape: BoolShape{}, Data: false})
				}
			default:
				yield(Value{Shape: BoolShape{}, Data: false})
			}
			return
		}
	}
}

func builtinAscii(_ Value, _ []Node, _ *Environment) iter.Seq[Value] {
	return one(Value{Shape: NumberShape{}, Data: float64(127)})
}

func builtinExplode(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: ArrayShape{Element: NumberShape{}}, Unknown: true})
	}
	s, ok := input.Data.(string)
	if !ok {
		return none()
	}
	runes := []rune(s)
	vals := make([]Value, len(runes))
	for i, r := range runes {
		vals[i] = Value{Shape: NumberShape{}, Data: float64(r)}
	}
	return one(Value{Shape: ArrayShape{Element: NumberShape{}}, Data: vals})
}

func builtinImplode(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: StringShape{}, Unknown: true})
	}
	arr, ok := input.Data.([]Value)
	if !ok {
		return none()
	}
	runes := make([]rune, len(arr))
	for i, v := range arr {
		if f, ok := v.Data.(float64); ok {
			runes[i] = rune(f)
		}
	}
	return one(Value{Shape: StringShape{}, Data: string(runes)})
}

func builtinAbs(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: NumberShape{}, Unknown: true})
	}
	if f, ok := input.Data.(float64); ok {
		return one(Value{Shape: NumberShape{}, Data: math.Abs(f)})
	}
	return none()
}

func builtinFloor(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: NumberShape{}, Unknown: true})
	}
	if f, ok := input.Data.(float64); ok {
		return one(Value{Shape: NumberShape{}, Data: math.Floor(f)})
	}
	return none()
}

func builtinCeil(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: NumberShape{}, Unknown: true})
	}
	if f, ok := input.Data.(float64); ok {
		return one(Value{Shape: NumberShape{}, Data: math.Ceil(f)})
	}
	return none()
}

func builtinRound(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: NumberShape{}, Unknown: true})
	}
	if f, ok := input.Data.(float64); ok {
		return one(Value{Shape: NumberShape{}, Data: math.Round(f)})
	}
	return none()
}

func builtinSqrt(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: NumberShape{}, Unknown: true})
	}
	if f, ok := input.Data.(float64); ok {
		return one(Value{Shape: NumberShape{}, Data: math.Sqrt(f)})
	}
	return none()
}

func builtinPow(input Value, args []Node, env *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: NumberShape{}, Unknown: true})
	}
	return func(yield func(Value) bool) {
		if len(args) < 2 {
			return
		}
		var base, exp float64
		for v := range Eval(args[0], input, env) {
			if f, ok := v.Data.(float64); ok {
				base = f
			}
			break
		}
		for v := range Eval(args[1], input, env) {
			if f, ok := v.Data.(float64); ok {
				exp = f
			}
			break
		}
		yield(Value{Shape: NumberShape{}, Data: math.Pow(base, exp)})
	}
}

func builtinLog(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: NumberShape{}, Unknown: true})
	}
	if f, ok := input.Data.(float64); ok {
		return one(Value{Shape: NumberShape{}, Data: math.Log(f)})
	}
	return none()
}

func builtinLog2(input Value, _ []Node, _ *Environment) iter.Seq[Value] {
	if input.Unknown {
		return one(Value{Shape: NumberShape{}, Unknown: true})
	}
	if f, ok := input.Data.(float64); ok {
		return one(Value{Shape: NumberShape{}, Data: math.Log2(f)})
	}
	return none()
}

// valueToJSON converts a Value to its JSON string representation.
func valueToJSON(v Value) string {
	raw := valueToAny(v)
	b, err := json.Marshal(raw)
	if err != nil {
		return "null"
	}
	return string(b)
}

// valueToAny converts a Value back to a plain Go value for JSON marshaling.
func valueToAny(v Value) any {
	if v.Unknown {
		return nil
	}
	switch d := v.Data.(type) {
	case nil:
		return nil
	case bool:
		return d
	case float64:
		return d
	case string:
		return d
	case []Value:
		arr := make([]any, len(d))
		for i, elem := range d {
			arr[i] = valueToAny(elem)
		}
		return arr
	case map[string]Value:
		obj := make(map[string]any, len(d))
		for k, elem := range d {
			obj[k] = valueToAny(elem)
		}
		return obj
	default:
		return nil
	}
}
