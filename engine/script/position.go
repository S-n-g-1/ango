package script

import "fmt"

// Position is a 1-based line and column. Column counts runes, not bytes.
type Position struct {
	Line   int
	Column int
}

// Span is a half-open range [Start, End) in a source file.
// A zero-width span (Start == End) marks synthetic tokens such as DEDENT.
type Span struct {
	File  string
	Start Position
	End   Position
}

// Kind identifies which stage produced an Error. It is purely
// informational: formatting and matching by message still work
// when Kind is left empty (the zero value), so existing call sites
// that build Error{Span: ..., Msg: ...} keep compiling unchanged.
type Kind string

const (
	KindLexer    Kind = "lexer"
	KindParser   Kind = "parser"
	KindCompiler Kind = "compiler"
	KindRuntime  Kind = "runtime"
)

// Label returns the text a CLI prints before the message, e.g. "Parser error".
func (k Kind) Label() string {
	switch k {
	case KindLexer:
		return "Lexer error"
	case KindParser:
		return "Parser error"
	case KindCompiler:
		return "Compiler error"
	case KindRuntime:
		return "Runtime error"
	default:
		return "Error"
	}
}

// Error is a positioned error shared by the lexer, parser, compiler and VM.
// Kind and Hint are optional metadata for nicer CLI output; Error() ignores
// them so existing "file:line:col: message" output is unchanged.
type Error struct {
	Span Span
	Msg  string
	Kind Kind
	Hint string // optional one-line suggestion, e.g. "available: normal, senang, marah"
}

func (e *Error) Error() string {
	if e.Span.File == "" {
		return fmt.Sprintf("%d:%d: %s", e.Span.Start.Line, e.Span.Start.Column, e.Msg)
	}
	return fmt.Sprintf("%s:%d:%d: %s", e.Span.File, e.Span.Start.Line, e.Span.Start.Column, e.Msg)
}

// newError is a package-internal constructor the lexer and parser can use
// instead of literal &Error{...}, so Kind is set consistently.
// (Kept unexported: outside packages should keep using &Error{...} literals
// via the constructors below, since script.newError is not visible to them.)
func newError(kind Kind, span Span, format string, args ...any) *Error {
	return &Error{Span: span, Kind: kind, Msg: fmt.Sprintf(format, args...)}
}

// NewLexerError, NewParserError, NewCompilerError and NewRuntimeError are the
// constructors for use outside package script (compiler, engine/game).
// Using them (instead of &script.Error{...} literals) makes sure Kind is
// always set, which the CLI formatter needs to pick the right label.
func NewLexerError(span Span, format string, args ...any) *Error {
	return newError(KindLexer, span, format, args...)
}

func NewParserError(span Span, format string, args ...any) *Error {
	return newError(KindParser, span, format, args...)
}

func NewCompilerError(span Span, format string, args ...any) *Error {
	return newError(KindCompiler, span, format, args...)
}

func NewRuntimeError(span Span, format string, args ...any) *Error {
	return newError(KindRuntime, span, format, args...)
}

// WithHint returns a copy of e with Hint set, for chaining at the call site:
//
//	return nil, script.NewCompilerError(span, "unknown expression %q", name).
//		WithHint("available: " + strings.Join(known, ", "))
func (e *Error) WithHint(hint string) *Error {
	c := *e
	c.Hint = hint
	return &c
}
