package script

import "fmt"

// TokenType identifies the kind of a token.
type TokenType uint8

const (
	TokenEOF TokenType = iota

	TokenIdentifier
	TokenString
	TokenInt
	TokenFloat

	// Layout tokens. The lexer guarantees that every logical line ends with
	// TokenNewline (even at EOF without a trailing "\n"), and that TokenIndent /
	// TokenDedent are balanced by the time TokenEOF is reached.
	TokenNewline
	TokenIndent
	TokenDedent

	TokenColon
	TokenNamespace
	TokenFlip
	TokenDot
	TokenComma
	TokenLParen
	TokenRParen
	TokenAtSign // "@", as in @id("...")

	TokenEqual // "="
	TokenPlus
	TokenMinus
	TokenStar
	TokenSlash
	TokenEqualEqual
	TokenNotEqual
	TokenLess
	TokenLessEqual
	TokenGreater
	TokenGreaterEqual

	// Keywords (lowercase, case-sensitive).
	TokenAnd
	TokenOr
	TokenNot
	TokenLabel
	TokenScene
	TokenShow
	TokenHide
	TokenChoice
	TokenJump
	TokenCall
	TokenReturn
	TokenSet
	TokenIf
	TokenElse
	TokenDefault
	TokenEnd
	TokenAt   // "at", as in: show asep "normal" at left
	TokenWith // "with", as in: scene "lorong" with fade 0.5
	TokenTrue
	TokenFalse
)

var keywords = map[string]TokenType{
	"and":       TokenAnd,
	"or":        TokenOr,
	"not":       TokenNot,
	"label":     TokenLabel,
	"scene":     TokenScene,
	"show":      TokenShow,
	"hide":      TokenHide,
	"choice":    TokenChoice,
	"jump":      TokenJump,
	"call":      TokenCall,
	"return":    TokenReturn,
	"set":       TokenSet,
	"if":        TokenIf,
	"else":      TokenElse,
	"default":   TokenDefault,
	"end":       TokenEnd,
	"at":        TokenAt,
	"with":      TokenWith,
	"true":      TokenTrue,
	"false":     TokenFalse,
	"namespace": TokenNamespace,
	"flip":      TokenFlip,
}

var tokenNames = map[TokenType]string{
	TokenEOF:          "EOF",
	TokenIdentifier:   "IDENT",
	TokenString:       "STRING",
	TokenInt:          "INT",
	TokenFloat:        "FLOAT",
	TokenNewline:      "NEWLINE",
	TokenIndent:       "INDENT",
	TokenDedent:       "DEDENT",
	TokenWith:         "with",
	TokenDot:          ".",
	TokenColon:        ":",
	TokenComma:        ",",
	TokenLParen:       "(",
	TokenRParen:       ")",
	TokenAtSign:       "@",
	TokenEqual:        "=",
	TokenPlus:         "+",
	TokenMinus:        "-",
	TokenStar:         "*",
	TokenSlash:        "/",
	TokenEqualEqual:   "==",
	TokenNotEqual:     "!=",
	TokenLess:         "<",
	TokenLessEqual:    "<=",
	TokenGreater:      ">",
	TokenGreaterEqual: ">=",
}

func init() {
	for word, tt := range keywords {
		tokenNames[tt] = word
	}
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return fmt.Sprintf("TokenType(%d)", uint8(t))
}

// StringPartKind distinguishes literal text from [variable] interpolation.
type StringPartKind uint8

const (
	PartText StringPartKind = iota
	PartVariable
)

// StringPart is one piece of a string literal after escapes and
// interpolation have been resolved by the lexer.
//
// For PartText, Text is the decoded literal text ("[[" -> "[", "]]" -> "]",
// backslash escapes applied). For PartVariable, Text is the variable name.
// Span covers the raw source of the part (for variables, including brackets).
type StringPart struct {
	Kind StringPartKind
	Text string
	Span Span
}

// Token is a lexical token.
//
// Lexeme is the source text exactly as written (for strings, including the
// quotes and undecoded escapes). It is "\n" for a real newline and empty for
// synthetic tokens (INDENT carries the leading spaces; DEDENT, EOF and the
// implicit NEWLINE at end of file are empty).
//
// Parts is set only for TokenString. An empty string literal has no parts.
type Token struct {
	Type   TokenType
	Lexeme string
	Span   Span
	Parts  []StringPart
}
