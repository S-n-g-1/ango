package script

import (
	"fmt"
	"strings"
	"testing"
)

// ============================================================
// ADAPTER: satu-satunya tempat yang bergantung pada API parser.
// ============================================================

func parseSource(src string) (*ProgramNode, error) {
	return Parse("test.ango", src)
}

// literalText menampilkan literal sebagai teks singkat.
// Jika value.Value nanti punya String(), fmt.Sprint memakainya langsung.
// Sebelum itu, format struct mentah "{kind bool num float str}" dibaca
// secara heuristik: kind 1 = bool, kind 2 = angka.
func literalText(l *LiteralExpr) string {
	s := fmt.Sprint(l.Value)
	if !strings.HasPrefix(s, "{") {
		return s
	}
	f := strings.Fields(strings.Trim(s, "{}"))
	if len(f) >= 3 {
		switch f[0] {
		case "1":
			return f[1]
		case "2":
			return f[2]
		}
	}
	return s
}

// ============================================================
// Helpers
// ============================================================

func mustParse(t *testing.T, src string) *ProgramNode {
	t.Helper()
	prog, err := parseSource(src)
	if err != nil {
		t.Fatalf("parse error: %v\nsource:\n%s", err, src)
	}
	if prog == nil {
		t.Fatalf("parse returned nil program")
	}
	return prog
}

// body membungkus baris-baris statement dalam satu label lalu mengembalikan
// body-nya. Body label tidak diindentasi; indentasi hanya di dalam
// if/else/choice, jadi baris dibawa apa adanya.
func body(t *testing.T, lines ...string) []Statement {
	t.Helper()
	src := "label start\n" + strings.Join(lines, "\n") + "\n"
	prog := mustParse(t, src)
	if len(prog.Labels) != 1 {
		t.Fatalf("want 1 label, got %d", len(prog.Labels))
	}
	return prog.Labels[0].Body
}

func one[T Statement](t *testing.T, stmts []Statement) T {
	t.Helper()
	if len(stmts) != 1 {
		t.Fatalf("want 1 statement, got %d", len(stmts))
	}
	got, ok := stmts[0].(T)
	if !ok {
		var zero T
		t.Fatalf("want %T, got %T", zero, stmts[0])
	}
	return got
}

var binOps = map[BinaryOp]string{
	BinaryAdd: "+", BinarySub: "-", BinaryMul: "*", BinaryDiv: "/",
	BinaryEqual: "==", BinaryNotEqual: "!=",
	BinaryLess: "<", BinaryLessEqual: "<=",
	BinaryGreater: ">", BinaryGreaterEqual: ">=",
	BinaryAnd: "and", BinaryOr: "or",
}

var unOps = map[UnaryOp]string{UnaryNot: "not", UnaryNeg: "neg"}

// exprString mengubah AST ekspresi menjadi s-expression agar
// struktur precedence/asosiatif bisa diverifikasi persis.
func exprString(e Expression) string {
	switch n := e.(type) {
	case nil:
		return "<nil>"
	case *LiteralExpr:
		return literalText(n)
	case *VariableExpr:
		return n.Name
	case *BinaryExpr:
		return fmt.Sprintf("(%s %s %s)", binOps[n.Op], exprString(n.Left), exprString(n.Right))
	case *UnaryExpr:
		return fmt.Sprintf("(%s %s)", unOps[n.Op], exprString(n.Operand))
	}
	return fmt.Sprintf("<unknown %T>", e)
}

func partsString(parts []TextPart) string {
	var out []string
	for _, p := range parts {
		switch n := p.(type) {
		case *TextLiteral:
			out = append(out, "L:"+n.Text)
		case *TextVariable:
			out = append(out, "V:"+n.Name)
		default:
			out = append(out, fmt.Sprintf("?%T", p))
		}
	}
	return strings.Join(out, "|")
}

func assertEq(t *testing.T, what, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s:\n  got  %q\n  want %q", what, got, want)
	}
}

// ============================================================
// Program: default & label
// ============================================================

