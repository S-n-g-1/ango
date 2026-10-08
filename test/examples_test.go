package test

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"ango/engine/game"
)

// Every folder under examples/ is a story: it must compile and reach [end]
// when the first enabled option is always chosen.
func TestExamplesPlayToTheEnd(t *testing.T) {
	for _, dir := range exampleDirs(t) {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			prog, err := compileDir(t, dir)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			if tr := play(t, game.New(prog)); !tr.Ended {
				t.Error("story never ended")
			}
		})
	}
}

// Always picking the *last* option exercises the branches the "first
// option" walk above never reaches.
func TestExamplesPlayLastBranch(t *testing.T) {
	for _, dir := range exampleDirs(t) {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			prog, err := compileDir(t, dir)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			vm := game.New(prog)
			for i := 0; i < 100000; i++ {
				ev, err := vm.Next()
				if err != nil {
					t.Fatalf("runtime: %v", err)
				}
				if ev.Kind == game.EventEnd {
					return
				}
				if ev.Kind == game.EventChoice {
					if err := vm.Choose(len(ev.Options) - 1); err != nil {
						t.Fatalf("choose: %v", err)
					}
				}
			}
			t.Fatal("story did not finish")
		})
	}
}

// Compiling the same folder twice must give the same program, regardless
// of directory-walk order or path separators in file names.
func TestExamplesCompileDeterministically(t *testing.T) {
	for _, dir := range exampleDirs(t) {
		a, err := compileDir(t, dir)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := compileDir(t, dir)
		if a.Disassemble() != b.Disassemble() {
			t.Errorf("%s: non-deterministic output", dir)
		}
	}
}

// exts mirrors the backend's default extension search order.
var exts = []string{"", ".webp", ".png", ".jpg", ".jpeg", ".svg"}

// existsExact reports whether rel (slash-separated) exists under root with
// exactly that capitalization. Windows and macOS file systems ignore case,
// Linux does not, so a story that only works because "calm.png" matched
// "Calm.png" would break the moment it is played on Linux.
func existsExact(root, rel string) (exact, anyCase bool) {
	cur := root
	exact, anyCase = true, true
	for _, part := range strings.Split(rel, "/") {
		entries, err := os.ReadDir(cur)
		if err != nil {
			return false, false
		}
		next := ""
		found := false
		for _, e := range entries {
			if e.Name() == part {
				next, found = e.Name(), true
				break
			}
		}
		if !found {
			exact = false
			for _, e := range entries {
				if strings.EqualFold(e.Name(), part) {
					next, found = e.Name(), true
					break
				}
			}
			if !found {
				return false, false
			}
		}
		cur = filepath.Join(cur, next)
	}
	return exact, anyCase
}

// resolve looks for any candidate base (with each extension) under root.
func resolve(root string, bases ...string) (exact, anyCase bool) {
	for _, b := range bases {
		for _, ext := range exts {
			e, a := existsExact(root, b+ext)
			if e {
				return true, true
			}
			anyCase = anyCase || a
		}
	}
	return false, anyCase
}

// Asset names in scripts must match file names exactly (case included).
// Missing assets are allowed (the backend draws a placeholder); assets that
// only match case-insensitively are not, because they fail on Linux.
func TestAssetNamesMatchCaseExactly(t *testing.T) {
	for _, dir := range exampleDirs(t) {
		root := filepath.Join(dir, "assets")
		if _, err := os.Stat(root); err != nil {
			continue
		}
		prog, err := compileDir(t, dir)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(filepath.Base(dir), func(t *testing.T) {
			for _, sc := range prog.Scenes {
				exact, anyCase := resolve(root, path.Join("backgrounds", sc.Asset))
				if !exact && anyCase {
					t.Errorf("scene %q only matches an asset with different capitalization", sc.Asset)
				} else if !exact {
					t.Logf("scene %q has no asset (placeholder will be used)", sc.Asset)
				}
			}
			for _, sp := range prog.Sprites {
				exact, anyCase := resolve(root,
					path.Join("characters", sp.Character+"_"+sp.Expression),
					path.Join("characters", sp.Character, sp.Expression),
					path.Join("characters", sp.Character))
				if !exact && anyCase {
					t.Errorf("sprite %s %q only matches an asset with different capitalization", sp.Character, sp.Expression)
				} else if !exact {
					t.Logf("sprite %s %q has no asset (placeholder will be used)", sp.Character, sp.Expression)
				}
			}
		})
	}
}

// Asset and script files must be addressable with forward slashes only:
// the backend reads them through fs.FS, where "\" is not a separator.
func TestNoBackslashesInAssetNames(t *testing.T) {
	for _, dir := range exampleDirs(t) {
		prog, err := compileDir(t, dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, sc := range prog.Scenes {
			if strings.Contains(sc.Asset, `\`) {
				t.Errorf("%s: scene asset %q contains a backslash", dir, sc.Asset)
			}
		}
	}
}
