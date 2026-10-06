package game

import (
	"fmt"
	"strconv"
	"strings"

	"ango/compiler"
	"ango/engine/value"
)

func typeName(v value.Value) string {
	switch v.Type {
	case value.Nil:
		return "nil"
	case value.Bool:
		return "bool"
	case value.Int:
		return "int"
	case value.Float:
		return "float"
	case value.String:
		return "string"
	}
	return "unknown"
}

// Format converts a value to the text shown in dialogue ([variable]).
func Format(v value.Value) string {
	switch v.Type {
	case value.Bool:
		return strconv.FormatBool(v.Bool)
	case value.Int:
		return strconv.FormatInt(v.Int, 10)
	case value.Float:
		return strconv.FormatFloat(v.Float, 'g', -1, 64)
	case value.String:
		return v.String
	}
	return ""
}

func isNumber(v value.Value) bool { return v.Type == value.Int || v.Type == value.Float }

func toFloat(v value.Value) float64 {
	if v.Type == value.Int {
		return float64(v.Int)
	}
	return v.Float
}

// asBool is strict: conditions and logic operators need real bools.
func asBool(v value.Value) (bool, error) {
	if v.Type != value.Bool {
		return false, fmt.Errorf("expected bool, got %s", typeName(v))
	}
	return v.Bool, nil
}

var symbols = map[compiler.Opcode]string{
	compiler.OpAdd: "+", compiler.OpSub: "-", compiler.OpMul: "*", compiler.OpDiv: "/",
	compiler.OpEq: "==", compiler.OpNeq: "!=",
	compiler.OpLt: "<", compiler.OpLe: "<=", compiler.OpGt: ">", compiler.OpGe: ">=",
	compiler.OpAnd: "and", compiler.OpOr: "or",
}

func binary(op compiler.Opcode, a, b value.Value) (value.Value, error) {
	switch op {
	case compiler.OpAdd, compiler.OpSub, compiler.OpMul, compiler.OpDiv:
		return arith(op, a, b)
	case compiler.OpEq:
		return value.OfBool(equal(a, b)), nil
	case compiler.OpNeq:
		return value.OfBool(!equal(a, b)), nil
	case compiler.OpLt, compiler.OpLe, compiler.OpGt, compiler.OpGe:
		return order(op, a, b)
	case compiler.OpAnd, compiler.OpOr:
		x, err := asBool(a)
		if err != nil {
			return value.Value{}, err
		}
		y, err := asBool(b)
		if err != nil {
			return value.Value{}, err
		}
		if op == compiler.OpAnd {
			return value.OfBool(x && y), nil
		}
		return value.OfBool(x || y), nil
	}
	return value.Value{}, fmt.Errorf("internal error: %s is not a binary operator", op)
}

// arith: Int op Int stays Int (division truncates); any Float makes the
// result Float; string + string concatenates. Division by zero is an error.
func arith(op compiler.Opcode, a, b value.Value) (value.Value, error) {
	if op == compiler.OpAdd && a.Type == value.String && b.Type == value.String {
		return value.OfString(a.String + b.String), nil
	}
	if !isNumber(a) || !isNumber(b) {
		return value.Value{}, fmt.Errorf("cannot apply %s to %s and %s", symbols[op], typeName(a), typeName(b))
	}
	if a.Type == value.Int && b.Type == value.Int {
		x, y := a.Int, b.Int
		switch op {
		case compiler.OpAdd:
			return value.OfInt(x + y), nil
		case compiler.OpSub:
			return value.OfInt(x - y), nil
		case compiler.OpMul:
			return value.OfInt(x * y), nil
		case compiler.OpDiv:
			if y == 0 {
				return value.Value{}, fmt.Errorf("division by zero")
			}
			return value.OfInt(x / y), nil
		}
	}
	x, y := toFloat(a), toFloat(b)
	switch op {
	case compiler.OpAdd:
		return value.OfFloat(x + y), nil
	case compiler.OpSub:
		return value.OfFloat(x - y), nil
	case compiler.OpMul:
		return value.OfFloat(x * y), nil
	case compiler.OpDiv:
		if y == 0 {
			return value.Value{}, fmt.Errorf("division by zero")
		}
		return value.OfFloat(x / y), nil
	}
	return value.Value{}, fmt.Errorf("internal error: bad arithmetic op %s", op)
}

// equal never errors: different types are simply not equal
// (Int and Float compare numerically).
func equal(a, b value.Value) bool {
	if isNumber(a) && isNumber(b) {
		if a.Type == value.Int && b.Type == value.Int {
			return a.Int == b.Int
		}
		return toFloat(a) == toFloat(b)
	}
	if a.Type != b.Type {
		return false
	}
	switch a.Type {
	case value.Nil:
		return true
	case value.Bool:
		return a.Bool == b.Bool
	case value.String:
		return a.String == b.String
	}
	return false
}

func order(op compiler.Opcode, a, b value.Value) (value.Value, error) {
	var c int
	switch {
	case a.Type == value.Int && b.Type == value.Int:
		switch {
		case a.Int < b.Int:
			c = -1
		case a.Int > b.Int:
			c = 1
		}
	case isNumber(a) && isNumber(b):
		x, y := toFloat(a), toFloat(b)
		switch {
		case x < y:
			c = -1
		case x > y:
			c = 1
		}
	case a.Type == value.String && b.Type == value.String:
		c = strings.Compare(a.String, b.String)
	default:
		return value.Value{}, fmt.Errorf("cannot compare %s with %s using %s", typeName(a), typeName(b), symbols[op])
	}
	switch op {
	case compiler.OpLt:
		return value.OfBool(c < 0), nil
	case compiler.OpLe:
		return value.OfBool(c <= 0), nil
	case compiler.OpGt:
		return value.OfBool(c > 0), nil
	}
	return value.OfBool(c >= 0), nil // OpGe
}
