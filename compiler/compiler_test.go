package compiler

import (
	"errors"
	"strings"
	"testing"

	"ango/engine/script"
	"ango/engine/value"
)

func compileSrc(t *testing.T, src string) (*Program, error) {
	t.Helper()
	ast, err := script.Parse("test.ango", src)
	if err != nil {
		t.Fatalf("parse error: %v\nsource:\n%s", err, src)
	}
	return Compile(ast)
}

func mustCompile(t *testing.T, src string) *Program {
	t.Helper()
	p, err := compileSrc(t, src)
	if err != nil {
		if es, ok := err.(Errors); ok {
			t.Fatalf("compile errors:\n%s", es.Details())
		}
		t.Fatalf("compile error: %v", err)
	}
	return p
}

func norm(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := range lines {
		lines[i] = strings.Join(strings.Fields(lines[i]), " ")
	}
	return strings.Join(lines, "\n")
}

func assertCode(t *testing.T, p *Program, want string) {
	t.Helper()
	if got := norm(p.Disassemble()); got != norm(want) {
		t.Errorf("bytecode mismatch\n--- got ---\n%s\n--- want ---\n%s", got, norm(want))
	}
}

// ============================================================
// Bytecode
// ============================================================

func TestSetAndImplicitEnd(t *testing.T) {
	p := mustCompile(t, "default courage = 0\nlabel start\nset courage = courage + 1\n")
	assertCode(t, p, `
0000 LOAD courage
0001 CONST 1
0002 ADD
0003 STORE courage
0004 END`)
	if p.Vars[0].Name != "courage" || p.Vars[0].Default != value.OfInt(0) {
		t.Errorf("bad var: %+v", p.Vars[0])
	}
}

func TestNoImplicitEndAfterTerminal(t *testing.T) {
	p := mustCompile(t, "label start\njump second\nlabel second\nend\n")
	assertCode(t, p, `
0000 JUMP 1
0001 END`)
	if p.Labels["second"] != 1 || p.Entry != 0 {
		t.Errorf("labels=%v entry=%d", p.Labels, p.Entry)
	}
}

func TestCallReturn(t *testing.T) {
	p := mustCompile(t, "label start\ncall sub\nend\nlabel sub\nreturn\n")
	assertCode(t, p, `
0000 CALL 2
0001 END
0002 RETURN`)
}

func TestIfElse(t *testing.T) {
	p := mustCompile(t, "default a = true\nlabel start\nif a:\n    jump start\nelse:\n    end\n")
	assertCode(t, p, `
0000 LOAD a
0001 JUMPIFFALSE 4
0002 JUMP 0
0003 JUMP 5
0004 END
0005 END`)
}

func TestIfWithoutElse(t *testing.T) {
	p := mustCompile(t, "default a = true\nlabel start\nif a:\n    end\nend\n")
	assertCode(t, p, `
0000 LOAD a
0001 JUMPIFFALSE 3
0002 END
0003 END`)
}

func TestChoiceLayout(t *testing.T) {
	src := `default k = false
default c = 0
label start
choice:
    "A" if not k:
        set k = true
    "B":
        set c = c + 1
`
	p := mustCompile(t, src)
	assertCode(t, p, `
0000 LOAD k
0001 NOT
0002 CONST true
0003 CHOICE 0
0004 CONST true
0005 STORE k
0006 JUMP 12
0007 LOAD c
0008 CONST 1
0009 ADD
0010 STORE c
0011 JUMP 12
0012 END`)
	ch := p.Choices[0]
	if len(ch.Options) != 2 || ch.Options[0].Target != 4 || ch.Options[1].Target != 7 {
		t.Errorf("bad choice table: %+v", ch)
	}
}

func TestSayWithInterpolation(t *testing.T) {
	p := mustCompile(t, "default name = \"Asep\"\nlabel start\n@id(\"i1\")\nAsep: \"Halo [name]!\"\n")
	assertCode(t, p, "0000 SAY 0\n0001 END")
	s := p.Says[0]
	if s.ID != "i1" || s.Speaker != "Asep" || len(s.Parts) != 3 {
		t.Fatalf("bad say: %+v", s)
	}
	if s.Parts[0].Literal != "Halo " || !s.Parts[1].IsVar || s.Parts[1].Slot != 0 || s.Parts[2].Literal != "!" {
		t.Errorf("bad parts: %+v", s.Parts)
	}
}

func TestSceneShowHide(t *testing.T) {
	p := mustCompile(t, "label start\nscene \"classroom\"\nshow asep \"happy\" at left\nhide asep\n")
	assertCode(t, p, `
0000 SCENE "classroom"
0001 SHOW 0
0002 HIDE "asep"
0003 END`)
	if p.Sprites[0] != (Sprite{Character: "asep", Expression: "happy", Position: "left"}) {
		t.Errorf("bad sprite: %+v", p.Sprites[0])
	}
}

