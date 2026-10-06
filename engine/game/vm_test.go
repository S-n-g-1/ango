package game

import (
	"strings"
	"testing"

	"ango/compiler"
	"ango/engine/script"
	"ango/engine/value"
)

func compile(t *testing.T, src string) *compiler.Program {
	t.Helper()
	ast, err := script.Parse("test.ango", src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	prog, err := compiler.Compile(ast)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	return prog
}

func newVM(t *testing.T, src string) *VM { t.Helper(); return New(compile(t, src)) }

func next(t *testing.T, vm *VM) Event {
	t.Helper()
	ev, err := vm.Next()
	if err != nil {
		t.Fatalf("Next error: %v", err)
	}
	return ev
}

func expectSay(t *testing.T, vm *VM, speaker, text string) {
	t.Helper()
	ev := next(t, vm)
	if ev.Kind != EventSay || ev.Speaker != speaker || ev.Text != text {
		t.Fatalf("want say %s:%q, got %+v", speaker, text, ev)
	}
}

func expectEnd(t *testing.T, vm *VM) {
	t.Helper()
	if ev := next(t, vm); ev.Kind != EventEnd {
		t.Fatalf("want end, got %+v", ev)
	}
}

func expectRuntimeError(t *testing.T, vm *VM, want string) *script.Error {
	t.Helper()
	for i := 0; i < 20; i++ {
		ev, err := vm.Next()
		if err != nil {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error %q does not contain %q", err.Error(), want)
			}
			se, ok := err.(*script.Error)
			if !ok {
				t.Fatalf("want *script.Error, got %T", err)
			}
			return se
		}
		if ev.Kind == EventEnd {
			break
		}
	}
	t.Fatalf("expected runtime error containing %q", want)
	return nil
}

func TestSayInterpolationAndSet(t *testing.T) {
	vm := newVM(t, "default n = 1\nlabel start\nAsep: \"n=[n]\"\nset n = n + 1\nAsep: \"n=[n]\"\n")
	expectSay(t, vm, "Asep", "n=1")
	expectSay(t, vm, "Asep", "n=2")
	expectEnd(t, vm)
	expectEnd(t, vm) // End is sticky
}

func TestSayCarriesID(t *testing.T) {
	vm := newVM(t, "label start\n@id(\"intro_001\")\nAsep: \"Halo.\"\n")
	if ev := next(t, vm); ev.ID != "intro_001" {
		t.Errorf("want id intro_001, got %q", ev.ID)
	}
}

func TestNarrationHasNoSpeaker(t *testing.T) {
	vm := newVM(t, "label start\n\"Malam itu sunyi.\"\n")
	expectSay(t, vm, "", "Malam itu sunyi.")
}

func TestArithmeticPrecedence(t *testing.T) {
	vm := newVM(t, "default r = 0\nlabel start\nset r = 1 + 2 * 3\nAsep: \"[r]\"\n")
	expectSay(t, vm, "Asep", "7")
}

func TestIntegerDivisionTruncates(t *testing.T) {
	vm := newVM(t, "default a = 7\ndefault r = 0\nlabel start\nset r = a / 2\nAsep: \"[r]\"\n")
	expectSay(t, vm, "Asep", "3")
}

func TestStringConcatAndFormat(t *testing.T) {
	vm := newVM(t, "default s = \"a\"\ndefault flag = true\nlabel start\nset s = s + \"b\"\nAsep: \"[s] [flag]\"\n")
	expectSay(t, vm, "Asep", "ab true")
}

func TestComparisonAndLogic(t *testing.T) {
	vm := newVM(t, "default a = 3\nlabel start\nif a >= 3 and a != 4:\n    Asep: \"yes\"\nelse:\n    Asep: \"no\"\n")
	expectSay(t, vm, "Asep", "yes")
}

func TestIfElseTakesElse(t *testing.T) {
	vm := newVM(t, "default a = false\nlabel start\nif a:\n    Asep: \"yes\"\nelse:\n    Asep: \"no\"\n")
	expectSay(t, vm, "Asep", "no")
	expectEnd(t, vm)
}