func TestDefaults(t *testing.T) {
	prog := mustParse(t, "default found_key = false\ndefault courage = 0\n\nlabel start\nend\n")

	if len(prog.Declarations) != 2 {
		t.Fatalf("want 2 defaults, got %d", len(prog.Declarations))
	}
	assertEq(t, "default[0].Name", prog.Declarations[0].Name, "found_key")
	assertEq(t, "default[0].Value", exprString(prog.Declarations[0].Value), "false")
	assertEq(t, "default[1].Name", prog.Declarations[1].Name, "courage")
	assertEq(t, "default[1].Value", exprString(prog.Declarations[1].Value), "0")
}

func TestLabelsKeepOrderAndBody(t *testing.T) {
	prog := mustParse(t, `label start
Asep: "Halo."

label hallway
Asep: "Kita di lorong."
end
`)
	if len(prog.Labels) != 2 {
		t.Fatalf("want 2 labels, got %d", len(prog.Labels))
	}
	assertEq(t, "label[0]", prog.Labels[0].Name, "start")
	assertEq(t, "label[1]", prog.Labels[1].Name, "hallway")
	if len(prog.Labels[0].Body) != 1 {
		t.Errorf("start body: want 1 stmt, got %d", len(prog.Labels[0].Body))
	}
	if len(prog.Labels[1].Body) != 2 {
		t.Errorf("hallway body: want 2 stmts, got %d", len(prog.Labels[1].Body))
	}
}

func TestEmptyProgram(t *testing.T) {
	prog := mustParse(t, "")
	if len(prog.Labels) != 0 || len(prog.Declarations) != 0 {
		t.Errorf("empty source should give empty program")
	}
}

// ============================================================
// Dialogue & narration
// ============================================================

func TestSay(t *testing.T) {
	s := one[*SayNode](t, body(t, `Asep: "Halo."`))
	assertEq(t, "speaker", s.Speaker, "Asep")
	assertEq(t, "id", s.ID, "")
	assertEq(t, "parts", partsString(s.Parts), "L:Halo.")
}

func TestSayWithID(t *testing.T) {
	s := one[*SayNode](t, body(t, `@id("intro_001")`, `Asep: "Halo."`))
	assertEq(t, "id", s.ID, "intro_001")
	assertEq(t, "speaker", s.Speaker, "Asep")
}

func TestNarration(t *testing.T) {
	s := one[*SayNode](t, body(t, `"Malam itu sunyi."`))
	assertEq(t, "speaker", s.Speaker, "")
	assertEq(t, "parts", partsString(s.Parts), "L:Malam itu sunyi.")
}

func TestInterpolation(t *testing.T) {
	s := one[*SayNode](t, body(t, `Asep: "Halo [name]!"`))
	assertEq(t, "parts", partsString(s.Parts), "L:Halo |V:name|L:!")
}

func TestInterpolationMultipleAndEdges(t *testing.T) {
	s := one[*SayNode](t, body(t, `Asep: "[a][b] dan [c]"`))
	assertEq(t, "parts", partsString(s.Parts), "V:a|V:b|L: dan |V:c")
}

// ============================================================
// Scene / show / hide
// ============================================================

func TestScene(t *testing.T) {
	s := one[*ShowBackgroundNode](t, body(t, `scene "classroom"`))
	assertEq(t, "asset", s.Asset, "classroom")
}

func TestShowSprite(t *testing.T) {
	s := one[*ShowSpriteNode](t, body(t, `show asep "happy" at left`))
	assertEq(t, "character", s.Character, "asep")
	assertEq(t, "expression", s.Expression, "happy")
	assertEq(t, "position", s.Position, "left")
}

func TestHide(t *testing.T) {
	s := one[*HideNode](t, body(t, `hide asep`))
	assertEq(t, "target", s.Target, "asep")
}

// ============================================================
// Set & flow control
// ============================================================

