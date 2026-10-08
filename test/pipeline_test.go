package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ango/compiler"
	"ango/engine/game"
	"ango/engine/value"
)

func readFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Scripts written on Windows use CRLF; Linux editors use LF; old Mac tools
// use CR. All three must compile to the exact same program.
func TestLineEndingsCompileIdentically(t *testing.T) {
	lf := strings.ReplaceAll(readFixture(t, "crlf_lf.ango"), "\r\n", "\n")
	variants := map[string]string{
		"LF":   lf,
		"CRLF": strings.ReplaceAll(lf, "\n", "\r\n"),
		"CR":   strings.ReplaceAll(lf, "\n", "\r"),
	}
	want := mustCompile(t, lf).Disassemble()
	for name, src := range variants {
		if got := mustCompile(t, src).Disassemble(); got != want {
			t.Errorf("%s produced a different program:\n%s\nwant:\n%s", name, got, want)
		}
	}
}

// Editors on Windows (notably Notepad) often save a UTF-8 BOM.
func TestBOMIsIgnored(t *testing.T) {
	src := readFixture(t, "crlf_lf.ango")
	want := mustCompile(t, src).Disassemble()
	if got := mustCompile(t, "\ufeff"+src).Disassemble(); got != want {
		t.Errorf("BOM changed the program:\n%s", got)
	}
}

func TestMissingFinalNewline(t *testing.T) {
	src := strings.TrimRight(readFixture(t, "crlf_lf.ango"), "\r\n")
	mustCompile(t, src)
}

func TestFixtureErrors(t *testing.T) {
	cases := []struct {
		file string
		want string
	}{
		{"bad_label.ango", "undefined label"},
		{"bad_indent.ango", "indent"},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			_, err := compileSource(t, c.file, readFixture(t, c.file))
			if err == nil {
				t.Fatal("expected a compile error")
			}
			if !strings.Contains(strings.ToLower(err.Error()), c.want) {
				t.Errorf("error %q should mention %q", err, c.want)
			}
			if !strings.Contains(err.Error(), c.file) {
				t.Errorf("error %q should name the file %s", err, c.file)
			}
		})
	}
}

func TestChoiceBranchesAreIndependent(t *testing.T) {
	prog := mustCompile(t, readFixture(t, "crlf_lf.ango"))
	a := play(t, game.New(prog), 0)
	b := play(t, game.New(prog), 1)
	if !a.has("pilih A") || a.has("pilih B") {
		t.Errorf("pick 0 should only reach A: %v", a.says())
	}
	if !b.has("pilih B") || b.has("pilih A") {
		t.Errorf("pick 1 should only reach B: %v", b.says())
	}
}

func TestSaveRestoreRoundTrip(t *testing.T) {
	prog := mustCompile(t, readFixture(t, "crlf_lf.ango"))
	vm := game.New(prog)

	for {
		ev, err := vm.Next()
		if err != nil {
			t.Fatal(err)
		}
		if ev.Kind == game.EventChoice {
			break
		}
	}
	saved := vm.State()

	if err := vm.Choose(0); err != nil {
		t.Fatal(err)
	}
	first := play(t, vm)

	fresh := game.New(prog)
	if err := fresh.Restore(saved); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if err := fresh.Choose(1); err != nil {
		t.Fatal(err)
	}
	second := play(t, fresh)

	if !first.has("pilih A") || !second.has("pilih B") {
		t.Errorf("restoring a saved state should allow a different branch: %v / %v", first.says(), second.says())
	}
	if v, ok := fresh.Var("score"); !ok || v != value.OfInt(1) {
		t.Errorf("score = %v, %v; want 1", v, ok)
	}
}

func TestCompileErrorsAreStructured(t *testing.T) {
	_, err := compileSource(t, "x.ango", readFixture(t, "bad_label.ango"))
	if _, ok := err.(compiler.Errors); !ok {
		t.Fatalf("want compiler.Errors, got %T: %v", err, err)
	}
}