func TestCallReturn(t *testing.T) {
	vm := newVM(t, `label start
Asep: "1"
call sub
Asep: "3"
end
label sub
Asep: "2"
return
`)
	expectSay(t, vm, "Asep", "1")
	expectSay(t, vm, "Asep", "2")
	expectSay(t, vm, "Asep", "3")
	expectEnd(t, vm)
}

func TestSceneShowHide(t *testing.T) {
	vm := newVM(t, "label start\nscene \"bg\"\nshow asep \"happy\" at left\nhide asep\n")

	if ev := next(t, vm); ev.Kind != EventScene || ev.Asset != "bg" {
		t.Fatalf("want scene bg, got %+v", ev)
	}
	ev := next(t, vm)
	if ev.Kind != EventShow || ev.Sprite != (SpriteState{Character: "asep", Expression: "happy", Position: "left"}) {
		t.Fatalf("want show asep, got %+v", ev)
	}
	st := vm.State()
	if len(st.Sprites) != 1 || st.Background != "bg" {
		t.Errorf("bad state: %+v", st)
	}
	if ev := next(t, vm); ev.Kind != EventHide || ev.Target != "asep" {
		t.Fatalf("want hide asep, got %+v", ev)
	}
	if n := len(vm.State().Sprites); n != 0 {
		t.Errorf("want 0 sprites, got %d", n)
	}
	expectEnd(t, vm)
}

func TestChoiceFiltersDisabledOptions(t *testing.T) {
	vm := newVM(t, `default k = false
label start
choice:
    "A" if k:
        Asep: "chose A"
    "B":
        Asep: "chose B"
`)
	ev := next(t, vm)
	if ev.Kind != EventChoice || len(ev.Options) != 1 || ev.Options[0] != "B" {
		t.Fatalf("want only B, got %+v", ev)
	}
	if err := vm.Choose(0); err != nil {
		t.Fatal(err)
	}
	expectSay(t, vm, "Asep", "chose B")
	expectEnd(t, vm)
}

func TestChoiceAllEnabledPicksRightBody(t *testing.T) {
	vm := newVM(t, `default k = true
label start
choice:
    "A" if k:
        Asep: "chose A"
    "B":
        Asep: "chose B"
`)
	ev := next(t, vm)
	if len(ev.Options) != 2 || ev.Options[0] != "A" || ev.Options[1] != "B" {
		t.Fatalf("want A,B got %+v", ev)
	}
	vm.Choose(1)
	expectSay(t, vm, "Asep", "chose B")
}

func TestChoiceTextInterpolates(t *testing.T) {
	vm := newVM(t, "default friend = \"Ujang\"\nlabel start\nchoice:\n    \"Ikut [friend]\":\n        end\n")
	if ev := next(t, vm); ev.Options[0] != "Ikut Ujang" {
		t.Errorf("got %+v", ev)
	}
}

func TestChoiceErrorsAndRepeat(t *testing.T) {
	vm := newVM(t, "label start\nchoice:\n    \"A\":\n        end\n")
	if err := vm.Choose(0); err == nil {
		t.Errorf("Choose without pending choice must fail")
	}
	first := next(t, vm)
	again := next(t, vm) // Next before Choose repeats the event
	if first.Kind != EventChoice || again.Kind != EventChoice {
		t.Fatalf("want repeated choice event, got %+v / %+v", first, again)
	}
	if err := vm.Choose(5); err == nil {
		t.Errorf("out-of-range Choose must fail")
	}
	if err := vm.Choose(0); err != nil {
		t.Errorf("valid Choose failed: %v", err)
	}
}

func TestNoAvailableChoiceIsError(t *testing.T) {
	vm := newVM(t, "default k = false\nlabel start\nchoice:\n    \"A\" if k:\n        end\n")
	expectRuntimeError(t, vm, "no available options")
}

func TestDivisionByZeroHasSourcePosition(t *testing.T) {
	vm := newVM(t, "default r = 0\nlabel start\nset r = 1 / 0\n")
	se := expectRuntimeError(t, vm, "division by zero")
	if se.Span.Start.Line != 3 {
		t.Errorf("want line 3, got %d (%v)", se.Span.Start.Line, se)
	}
	if _, err := vm.Next(); err == nil {
		t.Errorf("error must be sticky")
	}
}