func TestSet(t *testing.T) {
	s := one[*SetNode](t, body(t, `set courage = courage + 1`))
	assertEq(t, "name", s.Name, "courage")
	assertEq(t, "value", exprString(s.Value), "(+ courage 1)")
}

func TestJumpCallReturnEnd(t *testing.T) {
	stmts := body(t, `jump hallway`, `call shop`, `return`, `end`)
	if len(stmts) != 4 {
		t.Fatalf("want 4 statements, got %d", len(stmts))
	}
	j, ok := stmts[0].(*JumpNode)
	if !ok || j.Label != "hallway" {
		t.Errorf("stmt0: want Jump hallway, got %#v", stmts[0])
	}
	c, ok := stmts[1].(*CallNode)
	if !ok || c.Label != "shop" {
		t.Errorf("stmt1: want Call shop, got %#v", stmts[1])
	}
	if _, ok := stmts[2].(*ReturnNode); !ok {
		t.Errorf("stmt2: want Return, got %T", stmts[2])
	}
	if _, ok := stmts[3].(*EndNode); !ok {
		t.Errorf("stmt3: want End, got %T", stmts[3])
	}
}

// ============================================================
// If / else
// ============================================================

func TestIfWithoutElse(t *testing.T) {
	n := one[*IfNode](t, body(t,
		`if found_key:`,
		`    Asep: "Ada kunci."`,
	))
	assertEq(t, "condition", exprString(n.Condition), "found_key")
	if len(n.Then) != 1 {
		t.Errorf("then: want 1, got %d", len(n.Then))
	}
	if len(n.Else) != 0 {
		t.Errorf("else: want 0, got %d", len(n.Else))
	}
}

func TestIfElse(t *testing.T) {
	n := one[*IfNode](t, body(t,
		`if courage >= 3:`,
		`    jump brave`,
		`else:`,
		`    jump coward`,
		`    end`,
	))
	assertEq(t, "condition", exprString(n.Condition), "(>= courage 3)")
	if len(n.Then) != 1 || len(n.Else) != 2 {
		t.Fatalf("want then=1 else=2, got then=%d else=%d", len(n.Then), len(n.Else))
	}
	if j, ok := n.Then[0].(*JumpNode); !ok || j.Label != "brave" {
		t.Errorf("then[0]: want Jump brave, got %#v", n.Then[0])
	}
	if j, ok := n.Else[0].(*JumpNode); !ok || j.Label != "coward" {
		t.Errorf("else[0]: want Jump coward, got %#v", n.Else[0])
	}
}

func TestNestedIf(t *testing.T) {
	n := one[*IfNode](t, body(t,
		`if a:`,
		`    if b:`,
		`        end`,
	))
	inner := one[*IfNode](t, n.Then)
	assertEq(t, "inner cond", exprString(inner.Condition), "b")
}

// ============================================================
// Expression precedence & associativity
// ============================================================

func TestExpressionPrecedence(t *testing.T) {
	tests := []struct{ src, want string }{
		{"1 + 2 * 3", "(+ 1 (* 2 3))"},
		{"1 * 2 + 3", "(+ (* 1 2) 3)"},
		{"(1 + 2) * 3", "(* (+ 1 2) 3)"},
		{"1 - 2 - 3", "(- (- 1 2) 3)"},
		{"8 / 4 / 2", "(/ (/ 8 4) 2)"},
		{"a or b and c", "(or a (and b c))"},
		{"a and b or c", "(or (and a b) c)"},
		{"not a and b", "(and (not a) b)"},
		{"not a == b", "(not (== a b))"},
		{"-x + 1", "(+ (neg x) 1)"},
		{"a + 1 >= b * 2", "(>= (+ a 1) (* b 2))"},
		{"a != b or c < d", "(or (!= a b) (< c d))"},
		{"a <= 1 and b > 2", "(and (<= a 1) (> b 2))"},
	}
	for _, tc := range tests {
		t.Run(tc.src, func(t *testing.T) {
			s := one[*SetNode](t, body(t, "set v = "+tc.src))
			assertEq(t, "expr", exprString(s.Value), tc.want)
		})
	}
}

