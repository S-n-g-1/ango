package value

type Type uint8

const (
	Nil Type = iota
	Bool
	Int
	Float
	String
)

type Value struct {
	Type   Type
	Bool   bool
	Int    int64
	Float  float64
	String string
}

func OfNil() Value {
	return Value{
		Type: Nil,
	}
}

func OfBool(v bool) Value {
	return Value{
		Type: Bool,
		Bool: v,
	}
}

func OfInt(v int64) Value {
	return Value{
		Type: Int,
		Int:  v,
	}
}

func OfFloat(v float64) Value {
	return Value{
		Type:  Float,
		Float: v,
	}
}

func OfString(v string) Value {
	return Value{
		Type:   String,
		String: v,
	}
}