func TestConstantsAreDeduplicated(t *testing.T) {
	p := mustCompile(t, "default x = 0\nlabel start\nset x = 1\nset x = 1\n")
	if len(p.Consts) != 1 {
		t.Errorf("want 1 const, got %d: %v", len(p.Consts), p.Consts)
	}
}

func TestNegativeDefault(t *testing.T) {
	p := mustCompile(t, "default x = -5\nlabel start\nend\n")
	if p.Vars[0].Default != value.OfInt(-5) {
		t.Errorf("want -5, got %+v", p.Vars[0].Default)
	}
}

func TestSpansParallelToCode(t *testing.T) {
	p := mustCompile(t, "label start\nend\n")
	if len(p.Spans) != len(p.Code) {
		t.Fatalf("spans=%d code=%d", len(p.Spans), len(p.Code))
	}
	if p.Spans[0].Start.Line != 2 {
		t.Errorf("END should map to line 2, got %d", p.Spans[0].Start.Line)
	}
}

func TestFullScriptCompiles(t *testing.T) {
	p := mustCompile(t, `default found_key = false
default courage = 0
default name = "Asep"

label start
scene "classroom"
show asep "happy" at left
@id("intro_001")
Asep: "Halo [name]!"
choice:
    "Ambil kunci" if not found_key:
        set found_key = true
        jump hallway
    "Periksa meja":
        set courage = courage + 1

label hallway
if found_key:
    Asep: "Kita di lorong."
else:
    end
`)
	if _, ok := p.Labels["hallway"]; !ok {
		t.Errorf("hallway label missing")
	}
	// every jump-like target must be a valid pc
	for pc, in := range p.Code {
		switch in.Op {
		case OpJump, OpJumpIfFalse, OpCall:
			if in.A < 0 || in.A >= len(p.Code) {
				t.Errorf("pc %d: %s target %d out of range (len %d)", pc, in.Op, in.A, len(p.Code))
			}
		}
	}
}

// ============================================================
// Semantic errors
// ============================================================

func TestSemanticErrors(t *testing.T) {
	tests := []struct{ name, src, want string }{
		{"no start label", "label other\nend\n", "missing entry label"},
		{"duplicate label", "label start\nend\nlabel start\nend\n", "already defined"},
		{"duplicate default", "default x = 1\ndefault x = 2\nlabel start\nend\n", "already declared"},
		{"jump undefined", "label start\njump nowhere\n", "undefined label"},
		{"call undefined", "label start\ncall nowhere\n", "undefined label"},
		{"set undeclared", "label start\nset x = 1\n", "undeclared variable"},
		{"expr undeclared", "label start\nif y:\n    end\n", "undeclared variable"},
		{"interpolation undeclared", "label start\nAsep: \"Hi [who]\"\n", "undeclared variable"},
		{"choice condition undeclared", "label start\nchoice:\n    \"A\" if z:\n        end\n", "undeclared variable"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := compileSrc(t, tc.src)
			if err == nil {
				t.Fatalf("want error containing %q, got none", tc.want)
			}
			if p != nil {
				t.Errorf("program must be nil on error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.want)
			}
		})
	}
}

func TestDefaultMustBeLiteral(t *testing.T) {
	// Parser sudah menolak `default x = 1 + 2`, jadi AST dibangun manual
	// untuk menguji pertahanan di compiler.
	ast := &script.ProgramNode{
		Declarations: []*script.DefaultNode{{
			Name: "x",
			Value: &script.BinaryExpr{
				Left:  &script.LiteralExpr{Value: value.OfInt(1)},
				Op:    script.BinaryAdd,
				Right: &script.LiteralExpr{Value: value.OfInt(2)},
			},
		}},
		Labels: []*script.LabelNode{{
			Name: "start",
			Body: []script.Statement{&script.EndNode{}},
		}},
	}

	p, err := Compile(ast)
	if err == nil {
		t.Fatalf("want error, got none")
	}
	if p != nil {
		t.Errorf("program must be nil on error")
	}
	if !strings.Contains(err.Error(), "must be a literal") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestErrorsAreCollectedWithPositions(t *testing.T) {
	_, err := compileSrc(t, "label start\njump a\njump b\n")
	var errs Errors
	if !errors.As(err, &errs) {
		t.Fatalf("want compiler.Errors, got %T", err)
	}
	if len(errs) != 2 {
		t.Fatalf("want 2 errors, got %d:\n%s", len(errs), errs.Details())
	}
	if errs[0].Span.Start.Line != 2 || errs[1].Span.Start.Line != 3 {
		t.Errorf("errors not positioned/sorted: %s", errs.Details())
	}
	if !strings.HasPrefix(errs[0].Error(), "test.ango:2:") {
		t.Errorf("want file:line prefix, got %q", errs[0].Error())
	}
}
