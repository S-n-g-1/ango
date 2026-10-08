// Command ango runs an Ango script, or a whole story folder of them.
//
//	ango game.ango               play in the terminal
//	ango story/                  play every .ango file under story/ as one
//	                              story (labels/variables shared across
//	                              files; jump/call can cross files freely)
//	ango -auto game.ango          no prompts; always pick the first option (smoke test)
//	ango -check game.ango         compile only and report errors
//	ango -window game.ango        play in a graphical window instead of the terminal
//
// Any of the above also accepts a directory in place of a single file.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"ango/compiler"
	"ango/engine/game"
	"ango/engine/script"
)

// version is set at build time: -ldflags "-X main.version=1.2.3".
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	check := flag.Bool("check", false, "compile only; report errors and exit")
	auto := flag.Bool("auto", false, "no prompts: continue at once and pick the first option")
	window := flag.Bool("window", false, "play in a graphical window instead of the terminal (ignored with -check)")
	assets := flag.String("assets", "", "assets directory for -window (default: <script dir>/assets, or <story dir>/assets for a folder)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: ango [-version] [-check] [-auto] [-window] [-assets dir] script.ango|story-dir")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *showVersion {
		fmt.Printf("ango %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
		return
	}
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	path := flag.Arg(0)
	if *window && !*check {
		os.Exit(playWindow(path, *assets, os.Stderr))
	}
	os.Exit(run(path, *check, *auto, os.Stdin, os.Stdout, os.Stderr))
}

// run returns the process exit code.
func run(path string, check, auto bool, in io.Reader, out, errOut io.Writer) int {
	prog, err := buildPath(path)
	if err != nil {
		report(errOut, err)
		return 1
	}
	if check {
		fmt.Fprintf(out, "%s: OK (%d instructions, %d variables)\n", path, len(prog.Code), len(prog.Vars))
		return 0
	}
	if err := play(game.New(prog), bufio.NewReader(in), out, auto); err != nil {
		report(errOut, err)
		return 1
	}
	return 0
}

// build compiles a single file already read into memory. It's also used
// directly by in-memory tests that never touch disk.
func build(path, src string) (*compiler.Program, error) {
	ast, err := script.Parse(path, src)
	if err != nil {
		return nil, err
	}
	return compiler.Compile(ast)
}

// buildPath builds path, which may be a single .ango file (the original,
// still fully supported, behavior) or a directory of them (see buildDir).
// A story can grow from one file into a folder of chapters without the
// command line invocation changing at all.
func buildPath(path string) (*compiler.Program, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return buildDir(path)
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return build(path, string(src))
}

// buildDir parses every .ango file under dir (recursively, so chapters can
// be organized into subfolders) and compiles them together as one story:
// declarations and labels are shared across files, so `jump`/`call` can
// freely cross from one file into another. There's still exactly one
// entry point, the `start` label, wherever in the tree it lives.
//
// Label and variable names must stay unique across the whole story; a
// duplicate is reported with the file and line of both occurrences, since
// every error already carries its own source file.
func buildDir(dir string) (*compiler.Program, error) {
	var asts []*script.ProgramNode
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".ango" {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		ast, err := script.Parse(path, string(src))
		if err != nil {
			return err
		}
		asts = append(asts, ast)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(asts) == 0 {
		return nil, fmt.Errorf("%s: no .ango files found", dir)
	}
	return compiler.CompileFiles(asts...)
}

// report prints one or more errors with a source snippet and a "^" pointer
// under the offending column, reading each error's own source file from
// disk by e.Span.File — necessary once a build can span several files, so
// two errors in the same report can each show the right file's line.
func report(w io.Writer, err error) {
	var es compiler.Errors
	if errors.As(err, &es) {
		for i, e := range es {
			if i > 0 {
				fmt.Fprintln(w)
			}
			printError(w, e)
		}
		return
	}
	var se *script.Error
	if errors.As(err, &se) {
		printError(w, se)
		return
	}
	fmt.Fprintln(w, err)
}

func printError(w io.Writer, e *script.Error) {
	fmt.Fprintf(w, "%s: %s\n", e.Kind.Label(), e.Msg)
	if e.Span.File != "" {
		fmt.Fprintf(w, "  --> %s:%d:%d\n", e.Span.File, e.Span.Start.Line, e.Span.Start.Column)
	}
	if line, ok := sourceLineFromFile(e.Span.File, e.Span.Start.Line); ok {
		col := e.Span.Start.Column
		if col < 1 {
			col = 1
		}
		fmt.Fprintf(w, "\n  %s\n", line)
		fmt.Fprintf(w, "  %s^\n", strings.Repeat(" ", col-1))
	}
	if e.Hint != "" {
		fmt.Fprintf(w, "\n  hint: %s\n", e.Hint)
	}
}

// sourceLineFromFile returns the 1-indexed line from the named file.
// Reading the file again here (rather than threading source text through
// every caller) is what lets each error in a multi-file build show its
// own file's line, not whichever file happened to be passed in first.
func sourceLineFromFile(path string, line int) (string, bool) {
	if line < 1 || path == "" {
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	lines := strings.Split(string(data), "\n")
	if line > len(lines) {
		return "", false
	}
	return lines[line-1], true
}

// play drives the VM until the script ends, input runs out, or an error occurs.
func play(vm *game.VM, in *bufio.Reader, out io.Writer, auto bool) error {
	for {
		ev, err := vm.Next()
		if err != nil {
			return err
		}
		switch ev.Kind {
		case game.EventSay:
			if ev.Speaker != "" {
				fmt.Fprintf(out, "%s: %s\n", ev.Speaker, ev.Text)
			} else {
				fmt.Fprintln(out, ev.Text)
			}
			if !auto && !waitEnter(in) {
				return nil
			}

		case game.EventScene:
			fmt.Fprintf(out, "[scene: %s]\n", ev.Asset)

		case game.EventShow:
			fmt.Fprintf(out, "[show: %s %s %s]\n", ev.Sprite.Character, ev.Sprite.Expression, ev.Sprite.Position)

		case game.EventHide:
			fmt.Fprintf(out, "[hide: %s]\n", ev.Target)

		case game.EventChoice:
			for i, o := range ev.Options {
				fmt.Fprintf(out, "  %d) %s\n", i+1, o)
			}
			idx := 0
			if auto {
				fmt.Fprintln(out, "  > 1")
			} else {
				var ok bool
				if idx, ok = askChoice(in, out, len(ev.Options)); !ok {
					return nil
				}
			}
			if err := vm.Choose(idx); err != nil {
				return err
			}

		case game.EventEnd:
			fmt.Fprintln(out, "[end]")
			return nil
		}
	}
}

// waitEnter blocks for Enter. It returns false when input is exhausted.
func waitEnter(in *bufio.Reader) bool {
	_, err := in.ReadString('\n')
	return err == nil
}

// askChoice re-prompts until it reads a number in 1..n.
// It returns ok=false when input is exhausted.
func askChoice(in *bufio.Reader, out io.Writer, n int) (int, bool) {
	for {
		fmt.Fprintf(out, "Choose 1-%d: ", n)
		line, err := in.ReadString('\n')
		if k, convErr := strconv.Atoi(strings.TrimSpace(line)); convErr == nil && k >= 1 && k <= n {
			return k - 1, true
		}
		if err != nil {
			fmt.Fprintln(out)
			return 0, false
		}
	}
}
