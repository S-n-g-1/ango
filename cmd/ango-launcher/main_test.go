package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func write(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("label start\nend\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func names(ps []project) []string {
	var out []string
	for _, p := range ps {
		out = append(out, p.name)
	}
	return out
}

func TestScanRootsSubfolders(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "zeta", "main.ango"))
	write(t, filepath.Join(root, "Alpha", "main.ango"))
	write(t, filepath.Join(root, "Alpha", "chapter1.ango"))
	write(t, filepath.Join(root, ".hidden", "main.ango"))
	write(t, filepath.Join(root, "kosong", "readme.txt"))
	write(t, filepath.Join(root, "loose-not-in-root-listing", "x.txt"))

	ps, errs := scanRoots([]string{root})
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	if want := []string{"Alpha", "zeta"}; !reflect.DeepEqual(names(ps), want) {
		t.Fatalf("got %v, want %v", names(ps), want)
	}
	if ps[0].files != 2 || !filepath.IsAbs(ps[0].dir) || ps[0].root == "" {
		t.Errorf("bad project metadata: %+v", ps[0])
	}
}

func TestScanRootsRootIsAProject(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "main.ango"))
	write(t, filepath.Join(root, "sub", "main.ango"))

	ps, _ := scanRoots([]string{root})
	if len(ps) != 2 {
		t.Fatalf("want the root and its sub-folder, got %v", names(ps))
	}
}

func TestScanRootsDedupAndMissing(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "game", "main.ango"))

	ps, errs := scanRoots([]string{root, root, filepath.Join(root, "tidak-ada")})
	if len(ps) != 1 {
		t.Errorf("duplicate roots must list a project once, got %v", names(ps))
	}
	if len(errs) != 1 {
		t.Errorf("want 1 error for the missing root, got %v", errs)
	}
}

func TestScanRootsSymlinkedFolder(t *testing.T) {
	root, elsewhere := t.TempDir(), t.TempDir()
	write(t, filepath.Join(elsewhere, "main.ango"))
	if err := os.Symlink(elsewhere, filepath.Join(root, "linked")); err != nil {
		t.Skip("symlinks not supported here")
	}
	ps, _ := scanRoots([]string{root})
	if len(ps) != 1 || ps[0].name != "linked" {
		t.Errorf("symlinked project folder should be listed, got %v", names(ps))
	}
}

func TestResolveRootsPrecedence(t *testing.T) {
	cfg := config{Roots: []string{"/cfg"}}
	cases := []struct {
		name       string
		args       []string
		env        string
		cfg        config
		want       []string
		wantSource string
	}{
		{"args win", []string{"/a"}, "/env", cfg, []string{"/a"}, "argumen"},
		{"env next", nil, "/e1:/e2", cfg, []string{"/e1", "/e2"}, "ANGO_PROJECTS"},
		{"config next", nil, "", cfg, []string{"/cfg"}, "konfigurasi"},
		{"fallback", nil, "", config{}, []string{"/fb"}, "bawaan"},
		{"blank entries ignored", []string{" "}, "", config{}, []string{"/fb"}, "bawaan"},
	}
	for _, c := range cases {
		got, src := resolveRoots(c.args, c.env, c.cfg, "/fb")
		if !reflect.DeepEqual(got, c.want) || src != c.wantSource {
			t.Errorf("%s: got %v (%s), want %v (%s)", c.name, got, src, c.want, c.wantSource)
		}
	}
}

func TestExpandHome(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	if got := expandHome("~/games"); got != "/home/tester/games" {
		t.Errorf("got %q", got)
	}
	if got := expandHome("/abs/path"); got != "/abs/path" {
		t.Errorf("got %q", got)
	}
}

func TestConfigRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	c, err := loadConfig()
	if err != nil || len(c.Roots) != 0 {
		t.Fatalf("missing config should be empty, got %+v, %v", c, err)
	}
	want := config{Roots: []string{"/a", "/b"}, Ango: "/bin/ango"}
	if err := saveConfig(want); err != nil {
		t.Fatal(err)
	}
	got, err := loadConfig()
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, %v; want %+v", got, err, want)
	}
}

func TestFindAngoExplicit(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "ango")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := findAngo(bin); err != nil || got != bin {
		t.Errorf("findAngo(%q) = %q, %v", bin, got, err)
	}
	if _, err := findAngo(filepath.Join(dir, "nope")); err == nil {
		t.Error("expected an error for a missing explicit path")
	}
}
