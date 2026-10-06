package script

import "ango/engine/value"

// ============================================================
// Interfaces
// ============================================================

type Node interface {
	Pos() Span
}

type Statement interface {
	Node
	statementNode()
}

type TextPart interface {
	Node
	textPartNode()
}

type Expression interface {
	Node
	expressionNode()
}

// ============================================================
// Program
// ============================================================

type ProgramNode struct {
	Span          Span
	Namespace     string // "" = global
	NamespaceSpan Span
	Declarations  []*DefaultNode
	Labels        []*LabelNode
}

type DefaultNode struct {
	Span  Span
	Name  string
	Value Expression
}

type LabelNode struct {
	Span      Span
	Name      string
	Body      []Statement
	Namespace string
}

// ============================================================
// Statements
// ============================================================

// Transition is an optional "with <name> <duration>" clause on scene, show
// and hide. It is opaque to the compiler and VM: Name is not validated
// against any known list, so new transitions (dissolve, slide_left, ...)
// can be added purely in the rendering backend without touching the
// language. nil means no transition (a hard cut).
type Transition struct {
	Name     string
	Duration float64
}

type SayNode struct {
	Span    Span
	ID      string
	Speaker string
	Parts   []TextPart
}

type ShowBackgroundNode struct {
	Span       Span
	Asset      string
	Transition *Transition
}

type ShowSpriteNode struct {
	Span       Span
	Character  string
	Expression string
	Position   string
	Flip       bool
	Transition *Transition
}

type HideNode struct {
	Span       Span
	Target     string
	Transition *Transition
}

type SetNode struct {
	Span  Span
	Name  string
	Value Expression
}

type JumpNode struct {
	Span  Span
	Label string
}

type CallNode struct {
	Span  Span
	Label string
}

type ReturnNode struct {
	Span Span
}

type EndNode struct {
	Span Span
}

type IfNode struct {
	Span      Span
	Condition Expression
	Then      []Statement
	Else      []Statement
}

type ChoiceNode struct {
	Span    Span
	Options []ChoiceOption
}

// ChoiceOption bukan Node.
// Ia hanya merupakan data pembentuk ChoiceNode.
type ChoiceOption struct {
	Span      Span
	Text      []TextPart
	Condition Expression
	Body      []Statement
}

// ============================================================
// Text
// ============================================================

type TextLiteral struct {
	Span Span
	Text string
}

type TextVariable struct {
	Span Span
	Name string
}

// ============================================================
// Expressions
// ============================================================

type LiteralExpr struct {
	Span  Span
	Value value.Value
}

type VariableExpr struct {
	Span Span
	Name string
}

type BinaryExpr struct {
	Span  Span
	Left  Expression
	Op    BinaryOp
	Right Expression
}

type UnaryExpr struct {
	Span    Span
	Op      UnaryOp
	Operand Expression
}

// ============================================================
// Binary Operators
// ============================================================

type BinaryOp uint8

const (
	BinaryAdd BinaryOp = iota
	BinarySub
	BinaryMul
	BinaryDiv
	BinaryEqual
	BinaryNotEqual
	BinaryLess
	BinaryLessEqual
	BinaryGreater
	BinaryGreaterEqual
	BinaryAnd
	BinaryOr
)

func (op BinaryOp) String() string {
	switch op {
	case BinaryAdd:
		return "+"
	case BinarySub:
		return "-"
	case BinaryMul:
		return "*"
	case BinaryDiv:
		return "/"
	case BinaryEqual:
		return "=="
	case BinaryNotEqual:
		return "!="
	case BinaryLess:
		return "<"
	case BinaryLessEqual:
		return "<="
	case BinaryGreater:
		return ">"
	case BinaryGreaterEqual:
		return ">="
	case BinaryAnd:
		return "and"
	case BinaryOr:
		return "or"
	default:
		return "<unknown-binary-op>"
	}
}

// ============================================================
// Unary Operators
// ============================================================

type UnaryOp uint8

const (
	UnaryNot UnaryOp = iota
	UnaryNeg
)

func (op UnaryOp) String() string {
	switch op {
	case UnaryNot:
		return "not"
	case UnaryNeg:
		return "-"
	default:
		return "<unknown-unary-op>"
	}
}

// ============================================================
// Pos()
// ============================================================

func (n *ProgramNode) Pos() Span        { return n.Span }
func (n *DefaultNode) Pos() Span        { return n.Span }
func (n *LabelNode) Pos() Span          { return n.Span }
func (n *SayNode) Pos() Span            { return n.Span }
func (n *ShowBackgroundNode) Pos() Span { return n.Span }
func (n *ShowSpriteNode) Pos() Span     { return n.Span }
func (n *HideNode) Pos() Span           { return n.Span }
func (n *SetNode) Pos() Span            { return n.Span }
func (n *JumpNode) Pos() Span           { return n.Span }
func (n *CallNode) Pos() Span           { return n.Span }
func (n *ReturnNode) Pos() Span         { return n.Span }
func (n *EndNode) Pos() Span            { return n.Span }
func (n *IfNode) Pos() Span             { return n.Span }
func (n *ChoiceNode) Pos() Span         { return n.Span }
func (n *TextLiteral) Pos() Span        { return n.Span }
func (n *TextVariable) Pos() Span       { return n.Span }
func (n *LiteralExpr) Pos() Span        { return n.Span }
func (n *VariableExpr) Pos() Span       { return n.Span }
func (n *BinaryExpr) Pos() Span         { return n.Span }
func (n *UnaryExpr) Pos() Span          { return n.Span }

// ============================================================
// Marker Methods
// ============================================================

func (*SayNode) statementNode()            {}
func (*ShowBackgroundNode) statementNode() {}
func (*ShowSpriteNode) statementNode()     {}
func (*HideNode) statementNode()           {}
func (*SetNode) statementNode()            {}
func (*JumpNode) statementNode()           {}
func (*CallNode) statementNode()           {}
func (*ReturnNode) statementNode()         {}
func (*EndNode) statementNode()            {}
func (*IfNode) statementNode()             {}
func (*ChoiceNode) statementNode()         {}

func (*TextLiteral) textPartNode()  {}
func (*TextVariable) textPartNode() {}

func (*LiteralExpr) expressionNode()  {}
func (*VariableExpr) expressionNode() {}
func (*BinaryExpr) expressionNode()   {}
func (*UnaryExpr) expressionNode()    {}

// ============================================================
// Compile-time Assertions
// ============================================================

var (
	_ Node = (*ProgramNode)(nil)
	_ Node = (*DefaultNode)(nil)
	_ Node = (*LabelNode)(nil)

	_ Statement = (*SayNode)(nil)
	_ Statement = (*ShowBackgroundNode)(nil)
	_ Statement = (*ShowSpriteNode)(nil)
	_ Statement = (*HideNode)(nil)
	_ Statement = (*SetNode)(nil)
	_ Statement = (*JumpNode)(nil)
	_ Statement = (*CallNode)(nil)
	_ Statement = (*ReturnNode)(nil)
	_ Statement = (*EndNode)(nil)
	_ Statement = (*IfNode)(nil)
	_ Statement = (*ChoiceNode)(nil)

	_ TextPart = (*TextLiteral)(nil)
	_ TextPart = (*TextVariable)(nil)

	_ Expression = (*LiteralExpr)(nil)
	_ Expression = (*VariableExpr)(nil)
	_ Expression = (*BinaryExpr)(nil)
	_ Expression = (*UnaryExpr)(nil)
)
