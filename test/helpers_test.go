// Package test holds Ango's cross-package and cross-platform tests.
//
// Unit tests live next to the code they cover (compiler/, engine/...,
// cmd/...). This package covers what no single package can: the whole
// pipeline (source -> parser -> compiler -> VM), every shipped example,
// and properties that must hold on Linux and Windows alike (line endings,
// path separators, case-sensitive asset names).
package test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ango/compiler"
	"ango/engine/game"
	"ango/engine/script"
)

// compileSource compiles one in-memory script.
func compileSource(t testing.TB, name, src string) (*compiler.Program, error) {
	t.Helper()
	ast, err := script.Parse(name, src)
	if err != nil {
		return nil, err
	}
	return compiler.Compile(ast)
}

// mustCompile fails the test if src does not compile.
func mustCompile(t testing.TB, src string) *compiler.Program {
	t.Helper()
	prog, err := compileSource(t, "test.ango", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	return prog
}

// compileDir compiles every .ango file under dir as one story, the same way
// `ango <folder>` does.
func compileDir(t testing.TB, dir string) (*compiler.Program, error) {
	t.Helper()
	var asts []*script.ProgramNode
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(p) != ".ango" {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		ast, err := script.Parse(p, string(src))
		if err != nil {
			return err
		}
		asts = append(asts, ast)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return compiler.CompileFiles(asts...)
}

// exampleDirs lists the story folders under ../examples.
func exampleDirs(t testing.TB) []string {
	t.Helper()
	root := filepath.Join("..", "examples")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(root, e.Name()))
		}
	}
	if len(dirs) == 0 {
		t.Fatal("no examples found")
	}
	return dirs
}

// trace is what a headless playthrough produced.
type trace struct {
	Events []game.Event
	Ended  bool
}

// says returns "Speaker: text" for every say event, in order.
func (tr trace) says() []string {
	var out []string
	for _, ev := range tr.Events {
		if ev.Kind == game.EventSay {
			if ev.Speaker == "" {
				out = append(out, ev.Text)
			} else {
				out = append(out, ev.Speaker+": "+ev.Text)
			}
		}
	}
	return out
}

func (tr trace) has(sub string) bool {
	for _, s := range tr.says() {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// play drives vm to the end. Each choice consumes the next entry of picks
// (an index into the enabled options); once picks runs out it takes the
// first enabled option. It fails the test on a runtime error or if the
// story does not finish within a generous step budget.
func play(t testing.TB, vm *game.VM, picks ...int) trace {
	t.Helper()
	var tr trace
	for step := 0; step < 100000; step++ {
		ev, err := vm.Next()
		if err != nil {
			t.Fatalf("runtime error after %d events: %v", len(tr.Events), err)
		}
		tr.Events = append(tr.Events, ev)
		switch ev.Kind {
		case game.EventEnd:
			tr.Ended = true
			return tr
		case game.EventChoice:
			pick := 0
			if len(picks) > 0 {
				pick, picks = picks[0], picks[1:]
			}
			if err := vm.Choose(pick); err != nil {
				t.Fatalf("choose %d: %v", pick, err)
			}
		}
	}
	t.Fatal("story did not finish within 100000 events (infinite loop?)")
	return tr
}
