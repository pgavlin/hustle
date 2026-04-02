package jq

import (
	"iter"
	"math"
	"strconv"
)

// Environment holds built-in function definitions and variable bindings.
type Environment struct {
	builtins map[string]Builtin
	vars     map[string]Value
	parent   *Environment
}

// Builtin is a built-in function implementation.
type Builtin struct {
	// Eval evaluates the builtin with the given input and arguments.
	// Args are unevaluated AST nodes -- the builtin decides when/how to evaluate them.
	Eval func(input Value, args []Node, env *Environment) iter.Seq[Value]
}

// DefaultEnv returns an environment with all standard jq builtins.
func DefaultEnv() *Environment {
	env := &Environment{builtins: make(map[string]Builtin), vars: make(map[string]Value)}
	registerBuiltins(env)
	return env
}

// childEnv creates a child environment that inherits builtins and can see parent vars.
func (env *Environment) childEnv() *Environment {
	return &Environment{
		builtins: env.builtins,
		vars:     make(map[string]Value),
		parent:   env,
	}
}

// setVar sets a variable in this environment.
func (env *Environment) setVar(name string, v Value) {
	env.vars[name] = v
}

// lookupVar looks up a variable, walking up the parent chain.
func (env *Environment) lookupVar(name string) (Value, bool) {
	if v, ok := env.vars[name]; ok {
		return v, true
	}
	if env.parent != nil {
		return env.parent.lookupVar(name)
	}
	return Value{}, false
}

// lookupBuiltin looks up a builtin function by name.
func (env *Environment) lookupBuiltin(name string) (Builtin, bool) {
	b, ok := env.builtins[name]
	return b, ok
}

// one returns an iter.Seq[Value] that yields exactly one value.
func one(v Value) iter.Seq[Value] {
	return func(yield func(Value) bool) {
		yield(v)
	}
}

// none returns an iter.Seq[Value] that yields nothing.
func none() iter.Seq[Value] {
	return func(yield func(Value) bool) {}
}

// fieldAccess accesses a field on a value.
func fieldAccess(v Value, name string) Value {
	shape := LookupField(v.Shape, name)
	if v.Unknown {
		return Value{Shape: shape, Unknown: true}
	}
	if data, ok := v.Data.(map[string]Value); ok {
		if field, exists := data[name]; exists {
			return field
		}
	}
	return Value{Shape: NullShape{}, Data: nil}
}

// Eval evaluates a node with the given input, returning a sequence of output values.
func Eval(node Node, input Value, env *Environment) iter.Seq[Value] {
	if node == nil {
		return one(input)
	}

	switch n := node.(type) {
	case *IdentityNode:
		return one(input)

	case *FieldNode:
		return one(fieldAccess(input, n.Name))

	case *PipeNode:
		return evalPipe(n, input, env)

	case *CommaNode:
		return evalComma(n, input, env)

	case *BinOpNode:
		return evalBinOp(n, input, env)

	case *FuncNode:
		return evalFunc(n, input, env)

	case *SuffixNode:
		return evalSuffix(n, input, env)

	case *IterNode:
		return evalIter(input)

	case *IndexNode:
		return evalIndex(n, input, env)

	case *SliceNode:
		return evalSlice(n, input, env)

	case *ObjectNode:
		return evalObject(n, input, env)

	case *ArrayNode:
		return evalArray(n, input, env)

	case *ParenNode:
		return Eval(n.Expr, input, env)

	case *IfNode:
		return evalIf(n, input, env)

	case *TryNode:
		return evalTry(n, input, env)

	case *ReduceNode:
		return evalReduce(n, input, env)

	case *ForeachNode:
		return evalForeach(n, input, env)

	case *UnaryNode:
		return evalUnary(n, input, env)

	case *StringNode:
		return one(Value{Shape: StringShape{}, Data: n.Value})

	case *NumberNode:
		f, err := strconv.ParseFloat(n.Value, 64)
		if err != nil {
			return none()
		}
		return one(Value{Shape: NumberShape{}, Data: f})

	case *BoolNode:
		return one(Value{Shape: BoolShape{}, Data: n.Value})

	case *NullNode:
		return one(Value{Shape: NullShape{}, Data: nil})

	case *OptionalNode:
		return evalOptional(n, input, env)

	case *AsNode:
		return evalAs(n, input, env)

	case *FuncDefNode:
		return evalFuncDef(n, input, env)

	case *IncompleteNode:
		return one(Value{Shape: AnyShape{}, Unknown: true})

	case *RecurseNode:
		return evalRecurse(input)

	case *LabelNode:
		return Eval(n.Body, input, env)

	case *BreakNode:
		return none()

	default:
		return none()
	}
}