func TestTypeErrors(t *testing.T) {
	expectRuntimeError(t, newVM(t, "default s = \"a\"\nlabel start\nset s = s - 1\n"), "cannot apply")
	expectRuntimeError(t, newVM(t, "default n = 1\nlabel start\nif n:\n    end\n"), "expected bool")
}

func TestRuntimeGuards(t *testing.T) {
	expectRuntimeError(t, newVM(t, "label start\njump start\n"), "infinite loop")
	expectRuntimeError(t, newVM(t, "label start\ncall start\n"), "call stack overflow")
	expectRuntimeError(t, newVM(t, "label start\nreturn\n"), "return without call")
}

func TestSaveRestoreContinuesIdentically(t *testing.T) {
	prog := compile(t, "default n = 0\nlabel start\nAsep: \"one\"\nset n = n + 1\nAsep: \"two [n]\"\n")
	vm := New(prog)
	expectSay(t, vm, "Asep", "one")
	snap := vm.State()
	expectSay(t, vm, "Asep", "two 1")

	vm2 := New(prog)
	if err := vm2.Restore(snap); err != nil {
		t.Fatal(err)
	}
	expectSay(t, vm2, "Asep", "two 1")
	expectEnd(t, vm2)
}

func TestSaveRestorePendingChoice(t *testing.T) {
	prog := compile(t, "label start\nchoice:\n    \"A\":\n        Asep: \"a\"\n    \"B\":\n        Asep: \"b\"\n")
	vm := New(prog)
	next(t, vm)
	snap := vm.State()

	vm2 := New(prog)
	if err := vm2.Restore(snap); err != nil {
		t.Fatal(err)
	}
	if ev := next(t, vm2); ev.Kind != EventChoice || len(ev.Options) != 2 {
		t.Fatalf("pending choice not restored: %+v", ev)
	}
	vm2.Choose(0)
	expectSay(t, vm2, "Asep", "a")
}

func TestStateIsACopyAndRestoreValidates(t *testing.T) {
	prog := compile(t, "default n = 5\nlabel start\nend\n")
	vm := New(prog)
	st := vm.State()
	st.Vars[0] = value.OfInt(99)
	if v, _ := vm.Var("n"); v != value.OfInt(5) {
		t.Errorf("State() must return a copy, got %+v", v)
	}
	if err := vm.Restore(GameState{PC: 0}); err == nil {
		t.Errorf("Restore must reject wrong variable count")
	}
	if err := vm.Restore(GameState{PC: 999, Vars: []value.Value{value.OfInt(1)}}); err == nil {
		t.Errorf("Restore must reject out-of-range pc")
	}
}

func TestEndedStateCanBeRestored(t *testing.T) {
	prog := compile(t, "label start\nend\n")
	vm := New(prog)
	expectEnd(t, vm)
	snap := vm.State()
	if !snap.Ended {
		t.Fatalf("state should be ended")
	}
	vm2 := New(prog)
	if err := vm2.Restore(snap); err != nil {
		t.Fatalf("ended state must be restorable: %v", err)
	}
	expectEnd(t, vm2)
}

func TestShowFlip(t *testing.T) {
	vm := newVM(t, "label start\nshow asep \"happy\" at right flip\nshow asep \"happy\" at left\n")

	ev := next(t, vm)
	want := SpriteState{Character: "asep", Expression: "happy", Position: "right", Flip: true}
	if ev.Kind != EventShow || ev.Sprite != want {
		t.Fatalf("want flipped sprite, got %+v", ev)
	}

	ev = next(t, vm)
	want = SpriteState{Character: "asep", Expression: "happy", Position: "left"}
	if ev.Kind != EventShow || ev.Sprite != want {
		t.Fatalf("want unflipped sprite, got %+v", ev)
	}
	if n := len(vm.State().Sprites); n != 1 {
		t.Errorf("want 1 sprite after re-show, got %d", n)
	}
}
