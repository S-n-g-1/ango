package compiler

import (
	"sort"
	"strings"

	"ango/engine/script"
	"ango/engine/value"
)

// EntryLabel is the label where execution starts.
const EntryLabel = "start"

type fixup struct {
	pc    int
	label string
}

type compiler struct {
	prog *Program
	errs Errors

	vars       map[string]int
	varDecls   map[string]*script.DefaultNode
	labelDecls map[string]*script.LabelNode

	curNS string // namespace label yang sedang divalidasi/di-emit

	consts map[value.Value]int
	fixups []fixup
}

// Compile validates the AST (pass 1) and, if it is clean, emits
// instructions (pass 2). On failure it returns every error found.
func Compile(ast *script.ProgramNode) (*Program, error) {
	c := &compiler{
		prog:       &Program{Labels: map[string]int{}},
		vars:       map[string]int{},
		varDecls:   map[string]*script.DefaultNode{},
		labelDecls: map[string]*script.LabelNode{},
		consts:     map[value.Value]int{},
	}

	// Pass 1: collect declarations and validate.
	c.collect(ast)
	c.validate(ast)

	// Pass 2: emit code, only when pass 1 found nothing.
	if len(c.errs) == 0 {
		c.emitProgram(ast)
	}

	if len(c.errs) > 0 {
		sort.SliceStable(c.errs, func(i, j int) bool {
			a, b := c.errs[i].Span.Start, c.errs[j].Span.Start
			if a.Line != b.Line {
				return a.Line < b.Line
			}
			return a.Column < b.Column
		})
		return nil, c.errs
	}
	return c.prog, nil
}

// errorf records a plain compiler error with no hint.
func (c *compiler) errorf(span script.Span, format string, args ...any) {
	c.errs = append(c.errs, script.NewCompilerError(span, format, args...))
}

// errorHintf records a compiler error with a one-line suggestion.
// Pass hint == "" to behave exactly like errorf (no "hint:" line printed).
func (c *compiler) errorHintf(span script.Span, hint, format string, args ...any) {
	e := script.NewCompilerError(span, format, args...)
	if hint != "" {
		e = e.WithHint(hint)
	}
	c.errs = append(c.errs, e)
}

// ============================================================
// Pass 1: collect + validate
// ============================================================

func (c *compiler) collect(p *script.ProgramNode) {
	for _, d := range p.Declarations {
		if prev, dup := c.varDecls[d.Name]; dup {
			// baru:
			c.errorf(d.Span, "variable %q already declared at %s:%d", d.Name, prev.Span.File, prev.Span.Start.Line)
			continue
		}
		v, ok := constValue(d.Value)
		if !ok {
			c.errorf(d.Value.Pos(), "default value of %q must be a literal", d.Name)
			v = value.OfNil()
		}
		c.varDecls[d.Name] = d
		c.vars[d.Name] = len(c.prog.Vars)
		c.prog.Vars = append(c.prog.Vars, Var{Name: d.Name, Default: v})
	}

	for _, l := range p.Labels {
		key := qualifiedName(l)
		if prev, dup := c.labelDecls[key]; dup {
			c.errorf(l.Span, "label %q already defined at %s:%d", key, prev.Span.File, prev.Span.Start.Line)
			continue
		}
		c.labelDecls[key] = l
	}

	if _, ok := c.labelDecls[EntryLabel]; !ok {
		at := script.Position{Line: 1, Column: 1}
		c.errorf(script.Span{File: p.Span.File, Start: at, End: at},
			"missing entry label %q", EntryLabel)
	}
}

// constValue evaluates a default: a literal, or a negated numeric literal.
func constValue(e script.Expression) (value.Value, bool) {
	switch n := e.(type) {
	case *script.LiteralExpr:
		return n.Value, true
	case *script.UnaryExpr:
		if n.Op != script.UnaryNeg {
			break
		}
		lit, ok := n.Operand.(*script.LiteralExpr)
		if !ok {
			break
		}
		switch lit.Value.Type {
		case value.Int:
			return value.OfInt(-lit.Value.Int), true
		case value.Float:
			return value.OfFloat(-lit.Value.Float), true
		}
	}
	return value.Value{}, false
}