// ============================================================
// Choice
// ============================================================

func TestChoiceWithAndWithoutCondition(t *testing.T) {
	c := one[*ChoiceNode](t, body(t,
		`choice:`,
		`    "Ambil kunci" if not found_key:`,
		`        set found_key = true`,
		``,
		`    "Periksa meja":`,
		`        set courage = courage + 1`,
	))
	if len(c.Options) != 2 {
		t.Fatalf("want 2 options, got %d", len(c.Options))
	}

	o0, o1 := c.Options[0], c.Options[1]
	assertEq(t, "opt0 text", partsString(o0.Text), "L:Ambil kunci")
	assertEq(t, "opt0 cond", exprString(o0.Condition), "(not found_key)")
	set0 := one[*SetNode](t, o0.Body)
	assertEq(t, "opt0 set", set0.Name+"="+exprString(set0.Value), "found_key=true")

	assertEq(t, "opt1 text", partsString(o1.Text), "L:Periksa meja")
	if o1.Condition != nil {
		t.Errorf("opt1 cond: want nil, got %s", exprString(o1.Condition))
	}
	set1 := one[*SetNode](t, o1.Body)
	assertEq(t, "opt1 set", set1.Name+"="+exprString(set1.Value), "courage=(+ courage 1)")
}

func TestChoiceOptionInterpolationAndJump(t *testing.T) {
	c := one[*ChoiceNode](t, body(t,
		`choice:`,
		`    "Ikut [friend]":`,
		`        jump follow`,
	))
	assertEq(t, "text", partsString(c.Options[0].Text), "L:Ikut |V:friend")
	j := one[*JumpNode](t, c.Options[0].Body)
	assertEq(t, "jump", j.Label, "follow")
}

// ============================================================
// Full script (integrasi)
// ============================================================

func TestFullScript(t *testing.T) {
	prog := mustParse(t, `default found_key = false
default courage = 0

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
	if len(prog.Declarations) != 2 || len(prog.Labels) != 2 {
		t.Fatalf("want 2 defaults + 2 labels, got %d + %d", len(prog.Declarations), len(prog.Labels))
	}
	kinds := []string{}
	for _, s := range prog.Labels[0].Body {
		kinds = append(kinds, fmt.Sprintf("%T", s))
	}
	assertEq(t, "start body kinds", strings.Join(kinds, ","),
		"*script.ShowBackgroundNode,*script.ShowSpriteNode,*script.SayNode,*script.ChoiceNode")
	say := prog.Labels[0].Body[2].(*SayNode)
	assertEq(t, "say id", say.ID, "intro_001")
}

// ============================================================
// Syntax errors: parser HARUS menolak, bukan menghasilkan AST diam-diam.
// ============================================================

func TestSyntaxErrors(t *testing.T) {
	tests := []struct{ name, src string }{
		{"label tanpa nama", "label\nend\n"},
		{"body label diindentasi", "label start\n    end\n"},
		{"jump tanpa target", "label start\njump\n"},
		{"call tanpa target", "label start\ncall\n"},
		{"set tanpa nilai", "label start\nset x =\n"},
		{"set tanpa assignment", "label start\nset x\n"},
		{"if tanpa kondisi", "label start\nif:\n    end\n"},
		{"else tanpa if", "label start\nelse:\n    end\n"},
		{"string tak tertutup", "label start\nAsep: \"Halo\n"},
		{"interpolasi tak tertutup", "label start\nAsep: \"Halo [name\"\n"},
		{"kurung tak tertutup", "label start\nset x = (1 + 2\n"},
		{"operator tanpa operand kanan", "label start\nset x = 1 +\n"},
		{"default tanpa nilai", "default x =\n"},
		{"choice kosong", "label start\nchoice:\n"},
		{"statement di luar label", "Asep: \"Halo.\"\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog, err := parseSource(tc.src)
			if err == nil {
				t.Fatalf("want error, got nil (program: %+v)", prog)
			}
		})
	}
}
