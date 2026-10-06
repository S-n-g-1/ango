package game

import (
	"errors"
	"fmt"
	"strings"

	"ango/compiler"
	"ango/engine/script"
	"ango/engine/value"
)

const (
	// MaxStepsPerEvent stops scripts that loop forever without producing output.
	MaxStepsPerEvent = 1_000_000
	// MaxCallDepth stops runaway recursion via `call`.
	MaxCallDepth = 1024
)

type EventKind uint8

const (
	EventSay EventKind = iota + 1
	EventScene
	EventShow
	EventHide
	EventChoice
	EventEnd
)

func (k EventKind) String() string {
	switch k {
	case EventSay:
		return "say"
	case EventScene:
		return "scene"
	case EventShow:
		return "show"
	case EventHide:
		return "hide"
	case EventChoice:
		return "choice"
	case EventEnd:
		return "end"
	}
	return "unknown"
}

// Event is what the VM hands to the frontend. Only the fields that
// belong to Kind are set.
type Event struct {
	Kind EventKind

	// EventSay
	ID      string
	Speaker string
	Text    string

	// EventScene
	Asset string

	// EventShow
	Sprite SpriteState

	// EventHide
	Target string

	// EventScene, EventShow, EventHide: nil means a hard cut (no "with" clause).
	Transition *compiler.Transition

	// EventChoice
	Options []string
}

// VM executes a compiled Program. Drive it with Next (and Choose after an
// EventChoice). After a runtime error the VM stays failed; create a new one
// or Restore a saved state.
type VM struct {
	prog  *compiler.Program
	state GameState
	stack []value.Value
	slots map[string]int
	err   error
}

func New(prog *compiler.Program) *VM {
	vm := &VM{prog: prog, slots: make(map[string]int, len(prog.Vars))}
	for i, v := range prog.Vars {
		vm.slots[v.Name] = i
	}
	vm.state = newState(prog)
	return vm
}

// State returns a deep copy of the current state, safe to save.
func (vm *VM) State() GameState { return vm.state.Clone() }

// Restore replaces the state with a saved one after validating it.
func (vm *VM) Restore(s GameState) error {
	p := vm.prog
	if len(s.Vars) != len(p.Vars) {
		return fmt.Errorf("state has %d variables, program has %d", len(s.Vars), len(p.Vars))
	}
	if s.PC < 0 || s.PC >= len(p.Code) {
		return fmt.Errorf("state pc %d out of range", s.PC)
	}
	for _, r := range s.CallStack {
		if r < 0 || r >= len(p.Code) {
			return fmt.Errorf("state return address %d out of range", r)
		}
	}
	if pc := s.Pending; pc != nil {
		if pc.Choice < 0 || pc.Choice >= len(p.Choices) {
			return fmt.Errorf("state pending choice %d out of range", pc.Choice)
		}
		n := len(p.Choices[pc.Choice].Options)
		if len(pc.Enabled) == 0 {
			return errors.New("state pending choice has no enabled options")
		}
		for _, idx := range pc.Enabled {
			if idx < 0 || idx >= n {
				return fmt.Errorf("state pending option %d out of range", idx)
			}
		}
	}
	vm.state = s.Clone()
	vm.stack = vm.stack[:0]
	vm.err = nil
	return nil
}

// Var reads a variable by name (for the UI, debugging and tests).
func (vm *VM) Var(name string) (value.Value, bool) {
	i, ok := vm.slots[name]
	if !ok {
		return value.Value{}, false
	}
	return vm.state.Vars[i], true
}

// Next runs until the next event for the frontend.
//   - After EventSay/Scene/Show/Hide, call Next again to continue.
//   - After EventChoice, call Choose; calling Next before that repeats the event.
//   - After EventEnd, Next keeps returning EventEnd.
func (vm *VM) Next() (Event, error) {
	if vm.err != nil {
		return Event{}, vm.err
	}
	st := &vm.state
	if st.Ended {
		return Event{Kind: EventEnd}, nil
	}
	if st.Pending != nil {
		return vm.choiceEvent(), nil
	}

	for steps := 0; steps < MaxStepsPerEvent; steps++ {
		pc := st.PC
		if pc < 0 || pc >= len(vm.prog.Code) {
			return Event{}, vm.fail(-1, fmt.Errorf("program counter %d out of range", pc))
		}
		st.PC = pc + 1
		ev, emit, err := vm.exec(pc, vm.prog.Code[pc])
		if err != nil {
			return Event{}, vm.fail(pc, err)
		}
		if emit {
			return ev, nil
		}
	}
	return Event{}, vm.fail(st.PC-1,
		fmt.Errorf("script ran %d instructions without producing output (infinite loop?)", MaxStepsPerEvent))
}

// Choose answers a pending EventChoice. index is a position in Event.Options.
func (vm *VM) Choose(index int) error {
	st := &vm.state
	if st.Pending == nil {
		return errors.New("no choice is pending")
	}
	if index < 0 || index >= len(st.Pending.Enabled) {
		return fmt.Errorf("choice index %d out of range (0..%d)", index, len(st.Pending.Enabled)-1)
	}
	opt := vm.prog.Choices[st.Pending.Choice].Options[st.Pending.Enabled[index]]
	st.PC = opt.Target
	st.Pending = nil
	return nil
}

// fail records a sticky error located at the source span of instruction pc.
func (vm *VM) fail(pc int, err error) error {
	var span script.Span
	if pc >= 0 && pc < len(vm.prog.Spans) {
		span = vm.prog.Spans[pc]
	}
	vm.err = &script.Error{Span: span, Msg: err.Error()}
	return vm.err
}