func (c *compiler) validate(p *script.ProgramNode) {
	for _, l := range p.Labels {
		c.curNS = l.Namespace
		c.checkBlock(l.Body)
	}
	c.curNS = ""
}

func (c *compiler) checkBlock(stmts []script.Statement) {
	for _, s := range stmts {
		c.checkStmt(s)
	}
}

func (c *compiler) checkStmt(s script.Statement) {
	switch n := s.(type) {
	case *script.SayNode:
		c.checkParts(n.Parts)
	case *script.ShowBackgroundNode:
		c.checkTransition(n.Transition, n.Span)
	case *script.ShowSpriteNode:
		c.checkTransition(n.Transition, n.Span)
	case *script.HideNode:
		c.checkTransition(n.Transition, n.Span)
	case *script.ReturnNode, *script.EndNode:
		// nothing to validate
	case *script.SetNode:
		c.needVar(n.Name, n.Span)
		c.checkExpr(n.Value)
	case *script.JumpNode:
		c.needLabel(n.Label, n.Span)
	case *script.CallNode:
		c.needLabel(n.Label, n.Span)
	case *script.IfNode:
		c.checkExpr(n.Condition)
		c.checkBlock(n.Then)
		c.checkBlock(n.Else)
	case *script.ChoiceNode:
		for _, o := range n.Options {
			c.checkParts(o.Text)
			if o.Condition != nil {
				c.checkExpr(o.Condition)
			}
			c.checkBlock(o.Body)
		}
	default:
		c.errorf(s.Pos(), "unsupported statement %T", s)
	}
}

func (c *compiler) checkParts(parts []script.TextPart) {
	for _, p := range parts {
		if v, ok := p.(*script.TextVariable); ok {
			c.needVar(v.Name, v.Span)
		}
	}
}

func (c *compiler) checkExpr(e script.Expression) {
	switch n := e.(type) {
	case *script.LiteralExpr:
	case *script.VariableExpr:
		c.needVar(n.Name, n.Span)
	case *script.BinaryExpr:
		c.checkExpr(n.Left)
		c.checkExpr(n.Right)
	case *script.UnaryExpr:
		c.checkExpr(n.Operand)
	default:
		c.errorf(e.Pos(), "unsupported expression %T", e)
	}
}

func (c *compiler) needVar(name string, span script.Span) {
	if _, ok := c.vars[name]; !ok {
		hint := ""
		if names := c.varNames(); len(names) > 0 {
			hint = "available: " + strings.Join(names, ", ")
		}
		c.errorHintf(span, hint, "undeclared variable %q (declare it with `default`)", name)
	}
}

func (c *compiler) needLabel(name string, span script.Span) {
	if _, ok := c.labelDecls[c.resolveLabel(name)]; !ok {
		hint := ""
		if names := c.labelNames(); len(names) > 0 {
			hint = "available: " + strings.Join(names, ", ")
		}
		c.errorHintf(span, hint, "undefined label %q", name)
	}
}

// checkTransition rejects a negative duration. The transition Name itself
// is intentionally never validated here: it's opaque data the rendering
// backend interprets, so new transitions need no compiler change.
func (c *compiler) checkTransition(t *script.Transition, span script.Span) {
	if t != nil && t.Duration < 0 {
		c.errorf(span, "transition duration must not be negative, got %g", t.Duration)
	}
}