func evalPipe(n *PipeNode, input Value, env *Environment) iter.Seq[Value] {
	return func(yield func(Value) bool) {
		for v := range Eval(n.Left, input, env) {
			for out := range Eval(n.Right, v, env) {
				if !yield(out) {
					return
				}
			}
		}
	}
}

func evalComma(n *CommaNode, input Value, env *Environment) iter.Seq[Value] {
	return func(yield func(Value) bool) {
		for v := range Eval(n.Left, input, env) {
			if !yield(v) {
				return
			}
		}
		for v := range Eval(n.Right, input, env) {
			if !yield(v) {
				return
			}
		}
	}
}

func evalBinOp(n *BinOpNode, input Value, env *Environment) iter.Seq[Value] {
	return func(yield func(Value) bool) {
		for lv := range Eval(n.Left, input, env) {
			for rv := range Eval(n.Right, input, env) {
				result := applyBinOp(n.Op, lv, rv)
				if !yield(result) {
					return
				}
			}
		}
	}
}

func applyBinOp(op string, left, right Value) Value {
	// Symbolic mode: if either is unknown, return shape-based result.
	if left.Unknown || right.Unknown {
		return symbolicBinOp(op, left, right)
	}
	return concreteBinOp(op, left, right)
}

func symbolicBinOp(op string, left, right Value) Value {
	switch op {
	case "==", "!=", "<", ">", "<=", ">=":
		return Value{Shape: BoolShape{}, Unknown: true}
	case "and", "or":
		return Value{Shape: BoolShape{}, Unknown: true}
	case "//":
		return Value{Shape: union(left.Shape, right.Shape), Unknown: true}
	case "+":
		return symbolicAdd(left, right)
	case "-", "*", "/", "%":
		return Value{Shape: NumberShape{}, Unknown: true}
	default:
		return Value{Shape: AnyShape{}, Unknown: true}
	}
}

func symbolicAdd(left, right Value) Value {
	switch left.Shape.(type) {
	case StringShape:
		if _, ok := right.Shape.(StringShape); ok {
			return Value{Shape: StringShape{}, Unknown: true}
		}
	case ArrayShape:
		if rb, ok := right.Shape.(ArrayShape); ok {
			la := left.Shape.(ArrayShape)
			return Value{Shape: ArrayShape{Element: Merge(la.Element, rb.Element)}, Unknown: true}
		}
	case ObjectShape:
		if rb, ok := right.Shape.(ObjectShape); ok {
			la := left.Shape.(ObjectShape)
			return Value{Shape: Merge(la, rb), Unknown: true}
		}
	}
	return Value{Shape: NumberShape{}, Unknown: true}
}

func concreteBinOp(op string, left, right Value) Value {
	switch op {
	case "==":
		return Value{Shape: BoolShape{}, Data: valueEqual(left, right)}
	case "!=":
		return Value{Shape: BoolShape{}, Data: !valueEqual(left, right)}
	case "<":
		return Value{Shape: BoolShape{}, Data: valueCompare(left, right) < 0}
	case ">":
		return Value{Shape: BoolShape{}, Data: valueCompare(left, right) > 0}
	case "<=":
		return Value{Shape: BoolShape{}, Data: valueCompare(left, right) <= 0}
	case ">=":
		return Value{Shape: BoolShape{}, Data: valueCompare(left, right) >= 0}
	case "and":
		return Value{Shape: BoolShape{}, Data: left.Truthy() && right.Truthy()}
	case "or":
		return Value{Shape: BoolShape{}, Data: left.Truthy() || right.Truthy()}
	case "//":
		if left.Data != nil && left.Data != false {
			return left
		}
		return right
	case "+":
		return concreteAdd(left, right)
	case "-":
		return concreteSub(left, right)
	case "*":
		return concreteMul(left, right)
	case "/":
		return concreteDiv(left, right)
	case "%":
		return concreteMod(left, right)
	default:
		return Value{Shape: AnyShape{}, Data: nil}
	}
}

