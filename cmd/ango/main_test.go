package main

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ango/compiler"
	"ango/engine/game"
)

const choiceScript = `
label start
Asep: "Halo"
choice:
    "A":
        Asep: "chose A"
    "B":
        Asep: "chose B"
`

func playSrc(t *testing.T, src, input string, auto bool) string {
	t.Helper()
	prog, err := build("test.ango", src)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	var out bytes.Buffer
	if err := play(game.New(prog), bufio.NewReader(strings.NewReader(input)), &out, auto); err != nil {
		t.Fatalf("play: %v", err)
	}
	return out.String()
}

func mustContain(t *testing.T, out string, parts ...string) {
	t.Helper()
	for _, p := range parts {
		if !strings.Contains(out, p) {
			t.Errorf("output missing %q:\n%s", p, out)
		}
	}
}

func TestAutoModePicksFirstOption(t *testing.T) {
	out := playSrc(t, choiceScript, "", true)
	mustContain(t, out, "Asep: Halo", "1) A", "2) B", "chose A", "[end]")
	if strings.Contains(out, "chose B") {
		t.Errorf("auto mode must pick the first option:\n%s", out)
	}
}

func TestInteractiveChoice(t *testing.T) {
	out := playSrc(t, choiceScript, "\n2\n\n", false)
	mustContain(t, out, "Asep: chose B", "[end]")
}

func TestBadChoiceInputReprompts(t *testing.T) {
	out := playSrc(t, choiceScript, "\nabc\n9\n1\n\n", false)
	if n := strings.Count(out, "Choose 1-2:"); n != 3 {
		t.Errorf("want 3 prompts, got %d:\n%s", n, out)
	}
	mustContain(t, out, "chose A")
}

func TestEOFQuitsCleanly(t *testing.T) {
	out := playSrc(t, choiceScript, "", false)
	mustContain(t, out, "Asep: Halo")
	if strings.Contains(out, "chose") {
		t.Errorf("must stop when input ends:\n%s", out)
	}
}

// TestExamplesRun treats every immediate subfolder of examples/ as one
// story (buildDir handles both a single .ango file and many of them the
// same way) and plays it in -auto mode to the end. This replaces any test
// that pinned a story's exact dialogue or asset names: add, rename or
// rewrite any example and this test needs no changes, as long as the
// story itself still compiles and actually finishes.
//
// What this does NOT catch: branches the auto-picker never takes (it
// always chooses the first available option), so a bug reachable only
// through a later choice can slip through. For that, add a small, focused
// test for that specific script instead.
func TestExamplesRun(t *testing.T) {
	root := "../../examples"
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		t.Run(e.Name(), func(t *testing.T) {
			prog, err := buildDir(dir)
			if err != nil {
				if es, ok := err.(compiler.Errors); ok {
					t.Fatalf("compile failed:\n%s", es.Details())
				}
				t.Fatalf("compile failed: %v", err)
			}
			var out bytes.Buffer
			if err := play(game.New(prog), bufio.NewReader(strings.NewReader("")), &out, true); err != nil {
				t.Fatalf("runtime error: %v\noutput so far:\n%s", err, out.String())
			}
			if !strings.Contains(out.String(), "[end]") {
				t.Errorf("story never reached [end]:\n%s", out.String())
			}
		})
	}
}

func TestRunCheck(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.ango")
	bad := filepath.Join(dir, "bad.ango")
	os.WriteFile(good, []byte("label start\nend\n"), 0o644)
	os.WriteFile(bad, []byte("label start\njump nowhere\n"), 0o644)

	var out, errOut bytes.Buffer
	if code := run(good, true, false, strings.NewReader(""), &out, &errOut); code != 0 || !strings.Contains(out.String(), "OK") {
		t.Errorf("good script: code=%d out=%q err=%q", code, out.String(), errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if code := run(bad, true, false, strings.NewReader(""), &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "undefined label") {
		t.Errorf("bad script: code=%d err=%q", code, errOut.String())
	}

	errOut.Reset()
	if code := run(filepath.Join(dir, "missing.ango"), true, false, strings.NewReader(""), &out, &errOut); code != 1 {
		t.Errorf("missing file: want code 1, got %d", code)
	}
}

// TestRunCheckDirectory exercises the new multi-file path directly: a
// folder with two chapters, where the second's label is only reachable by
// jumping to it from the first, must compile as a single story.
func TestRunCheckDirectory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "ch1.ango"), []byte("label start\njump ch2\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "ch2.ango"), []byte("label ch2\nend\n"), 0o644)

	var out, errOut bytes.Buffer
	if code := run(dir, true, false, strings.NewReader(""), &out, &errOut); code != 0 || !strings.Contains(out.String(), "OK") {
		t.Errorf("story dir: code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
}

// TestDuplicateLabelAcrossFilesIsReported makes sure a label defined
// twice in two different files of the same story is caught, and that the
// error correctly points at the second file (not the first), proving
// Span.File survives the multi-file merge.
func TestDuplicateLabelAcrossFilesIsReported(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.ango"), []byte("label start\nend\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "b.ango"), []byte("label start\nend\n"), 0o644)

	_, err := buildDir(dir)
	if err == nil {
		t.Fatal("want a duplicate-label error, got none")
	}
	es, ok := err.(compiler.Errors)
	if !ok || len(es) == 0 {
		t.Fatalf("want compiler.Errors, got %v", err)
	}
	if !strings.Contains(es[0].Error(), "already defined") {
		t.Errorf("unexpected error: %v", es[0])
	}
	if !strings.Contains(es[0].Span.File, "b.ango") {
		t.Errorf("want the error to point at b.ango (the second occurrence), got %q", es[0].Span.File)
	}
}