// varNames returns declared variable names, sorted, for error hints.
func (c *compiler) varNames() []string {
	names := make([]string, 0, len(c.vars))
	for n := range c.vars {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// labelNames returns declared label names, sorted, for error hints.
func (c *compiler) labelNames() []string {
	names := make([]string, 0, len(c.labelDecls))
	for n := range c.labelDecls {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// ============================================================
// Pass 2: emit
// ============================================================

func (c *compiler) emitProgram(p *script.ProgramNode) {
	for _, l := range p.Labels {
		c.curNS = l.Namespace
		c.prog.Labels[qualifiedName(l)] = c.pc()
		c.block(l.Body)
		if len(l.Body) == 0 || !isTerminal(l.Body[len(l.Body)-1]) {
			c.emit(OpEnd, 0, l.Span)
		}
	}
	c.curNS = ""
	for _, f := range c.fixups {
		c.prog.Code[f.pc].A = c.prog.Labels[f.label]
	}
	c.prog.Entry = c.prog.Labels[EntryLabel]
}

func isTerminal(s script.Statement) bool {
	switch s.(type) {
	case *script.JumpNode, *script.ReturnNode, *script.EndNode:
		return true
	}
	return false
}

func (c *compiler) pc() int { return len(c.prog.Code) }

func (c *compiler) emit(op Opcode, a int, span script.Span) int {
	c.prog.Code = append(c.prog.Code, Instruction{Op: op, A: a})
	c.prog.Spans = append(c.prog.Spans, span)
	return c.pc() - 1
}

func (c *compiler) patch(at, target int) { c.prog.Code[at].A = target }

func (c *compiler) constant(v value.Value) int {
	if i, ok := c.consts[v]; ok {
		return i
	}
	i := len(c.prog.Consts)
	c.prog.Consts = append(c.prog.Consts, v)
	c.consts[v] = i
	return i
}

// compileTransition copies an AST transition into its compiler-level form.
// A nil input (no "with" clause) stays nil.
func compileTransition(t *script.Transition) *Transition {
	if t == nil {
		return nil
	}
	return &Transition{Name: t.Name, Duration: t.Duration}
}

func (c *compiler) block(stmts []script.Statement) {
	for _, s := range stmts {
		c.stmt(s)
	}
}

func (c *compiler) stmt(s script.Statement) {
	switch n := s.(type) {
	case *script.SayNode:
		c.prog.Says = append(c.prog.Says, Say{ID: n.ID, Speaker: n.Speaker, Parts: c.segments(n.Parts)})
		c.emit(OpSay, len(c.prog.Says)-1, n.Span)

	case *script.ShowBackgroundNode:
		idx := len(c.prog.Scenes)
		c.prog.Scenes = append(c.prog.Scenes, SceneCmd{Asset: n.Asset, Transition: compileTransition(n.Transition)})
		c.emit(OpScene, idx, n.Span)

	case *script.ShowSpriteNode:
		c.prog.Sprites = append(c.prog.Sprites, Sprite{
			Character:  n.Character,
			Expression: n.Expression,
			Position:   n.Position,
			Flip:       n.Flip,
			Transition: compileTransition(n.Transition),
		})
		c.emit(OpShow, len(c.prog.Sprites)-1, n.Span)

	case *script.HideNode:
		idx := len(c.prog.Hides)
		c.prog.Hides = append(c.prog.Hides, HideCmd{Target: n.Target, Transition: compileTransition(n.Transition)})
		c.emit(OpHide, idx, n.Span)

	case *script.SetNode:
		c.expr(n.Value)
		c.emit(OpStore, c.vars[n.Name], n.Span)

	case *script.JumpNode:
		at := c.emit(OpJump, 0, n.Span)
		c.fixups = append(c.fixups, fixup{pc: at, label: c.resolveLabel(n.Label)})

	case *script.CallNode:
		at := c.emit(OpCall, 0, n.Span)
		c.fixups = append(c.fixups, fixup{pc: at, label: c.resolveLabel(n.Label)})

	case *script.ReturnNode:
		c.emit(OpReturn, 0, n.Span)

	case *script.EndNode:
		c.emit(OpEnd, 0, n.Span)

	case *script.IfNode:
		c.ifStmt(n)

	case *script.ChoiceNode:
		c.choice(n)

	default:
		c.errorf(s.Pos(), "internal error: cannot compile %T", s)
	}
}

func (c *compiler) ifStmt(n *script.IfNode) {
	c.expr(n.Condition)
	jf := c.emit(OpJumpIfFalse, 0, n.Span)
	c.block(n.Then)
	if len(n.Else) == 0 {
		c.patch(jf, c.pc())
		return
	}
	skip := c.emit(OpJump, 0, n.Span)
	c.patch(jf, c.pc())
	c.block(n.Else)
	c.patch(skip, c.pc())
}

// choice layout:
//
//	<cond 0> ... <cond N-1>   (option without condition pushes true)
//	CHOICE k
//	<body 0>  JUMP end
//	<body 1>  JUMP end
//	end:
func (c *compiler) choice(n *script.ChoiceNode) {
	for _, o := range n.Options {
		if o.Condition != nil {
			c.expr(o.Condition)
		} else {
			c.emit(OpConst, c.constant(value.OfBool(true)), o.Span)
		}
	}

	idx := len(c.prog.Choices)
	c.prog.Choices = append(c.prog.Choices, Choice{}) // reserve; nested choices append after
	c.emit(OpChoice, idx, n.Span)

	info := Choice{Options: make([]ChoiceOption, 0, len(n.Options))}
	var toEnd []int
	for _, o := range n.Options {
		info.Options = append(info.Options, ChoiceOption{Text: c.segments(o.Text), Target: c.pc()})
		c.block(o.Body)
		toEnd = append(toEnd, c.emit(OpJump, 0, o.Span))
	}
	for _, at := range toEnd {
		c.patch(at, c.pc())
	}
	c.prog.Choices[idx] = info
}

func (c *compiler) segments(parts []script.TextPart) []Segment {
	out := make([]Segment, 0, len(parts))
	for _, p := range parts {
		switch n := p.(type) {
		case *script.TextLiteral:
			out = append(out, Segment{Literal: n.Text})
		case *script.TextVariable:
			out = append(out, Segment{IsVar: true, Name: n.Name, Slot: c.vars[n.Name]})
		}
	}
	return out
}

var binaryOps = map[script.BinaryOp]Opcode{
	script.BinaryAdd:          OpAdd,
	script.BinarySub:          OpSub,
	script.BinaryMul:          OpMul,
	script.BinaryDiv:          OpDiv,
	script.BinaryEqual:        OpEq,
	script.BinaryNotEqual:     OpNeq,
	script.BinaryLess:         OpLt,
	script.BinaryLessEqual:    OpLe,
	script.BinaryGreater:      OpGt,
	script.BinaryGreaterEqual: OpGe,
	script.BinaryAnd:          OpAnd,
	script.BinaryOr:           OpOr,
}

func (c *compiler) expr(e script.Expression) {
	switch n := e.(type) {
	case *script.LiteralExpr:
		c.emit(OpConst, c.constant(n.Value), n.Span)
	case *script.VariableExpr:
		c.emit(OpLoad, c.vars[n.Name], n.Span)
	case *script.BinaryExpr:
		c.expr(n.Left)
		c.expr(n.Right)
		c.emit(binaryOps[n.Op], 0, n.Span)
	case *script.UnaryExpr:
		c.expr(n.Operand)
		if n.Op == script.UnaryNot {
			c.emit(OpNot, 0, n.Span)
		} else {
			c.emit(OpNeg, 0, n.Span)
		}
	default:
		c.errorf(e.Pos(), "internal error: cannot compile %T", e)
	}
}

// CompileFiles merges multiple parsed files (e.g. the chapters of a story
// split across a directory) into one Program, as if their declarations and
// labels had all been written in a single file. Labels and variables must
// still be unique across all of them; duplicates are reported with the
// exact file and line of both occurrences, since every Span already
// carries its source file.
func CompileFiles(asts ...*script.ProgramNode) (*Program, error) {
	merged := &script.ProgramNode{}
	for _, a := range asts {
		merged.Declarations = append(merged.Declarations, a.Declarations...)
		merged.Labels = append(merged.Labels, a.Labels...)
	}
	return Compile(merged)
}

func qualifiedName(l *script.LabelNode) string {
	if l.Namespace == "" {
		return l.Name
	}
	return l.Namespace + "." + l.Name
}

// resolveLabel: nama berkualifikasi dipakai apa adanya; nama bare dicari
// di namespace saat ini dulu, lalu global.
func (c *compiler) resolveLabel(name string) string {
	if strings.Contains(name, ".") {
		return name
	}
	if c.curNS != "" {
		q := c.curNS + "." + name
		if _, ok := c.labelDecls[q]; ok {
			return q
		}
	}
	return name
}
