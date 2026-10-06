package compiler

import (
	"fmt"
	"strconv"
	"strings"

	"ango/engine/script"
	"ango/engine/value"
)

// Opcode is a single VM operation. Operand A meaning depends on the opcode.
type Opcode uint8

const (
	OpConst Opcode = iota // push Consts[A]
	OpLoad                // push variable slot A
	OpStore               // pop into variable slot A

	OpAdd
	OpSub
	OpMul
	OpDiv

	OpEq
	OpNeq
	OpLt
	OpLe
	OpGt
	OpGe

	OpAnd
	OpOr
	OpNot
	OpNeg

	OpJump        // pc = A
	OpJumpIfFalse // pop; if falsy, pc = A
	OpCall        // push return address; pc = A
	OpReturn      // pop return address into pc
	OpEnd         // stop the script

	OpSay    // show Says[A]
	OpScene  // set background to Scenes[A]
	OpShow   // show Sprites[A]
	OpHide   // hide Hides[A]
	OpChoice // pop len(Choices[A].Options) bools (last option on top),
	//          then present the enabled options of Choices[A]
)

var opNames = [...]string{
	OpConst: "CONST", OpLoad: "LOAD", OpStore: "STORE",
	OpAdd: "ADD", OpSub: "SUB", OpMul: "MUL", OpDiv: "DIV",
	OpEq: "EQ", OpNeq: "NEQ", OpLt: "LT", OpLe: "LE", OpGt: "GT", OpGe: "GE",
	OpAnd: "AND", OpOr: "OR", OpNot: "NOT", OpNeg: "NEG",
	OpJump: "JUMP", OpJumpIfFalse: "JUMPIFFALSE", OpCall: "CALL",
	OpReturn: "RETURN", OpEnd: "END",
	OpSay: "SAY", OpScene: "SCENE", OpShow: "SHOW", OpHide: "HIDE", OpChoice: "CHOICE",
}

func (o Opcode) String() string {
	if int(o) < len(opNames) && opNames[o] != "" {
		return opNames[o]
	}
	return fmt.Sprintf("OP(%d)", uint8(o))
}

type Instruction struct {
	Op Opcode
	A  int
}

// Var is a variable slot. Default is its initial value from a `default` line.
type Var struct {
	Name    string
	Default value.Value
}

// Segment is one piece of interpolated text: a literal or a variable slot.
type Segment struct {
	IsVar   bool
	Literal string
	Name    string // variable name, for debugging
	Slot    int
}

type Say struct {
	ID      string // empty when no @id
	Speaker string // empty for narration
	Parts   []Segment
}

// Transition is optional presentation metadata attached to a Scene, Show or
// Hide command. The compiler and VM treat it as opaque data: Name is never
// checked against a known list here, so new transitions can be added
// purely in the rendering backend. A nil Transition means a hard cut.
type Transition struct {
	Name     string
	Duration float64
}

// SceneCmd is one `scene "asset" [with name duration]` command.
type SceneCmd struct {
	Asset      string
	Transition *Transition
}

// HideCmd is one `hide target [with name duration]` command.
type HideCmd struct {
	Target     string
	Transition *Transition
}

type Sprite struct {
	Character  string
	Expression string
	Position   string
	Flip       bool
	Transition *Transition
}

type ChoiceOption struct {
	Text   []Segment
	Target int // pc of the option body
}

type Choice struct {
	Options []ChoiceOption
}

// Program is the compiled output that the VM executes.
// Code, and Spans are parallel: Spans[pc] is the source location of Code[pc].
type Program struct {
	Code    []Instruction
	Spans   []script.Span
	Consts  []value.Value
	Vars    []Var
	Says    []Say
	Scenes  []SceneCmd
	Sprites []Sprite
	Hides   []HideCmd
	Choices []Choice
	Labels  map[string]int
	Entry   int // pc of label "start"
}

// Disassemble renders the code for debugging and golden tests.
func (p *Program) Disassemble() string {
	var b strings.Builder
	for pc, in := range p.Code {
		fmt.Fprintf(&b, "%04d %s", pc, in.Op)
		switch in.Op {
		case OpConst:
			fmt.Fprintf(&b, " %s", valueString(p.Consts[in.A]))
		case OpScene:
			cmd := p.Scenes[in.A]
			fmt.Fprintf(&b, " %s%s", strconv.Quote(cmd.Asset), transitionSuffix(cmd.Transition))
		case OpHide:
			cmd := p.Hides[in.A]
			fmt.Fprintf(&b, " %s%s", strconv.Quote(cmd.Target), transitionSuffix(cmd.Transition))
		case OpLoad, OpStore:
			fmt.Fprintf(&b, " %s", p.Vars[in.A].Name)
		case OpJump, OpJumpIfFalse, OpCall, OpSay, OpShow, OpChoice:
			fmt.Fprintf(&b, " %d", in.A)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func transitionSuffix(t *Transition) string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf(" with %s %g", t.Name, t.Duration)
}

func valueString(v value.Value) string {
	switch v.Type {
	case value.Nil:
		return "nil"
	case value.Bool:
		return strconv.FormatBool(v.Bool)
	case value.Int:
		return strconv.FormatInt(v.Int, 10)
	case value.Float:
		return strconv.FormatFloat(v.Float, 'g', -1, 64)
	case value.String:
		return strconv.Quote(v.String)
	}
	return "?"
}