func concreteAdd(left, right Value) Value {
	switch lv := left.Data.(type) {
	case float64:
		if rv, ok := right.Data.(float64); ok {
			return Value{Shape: NumberShape{}, Data: lv + rv}
		}
	case string:
		if rv, ok := right.Data.(string); ok {
			return Value{Shape: StringShape{}, Data: lv + rv}
		}
	case []Value:
		if rv, ok := right.Data.([]Value); ok {
			result := make([]Value, 0, len(lv)+len(rv))
			result = append(result, lv...)
			result = append(result, rv...)
			// Merge element shapes.
			var elemShape Shape = AnyShape{}
			if la, ok := left.Shape.(ArrayShape); ok {
				elemShape = la.Element
			}
			if ra, ok := right.Shape.(ArrayShape); ok {
				elemShape = Merge(elemShape, ra.Element)
			}
			return Value{Shape: ArrayShape{Element: elemShape}, Data: result}
		}
	case map[string]Value:
		if rv, ok := right.Data.(map[string]Value); ok {
			result := make(map[string]Value, len(lv)+len(rv))
			for k, v := range lv {
				result[k] = v
			}
			for k, v := range rv {
				result[k] = v
			}
			return Value{Shape: Merge(left.Shape, right.Shape), Data: result}
		}
	case nil:
		return right
	}
	// If right is nil, return left.
	if right.Data == nil {
		return left
	}
	return Value{Shape: AnyShape{}, Data: nil}
}

func concreteSub(left, right Value) Value {
	if lv, ok := left.Data.(float64); ok {
		if rv, ok := right.Data.(float64); ok {
			return Value{Shape: NumberShape{}, Data: lv - rv}
		}
	}
	return Value{Shape: NullShape{}, Data: nil}
}

func concreteMul(left, right Value) Value {
	if lv, ok := left.Data.(float64); ok {
		if rv, ok := right.Data.(float64); ok {
			return Value{Shape: NumberShape{}, Data: lv * rv}
		}
	}
	return Value{Shape: NullShape{}, Data: nil}
}

func concreteDiv(left, right Value) Value {
	if lv, ok := left.Data.(float64); ok {
		if rv, ok := right.Data.(float64); ok {
			if rv == 0 {
				return Value{Shape: NullShape{}, Data: nil}
			}
			return Value{Shape: NumberShape{}, Data: lv / rv}
		}
	}
	return Value{Shape: NullShape{}, Data: nil}
}

func concreteMod(left, right Value) Value {
	if lv, ok := left.Data.(float64); ok {
		if rv, ok := right.Data.(float64); ok {
			if rv == 0 {
				return Value{Shape: NullShape{}, Data: nil}
			}
			return Value{Shape: NumberShape{}, Data: math.Mod(lv, rv)}
		}
	}
	return Value{Shape: NullShape{}, Data: nil}
}