func (vm *VM) push(v value.Value) { vm.stack = append(vm.stack, v) }

func (vm *VM) pop() (value.Value, error) {
	n := len(vm.stack)
	if n == 0 {
		return value.Value{}, errors.New("internal error: operand stack underflow")
	}
	v := vm.stack[n-1]
	vm.stack = vm.stack[:n-1]
	return v, nil
}

func (vm *VM) text(segs []compiler.Segment) string {
	var b strings.Builder
	for _, s := range segs {
		if s.IsVar {
			b.WriteString(Format(vm.state.Vars[s.Slot]))
		} else {
			b.WriteString(s.Literal)
		}
	}
	return b.String()
}

func (vm *VM) choiceEvent() Event {
	p := vm.state.Pending
	ch := vm.prog.Choices[p.Choice]
	opts := make([]string, len(p.Enabled))
	for i, idx := range p.Enabled {
		opts[i] = vm.text(ch.Options[idx].Text)
	}
	return Event{Kind: EventChoice, Options: opts}
}

// exec runs one instruction. The bool reports whether an event was produced.
func (vm *VM) exec(pc int, in compiler.Instruction) (Event, bool, error) {
	st := &vm.state
	p := vm.prog

	switch in.Op {
	case compiler.OpConst:
		vm.push(p.Consts[in.A])

	case compiler.OpLoad:
		vm.push(st.Vars[in.A])

	case compiler.OpStore:
		v, err := vm.pop()
		if err != nil {
			return Event{}, false, err
		}
		st.Vars[in.A] = v

	case compiler.OpAdd, compiler.OpSub, compiler.OpMul, compiler.OpDiv,
		compiler.OpEq, compiler.OpNeq,
		compiler.OpLt, compiler.OpLe, compiler.OpGt, compiler.OpGe,
		compiler.OpAnd, compiler.OpOr:
		b, err := vm.pop()
		if err != nil {
			return Event{}, false, err
		}
		a, err := vm.pop()
		if err != nil {
			return Event{}, false, err
		}
		r, err := binary(in.Op, a, b)
		if err != nil {
			return Event{}, false, err
		}
		vm.push(r)

	case compiler.OpNot:
		v, err := vm.pop()
		if err != nil {
			return Event{}, false, err
		}
		b, err := asBool(v)
		if err != nil {
			return Event{}, false, err
		}
		vm.push(value.OfBool(!b))

	case compiler.OpNeg:
		v, err := vm.pop()
		if err != nil {
			return Event{}, false, err
		}
		switch v.Type {
		case value.Int:
			vm.push(value.OfInt(-v.Int))
		case value.Float:
			vm.push(value.OfFloat(-v.Float))
		default:
			return Event{}, false, fmt.Errorf("cannot negate %s", typeName(v))
		}

	case compiler.OpJump:
		st.PC = in.A

	case compiler.OpJumpIfFalse:
		v, err := vm.pop()
		if err != nil {
			return Event{}, false, err
		}
		b, err := asBool(v)
		if err != nil {
			return Event{}, false, err
		}
		if !b {
			st.PC = in.A
		}

	case compiler.OpCall:
		if len(st.CallStack) >= MaxCallDepth {
			return Event{}, false, fmt.Errorf("call stack overflow (more than %d nested calls)", MaxCallDepth)
		}
		st.CallStack = append(st.CallStack, st.PC)
		st.PC = in.A

	case compiler.OpReturn:
		n := len(st.CallStack)
		if n == 0 {
			return Event{}, false, errors.New("return without call")
		}
		st.PC = st.CallStack[n-1]
		st.CallStack = st.CallStack[:n-1]

	case compiler.OpEnd:
		st.Ended = true
		st.PC = pc // stay on END so a saved state has a valid pc
		return Event{Kind: EventEnd}, true, nil

	case compiler.OpSay:
		s := p.Says[in.A]
		return Event{Kind: EventSay, ID: s.ID, Speaker: s.Speaker, Text: vm.text(s.Parts)}, true, nil

	case compiler.OpScene:
		cmd := p.Scenes[in.A]
		st.Background = cmd.Asset
		return Event{Kind: EventScene, Asset: cmd.Asset, Transition: cmd.Transition}, true, nil

	case compiler.OpShow:
		sp := p.Sprites[in.A]
		s := SpriteState{Character: sp.Character, Expression: sp.Expression, Position: sp.Position, Flip: sp.Flip}
		st.showSprite(s)
		return Event{Kind: EventShow, Sprite: s, Transition: sp.Transition}, true, nil

	case compiler.OpHide:
		cmd := p.Hides[in.A]
		st.hideSprite(cmd.Target)
		return Event{Kind: EventHide, Target: cmd.Target, Transition: cmd.Transition}, true, nil

	case compiler.OpChoice:
		ch := p.Choices[in.A]
		conds := make([]bool, len(ch.Options))
		for i := len(conds) - 1; i >= 0; i-- { // last option is on top of the stack
			v, err := vm.pop()
			if err != nil {
				return Event{}, false, err
			}
			b, err := asBool(v)
			if err != nil {
				return Event{}, false, fmt.Errorf("choice condition: %w", err)
			}
			conds[i] = b
		}
		var enabled []int
		for i, ok := range conds {
			if ok {
				enabled = append(enabled, i)
			}
		}
		if len(enabled) == 0 {
			return Event{}, false, errors.New("choice has no available options")
		}
		st.Pending = &PendingChoice{Choice: in.A, Enabled: enabled}
		return vm.choiceEvent(), true, nil

	default:
		return Event{}, false, fmt.Errorf("internal error: unknown opcode %s", in.Op)
	}
	return Event{}, false, nil
}