// valueEqual checks if two concrete values are equal.
func valueEqual(a, b Value) bool {
	if a.Data == nil && b.Data == nil {
		return true
	}
	if a.Data == nil || b.Data == nil {
		return false
	}
	switch av := a.Data.(type) {
	case bool:
		bv, ok := b.Data.(bool)
		return ok && av == bv
	case float64:
		bv, ok := b.Data.(float64)
		return ok && av == bv
	case string:
		bv, ok := b.Data.(string)
		return ok && av == bv
	case []Value:
		bv, ok := b.Data.([]Value)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !valueEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	case map[string]Value:
		bv, ok := b.Data.(map[string]Value)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			bvv, ok := bv[k]
			if !ok || !valueEqual(v, bvv) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// valueCompare compares two values using jq ordering:
// null < false < true < number < string < array < object
func valueCompare(a, b Value) int {
	ta := typeOrder(a)
	tb := typeOrder(b)
	if ta != tb {
		if ta < tb {
			return -1
		}
		return 1
	}
	switch av := a.Data.(type) {
	case nil:
		return 0
	case bool:
		bv := b.Data.(bool)
		if av == bv {
			return 0
		}
		if !av {
			return -1
		}
		return 1
	case float64:
		bv := b.Data.(float64)
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
		return 0
	case string:
		bv := b.Data.(string)
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
		return 0
	default:
		return 0
	}
}

func typeOrder(v Value) int {
	switch v.Data.(type) {
	case nil:
		return 0
	case bool:
		if !v.Data.(bool) {
			return 1
		}
		return 2
	case float64:
		return 3
	case string:
		return 4
	case []Value:
		return 5
	case map[string]Value:
		return 6
	default:
		return 7
	}
}

func evalFunc(n *FuncNode, input Value, env *Environment) iter.Seq[Value] {
	// Check for variable reference ($foo).
	if len(n.Name) > 0 && n.Name[0] == '$' {
		if v, ok := env.lookupVar(n.Name); ok {
			return one(v)
		}
		return one(Value{Shape: AnyShape{}, Data: nil})
	}

	// Look up builtin.
	if b, ok := env.lookupBuiltin(n.Name); ok {
		return b.Eval(input, n.Args, env)
	}

	// Unknown function: yield nothing.
	return none()
}

func evalSuffix(n *SuffixNode, input Value, env *Environment) iter.Seq[Value] {
	return func(yield func(Value) bool) {
		for v := range Eval(n.Left, input, env) {
			for out := range Eval(n.Suffix, v, env) {
				if !yield(out) {
					return
				}
			}
		}
	}
}

func evalIter(input Value) iter.Seq[Value] {
	return func(yield func(Value) bool) {
		if input.Unknown {
			// Symbolic: yield element shape once.
			switch s := input.Shape.(type) {
			case ArrayShape:
				yield(Value{Shape: s.Element, Unknown: true})
			case ObjectShape:
				// Yield union of all field value shapes.
				var shapes []Shape
				for _, fs := range s.Fields {
					shapes = append(shapes, fs)
				}
				if len(shapes) == 0 {
					yield(Value{Shape: AnyShape{}, Unknown: true})
				} else if len(shapes) == 1 {
					yield(Value{Shape: shapes[0], Unknown: true})
				} else {
					yield(Value{Shape: union(shapes...), Unknown: true})
				}
			default:
				yield(Value{Shape: AnyShape{}, Unknown: true})
			}
			return
		}

		switch data := input.Data.(type) {
		case []Value:
			for _, v := range data {
				if !yield(v) {
					return
				}
			}
		case map[string]Value:
			for _, v := range data {
				if !yield(v) {
					return
				}
			}
		}
	}
}

func evalIndex(n *IndexNode, input Value, env *Environment) iter.Seq[Value] {
	return func(yield func(Value) bool) {
		for idx := range Eval(n.Expr, input, env) {
			if input.Unknown {
				switch s := input.Shape.(type) {
				case ArrayShape:
					yield(Value{Shape: s.Element, Unknown: true})
				case ObjectShape:
					if idxStr, ok := idx.Data.(string); ok {
						yield(Value{Shape: LookupField(s, idxStr), Unknown: true})
					} else {
						yield(Value{Shape: AnyShape{}, Unknown: true})
					}
				default:
					yield(Value{Shape: AnyShape{}, Unknown: true})
				}
				return
			}

			switch data := input.Data.(type) {
			case []Value:
				if idxNum, ok := idx.Data.(float64); ok {
					i := int(idxNum)
					if i < 0 {
						i = len(data) + i
					}
					if i >= 0 && i < len(data) {
						if !yield(data[i]) {
							return
						}
					} else {
						if !yield(Value{Shape: NullShape{}, Data: nil}) {
							return
						}
					}
				}
			case map[string]Value:
				if idxStr, ok := idx.Data.(string); ok {
					if v, exists := data[idxStr]; exists {
						if !yield(v) {
							return
						}
					} else {
						if !yield(Value{Shape: NullShape{}, Data: nil}) {
							return
						}
					}
				}
			default:
				if !yield(Value{Shape: NullShape{}, Data: nil}) {
					return
				}
			}
		}
	}
}

func evalSlice(n *SliceNode, input Value, env *Environment) iter.Seq[Value] {
	return func(yield func(Value) bool) {
		if input.Unknown {
			yield(Value{Shape: input.Shape, Unknown: true})
			return
		}

		arr, ok := input.Data.([]Value)
		if !ok {
			yield(Value{Shape: NullShape{}, Data: nil})
			return
		}

		low := 0
		high := len(arr)

		if n.Low != nil {
			for lv := range Eval(n.Low, input, env) {
				if f, ok := lv.Data.(float64); ok {
					low = int(f)
					if low < 0 {
						low = len(arr) + low
					}
				}
				break
			}
		}
		if n.High != nil {
			for hv := range Eval(n.High, input, env) {
				if f, ok := hv.Data.(float64); ok {
					high = int(f)
					if high < 0 {
						high = len(arr) + high
					}
				}
				break
			}
		}

		if low < 0 {
			low = 0
		}
		if high > len(arr) {
			high = len(arr)
		}
		if low > high {
			low = high
		}

		result := arr[low:high]
		sliced := make([]Value, len(result))
		copy(sliced, result)
		yield(Value{Shape: input.Shape, Data: sliced})
	}
}

func evalObject(n *ObjectNode, input Value, env *Environment) iter.Seq[Value] {
	return func(yield func(Value) bool) {
		if input.Unknown {
			// Symbolic: build ObjectShape from literal keys.
			fields := make(map[string]Shape)
			for _, pair := range n.Pairs {
				key := objectKeyName(pair.Key)
				if key != "" {
					var valShape Shape = AnyShape{}
					if pair.Value != nil {
						for v := range Eval(pair.Value, input, env) {
							valShape = v.Shape
							break
						}
					} else {
						valShape = LookupField(input.Shape, key)
					}
					fields[key] = valShape
				}
			}
			yield(Value{Shape: ObjectShape{Fields: fields}, Unknown: true})
			return
		}

		// Concrete: build object by evaluating each key-value pair.
		// Object construction can produce multiple outputs if keys/values are generators.
		// For simplicity, handle the common case of single-valued keys and values.
		buildObject(n.Pairs, 0, input, env, make(map[string]Value), make(map[string]Shape), yield)
	}
}

// buildObject recursively builds object values, handling generators in keys/values.
func buildObject(pairs []KeyValue, idx int, input Value, env *Environment, data map[string]Value, fields map[string]Shape, yield func(Value) bool) {
	if idx >= len(pairs) {
		// Clone the maps for this output.
		d := make(map[string]Value, len(data))
		f := make(map[string]Shape, len(fields))
		for k, v := range data {
			d[k] = v
		}
		for k, v := range fields {
			f[k] = v
		}
		yield(Value{Shape: ObjectShape{Fields: f}, Data: d})
		return
	}

	pair := pairs[idx]
	key := objectKeyName(pair.Key)

	if key != "" {
		// Literal key.
		valExpr := pair.Value
		if valExpr == nil {
			// Shorthand: {foo} means {foo: .foo}
			valExpr = &FieldNode{Name: key}
		}
		for v := range Eval(valExpr, input, env) {
			data[key] = v
			fields[key] = v.Shape
			buildObject(pairs, idx+1, input, env, data, fields, yield)
			delete(data, key)
			delete(fields, key)
		}
	} else {
		// Computed key: evaluate key expression.
		for kv := range Eval(pair.Key, input, env) {
			keyStr, ok := kv.Data.(string)
			if !ok {
				continue
			}
			valExpr := pair.Value
			if valExpr == nil {
				valExpr = &FieldNode{Name: keyStr}
			}
			for v := range Eval(valExpr, input, env) {
				data[keyStr] = v
				fields[keyStr] = v.Shape
				buildObject(pairs, idx+1, input, env, data, fields, yield)
				delete(data, keyStr)
				delete(fields, keyStr)
			}
		}
	}
}

// objectKeyName extracts the literal key name from a key node, or "" if it's a computed key.
func objectKeyName(key Node) string {
	switch k := key.(type) {
	case *StringNode:
		return k.Value
	case *FieldNode:
		return k.Name
	default:
		return ""
	}
}

func evalArray(n *ArrayNode, input Value, env *Environment) iter.Seq[Value] {
	return func(yield func(Value) bool) {
		if input.Unknown {
			// Symbolic: evaluate inner expr symbolically and wrap in ArrayShape.
			var elemShape Shape = AnyShape{}
			if n.Expr != nil {
				for v := range Eval(n.Expr, input, env) {
					elemShape = v.Shape
					break
				}
			}
			yield(Value{Shape: ArrayShape{Element: elemShape}, Unknown: true})
			return
		}

		// Concrete: collect all results.
		var vals []Value
		var elemShape Shape = AnyShape{}
		if n.Expr != nil {
			first := true
			for v := range Eval(n.Expr, input, env) {
				vals = append(vals, v)
				if first {
					elemShape = v.Shape
					first = false
				} else {
					elemShape = Merge(elemShape, v.Shape)
				}
			}
		}
		if vals == nil {
			vals = []Value{}
		}
		yield(Value{Shape: ArrayShape{Element: elemShape}, Data: vals})
	}
}

func evalIf(n *IfNode, input Value, env *Environment) iter.Seq[Value] {
	return func(yield func(Value) bool) {
		if input.Unknown {
			// Symbolic: yield union of all branches.
			var shapes []Shape
			for v := range Eval(n.Then, input, env) {
				shapes = append(shapes, v.Shape)
				break
			}
			for _, elif := range n.Elifs {
				for v := range Eval(elif.Body, input, env) {
					shapes = append(shapes, v.Shape)
					break
				}
			}
			if n.Else != nil {
				for v := range Eval(n.Else, input, env) {
					shapes = append(shapes, v.Shape)
					break
				}
			}
			if len(shapes) == 0 {
				yield(Value{Shape: AnyShape{}, Unknown: true})
			} else if len(shapes) == 1 {
				yield(Value{Shape: shapes[0], Unknown: true})
			} else {
				yield(Value{Shape: union(shapes...), Unknown: true})
			}
			return
		}

		// Concrete: evaluate condition and branch.
		for cv := range Eval(n.Cond, input, env) {
			if cv.Truthy() {
				for v := range Eval(n.Then, input, env) {
					if !yield(v) {
						return
					}
				}
				return
			}
			break
		}

		// Check elifs.
		for _, elif := range n.Elifs {
			for cv := range Eval(elif.Cond, input, env) {
				if cv.Truthy() {
					for v := range Eval(elif.Body, input, env) {
						if !yield(v) {
							return
						}
					}
					return
				}
				break
			}
		}

		// Else.
		if n.Else != nil {
			for v := range Eval(n.Else, input, env) {
				if !yield(v) {
					return
				}
			}
		} else {
			yield(input)
		}
	}
}

func evalTry(n *TryNode, input Value, env *Environment) iter.Seq[Value] {
	return func(yield func(Value) bool) {
		// Try to evaluate body; suppress errors by catching panics.
		// In our implementation, errors result in no output, so try just passes through.
		for v := range Eval(n.Body, input, env) {
			if !yield(v) {
				return
			}
		}
	}
}

func evalReduce(n *ReduceNode, input Value, env *Environment) iter.Seq[Value] {
	return func(yield func(Value) bool) {
		if input.Unknown {
			// Symbolic: evaluate init for shape.
			for v := range Eval(n.Init, input, env) {
				yield(Value{Shape: v.Shape, Unknown: true})
				return
			}
			yield(Value{Shape: AnyShape{}, Unknown: true})
			return
		}

		// Concrete: fold.
		var acc Value
		for v := range Eval(n.Init, input, env) {
			acc = v
			break
		}

		for elem := range Eval(n.Expr, input, env) {
			childEnv := env.childEnv()
			childEnv.setVar(n.Pattern, elem)
			for v := range Eval(n.Update, acc, childEnv) {
				acc = v
				break
			}
		}

		yield(acc)
	}
}

func evalForeach(n *ForeachNode, input Value, env *Environment) iter.Seq[Value] {
	return func(yield func(Value) bool) {
		if input.Unknown {
			yield(Value{Shape: AnyShape{}, Unknown: true})
			return
		}

		var acc Value
		for v := range Eval(n.Init, input, env) {
			acc = v
			break
		}

		for elem := range Eval(n.Expr, input, env) {
			childEnv := env.childEnv()
			childEnv.setVar(n.Pattern, elem)
			for v := range Eval(n.Update, acc, childEnv) {
				acc = v
				break
			}

			if n.Extract != nil {
				for v := range Eval(n.Extract, acc, childEnv) {
					if !yield(v) {
						return
					}
				}
			} else {
				if !yield(acc) {
					return
				}
			}
		}
	}
}

func evalUnary(n *UnaryNode, input Value, env *Environment) iter.Seq[Value] {
	return func(yield func(Value) bool) {
		for v := range Eval(n.Expr, input, env) {
			if v.Unknown {
				yield(Value{Shape: NumberShape{}, Unknown: true})
				return
			}
			if f, ok := v.Data.(float64); ok {
				if !yield(Value{Shape: NumberShape{}, Data: -f}) {
					return
				}
			}
		}
	}
}

func evalOptional(n *OptionalNode, input Value, env *Environment) iter.Seq[Value] {
	return func(yield func(Value) bool) {
		for v := range Eval(n.Expr, input, env) {
			if !yield(v) {
				return
			}
		}
	}
}

func evalAs(n *AsNode, input Value, env *Environment) iter.Seq[Value] {
	return func(yield func(Value) bool) {
		for v := range Eval(n.Expr, input, env) {
			childEnv := env.childEnv()
			childEnv.setVar(n.Pattern, v)
			for out := range Eval(n.Body, input, childEnv) {
				if !yield(out) {
					return
				}
			}
		}
	}
}

func evalFuncDef(n *FuncDefNode, input Value, env *Environment) iter.Seq[Value] {
	// Register the function as a builtin in the environment.
	defBody := n.Body
	defParams := n.Params
	env.builtins[n.Name] = Builtin{
		Eval: func(input Value, args []Node, callEnv *Environment) iter.Seq[Value] {
			childEnv := callEnv.childEnv()
			// Bind parameters: each param is a name that, when called as a function,
			// evaluates the corresponding arg node with the caller's input.
			for i, param := range defParams {
				if i < len(args) {
					// Evaluate the argument and bind it.
					for v := range Eval(args[i], input, callEnv) {
						childEnv.setVar(param, v)
						break
					}
				}
			}
			return Eval(defBody, input, childEnv)
		},
	}

	// Evaluate the rest.
	return Eval(n.Next, input, env)
}

func evalRecurse(input Value) iter.Seq[Value] {
	return func(yield func(Value) bool) {
		if input.Unknown {
			yield(Value{Shape: AnyShape{}, Unknown: true})
			return
		}
		recurseValue(input, yield)
	}
}

func recurseValue(v Value, yield func(Value) bool) bool {
	if !yield(v) {
		return false
	}
	switch data := v.Data.(type) {
	case []Value:
		for _, elem := range data {
			if !recurseValue(elem, yield) {
				return false
			}
		}
	case map[string]Value:
		for _, fv := range data {
			if !recurseValue(fv, yield) {
				return false
			}
		}
	}
	return true
}
