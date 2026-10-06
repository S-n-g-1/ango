package script

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// render turns tokens into a compact, readable form for table tests:
//
//	IDENT(Asep) : STRING["Halo, " {player_name} "."] NEWLINE EOF
func render(tokens []Token) string {
	out := make([]string, len(tokens))
	for i, t := range tokens {
		switch t.Type {
		case TokenIdentifier, TokenInt, TokenFloat:
			out[i] = fmt.Sprintf("%s(%s)", t.Type, t.Lexeme)
		case TokenString:
			var parts []string
			for _, p := range t.Parts {
				if p.Kind == PartVariable {
					parts = append(parts, "{"+p.Text+"}")
				} else {
					parts = append(parts, strconv.Quote(p.Text))
				}
			}
			out[i] = "STRING[" + strings.Join(parts, " ") + "]"
		default:
			out[i] = t.Type.String()
		}
	}
	return strings.Join(out, " ")
}

func mustLex(t *testing.T, src string) []Token {
	t.Helper()
	tokens, err := Lex("test.ango", src)
	if err != nil {
		t.Fatalf("Lex(%q) unexpected error: %v", src, err)
	}
	return tokens
}

func TestLexTokens(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"empty", ``, `EOF`},
		{"blank and comment lines only", "\n  \n# hi\n    # deeper\n", `EOF`},
		{"whitespace-only last line", "\"x\"\n    ", `STRING["x"] NEWLINE EOF`},

		{"dialogue", `Asep: "Halo."`, `IDENT(Asep) : STRING["Halo."] NEWLINE EOF`},
		{"dialogue with trailing newline", "Asep: \"Halo.\"\n", `IDENT(Asep) : STRING["Halo."] NEWLINE EOF`},
		{"narration", `"Suasana sunyi."`, `STRING["Suasana sunyi."] NEWLINE EOF`},
		{"string speaker", `"Bu Ratna": "Selamat pagi."`, `STRING["Bu Ratna"] : STRING["Selamat pagi."] NEWLINE EOF`},
		{"empty string", `""`, `STRING[] NEWLINE EOF`},
		{"hash inside string is not a comment", `"a # b"`, `STRING["a # b"] NEWLINE EOF`},

		{"interpolation", `Asep: "Halo, [player_name]."`,
			`IDENT(Asep) : STRING["Halo, " {player_name} "."] NEWLINE EOF`},
		{"adjacent interpolations", `"[a][b]"`, `STRING[{a} {b}] NEWLINE EOF`},
		{"bracket escapes", `"Gunakan [[player_name]] untuk"`, `STRING["Gunakan [player_name] untuk"] NEWLINE EOF`},
		{"escape next to interpolation", `"[[[name]]]"`, `STRING["[" {name} "]"] NEWLINE EOF`},

		{"numbers", `set t = 1.5 + 2`, `set IDENT(t) = FLOAT(1.5) + INT(2) NEWLINE EOF`},
		{"arithmetic", `x = (1 + 2) * 3 / 4 - 5`,
			`IDENT(x) = ( INT(1) + INT(2) ) * INT(3) / INT(4) - INT(5) NEWLINE EOF`},
		{"comparison operators", `a == b != c <= d >= e < f > g`,
			`IDENT(a) == IDENT(b) != IDENT(c) <= IDENT(d) >= IDENT(e) < IDENT(f) > IDENT(g) NEWLINE EOF`},
		{"comma and parens", `f(a, b)`, `IDENT(f) ( IDENT(a) , IDENT(b) ) NEWLINE EOF`},
		{"trailing comment", `set x = 1 # note`, `set IDENT(x) = INT(1) NEWLINE EOF`},
		{"trailing spaces", `jump a   `, `jump IDENT(a) NEWLINE EOF`},

		{"if expression", `if not found_key and courage > 5 or true:`,
			`if not IDENT(found_key) and IDENT(courage) > INT(5) or true : NEWLINE EOF`},
		{"else", `else:`, `else : NEWLINE EOF`},
		{"default", `default found_key = false`, `default IDENT(found_key) = false NEWLINE EOF`},
		{"label", `label start`, `label IDENT(start) NEWLINE EOF`},
		{"scene", `scene "school.png"`, `scene STRING["school.png"] NEWLINE EOF`},
		{"show sprite", `show asep "normal" at center`, `show IDENT(asep) STRING["normal"] at IDENT(center) NEWLINE EOF`},
		{"hide", `hide asep`, `hide IDENT(asep) NEWLINE EOF`},
		{"call return end", "call x\nreturn\nend\n", `call IDENT(x) NEWLINE return NEWLINE end NEWLINE EOF`},
		{"stable id", `@id("intro_01")`, `@ IDENT(id) ( STRING["intro_01"] ) NEWLINE EOF`},
		{"conditional choice option", `"Ambil kunci" if not found_key:`,
			`STRING["Ambil kunci"] if not IDENT(found_key) : NEWLINE EOF`},
		{"keywords are case-sensitive", `If Label`, `IDENT(If) IDENT(Label) NEWLINE EOF`},
		{"unicode identifiers", `_x1 naïve`, `IDENT(_x1) IDENT(naïve) NEWLINE EOF`},

		{"choice indentation", `choice:
    "A":
        jump a
`, `choice : NEWLINE INDENT STRING["A"] : NEWLINE INDENT jump IDENT(a) NEWLINE DEDENT DEDENT EOF`},

		{"dedent several levels then continue", `choice:
    "A":
        jump a
Asep: "x"
`, `choice : NEWLINE INDENT STRING["A"] : NEWLINE INDENT jump IDENT(a) NEWLINE DEDENT DEDENT IDENT(Asep) : STRING["x"] NEWLINE EOF`},

		{"dedent to nonzero level", `a:
    b:
        c
    d
`, `IDENT(a) : NEWLINE INDENT IDENT(b) : NEWLINE INDENT IDENT(c) NEWLINE DEDENT IDENT(d) NEWLINE DEDENT EOF`},

		{"no trailing newline inside block", "if x:\n    jump a",
			`if IDENT(x) : NEWLINE INDENT jump IDENT(a) NEWLINE DEDENT EOF`},

		{"blank and comment lines do not affect indentation", `choice:
    "A":

        # comment at deeper level
    # comment at shallower level
        jump a
`, `choice : NEWLINE INDENT STRING["A"] : NEWLINE INDENT jump IDENT(a) NEWLINE DEDENT DEDENT EOF`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := render(mustLex(t, tt.source))
			if got != tt.want {
				t.Errorf("source %q\n got: %s\nwant: %s", tt.source, got, tt.want)
			}
		})
	}
}

func TestLexStringEscapes(t *testing.T) {
	tokens := mustLex(t, `"a \"b\" \\ \n c"`)
	if len(tokens) != 3 || tokens[0].Type != TokenString {
		t.Fatalf("unexpected tokens: %s", render(tokens))
	}
	parts := tokens[0].Parts
	want := "a \"b\" \\ \n c"
	if len(parts) != 1 || parts[0].Kind != PartText || parts[0].Text != want {
		t.Errorf("parts = %+v, want single text part %q", parts, want)
	}
	if tokens[0].Lexeme != `"a \"b\" \\ \n c"` {
		t.Errorf("lexeme should keep the raw source, got %q", tokens[0].Lexeme)
	}
}

func sp(file string, l1, c1, l2, c2 int) Span {
	return Span{File: file, Start: Position{l1, c1}, End: Position{l2, c2}}
}

func TestLexSpans(t *testing.T) {
	const f = "main.ango"

	tests := []struct {
		name   string
		source string
		index  int
		typ    TokenType
		lexeme string
		span   Span
	}{
		{"identifier", "Asep: \"Halo.\"\n", 0, TokenIdentifier, "Asep", sp(f, 1, 1, 1, 5)},
		{"colon", "Asep: \"Halo.\"\n", 1, TokenColon, ":", sp(f, 1, 5, 1, 6)},
		{"string", "Asep: \"Halo.\"\n", 2, TokenString, `"Halo."`, sp(f, 1, 7, 1, 14)},
		{"real newline", "Asep: \"Halo.\"\n", 3, TokenNewline, "\n", sp(f, 1, 14, 1, 15)},
		{"eof after trailing newline", "Asep: \"Halo.\"\n", 4, TokenEOF, "", sp(f, 2, 1, 2, 1)},

		{"implicit newline", "jump a", 2, TokenNewline, "", sp(f, 1, 7, 1, 7)},
		{"eof without trailing newline", "jump a", 3, TokenEOF, "", sp(f, 1, 7, 1, 7)},

		{"columns count runes (accent)", `"héllo" x`, 1, TokenIdentifier, "x", sp(f, 1, 9, 1, 10)},
		{"columns count runes (emoji)", `"😀" x`, 1, TokenIdentifier, "x", sp(f, 1, 5, 1, 6)},

		{"indent", "choice:\n    \"A\":\n        jump a\nAsep: \"x\"\n", 3, TokenIndent, "    ", sp(f, 2, 1, 2, 5)},
		{"second indent", "choice:\n    \"A\":\n        jump a\nAsep: \"x\"\n", 7, TokenIndent, "        ", sp(f, 3, 1, 3, 9)},
		{"token after indent", "choice:\n    \"A\":\n        jump a\nAsep: \"x\"\n", 8, TokenJump, "jump", sp(f, 3, 9, 3, 13)},
		{"first dedent is zero-width", "choice:\n    \"A\":\n        jump a\nAsep: \"x\"\n", 11, TokenDedent, "", sp(f, 4, 1, 4, 1)},
		{"second dedent is zero-width", "choice:\n    \"A\":\n        jump a\nAsep: \"x\"\n", 12, TokenDedent, "", sp(f, 4, 1, 4, 1)},

		{"operator", "a <= b", 1, TokenLessEqual, "<=", sp(f, 1, 3, 1, 5)},
		{"float", "x = 12.50", 2, TokenFloat, "12.50", sp(f, 1, 5, 1, 10)},
		{"newline after comment", "a # hi\n", 1, TokenNewline, "\n", sp(f, 1, 7, 1, 8)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Lex(f, tt.source)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.index >= len(tokens) {
				t.Fatalf("only %d tokens: %s", len(tokens), render(tokens))
			}
			got := tokens[tt.index]
			if got.Type != tt.typ || got.Lexeme != tt.lexeme || got.Span != tt.span {
				t.Errorf("token[%d] = {%s %q %+v}, want {%s %q %+v}",
					tt.index, got.Type, got.Lexeme, got.Span, tt.typ, tt.lexeme, tt.span)
			}
		})
	}
}

func TestLexStringPartSpans(t *testing.T) {
	const f = "main.ango"
	tokens, err := Lex(f, `"Halo, [name]!"`)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := tokens[0].Span, sp(f, 1, 1, 1, 16); got != want {
		t.Errorf("string span = %+v, want %+v", got, want)
	}
	want := []StringPart{
		{PartText, "Halo, ", sp(f, 1, 2, 1, 8)},
		{PartVariable, "name", sp(f, 1, 8, 1, 14)},
		{PartText, "!", sp(f, 1, 14, 1, 15)},
	}
	if !reflect.DeepEqual(tokens[0].Parts, want) {
		t.Errorf("parts = %+v\nwant   %+v", tokens[0].Parts, want)
	}
}

func TestLexErrors(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"tab indentation", "choice:\n\t\"A\":\n",
			"test.ango:2:1: tabs are not allowed in indentation; use 4 spaces per level"},
		{"tab after spaces", "choice:\n    \tjump a\n",
			"test.ango:2:5: tabs are not allowed in indentation; use 4 spaces per level"},
		{"indentation not multiple of 4", "choice:\n  \"A\":\n",
			"test.ango:2:1: indentation must be a multiple of 4 spaces (found 2)"},
		{"indentation jumps two levels", "choice:\n        \"A\":\n",
			"test.ango:2:1: indentation jumps more than one level (found 8 spaces, expected at most 4)"},

		{"unterminated string", `Asep: "Halo.`, `test.ango:1:7: unterminated string`},
		{"string ends in backslash", `"abc\`, `test.ango:1:1: unterminated string`},
		{"unknown escape", `"a\qb"`, `test.ango:1:3: unknown escape sequence "\q"`},
		{"unclosed bracket", `"Halo [name"`,
			`test.ango:1:7: unclosed "[" in string; use "[[" for a literal "["`},
		{"unclosed bracket at end of line", `"Halo [name`,
			`test.ango:1:7: unclosed "[" in string; use "[[" for a literal "["`},
		{"invalid interpolation name", `"a [b c]"`,
			`test.ango:1:4: invalid variable name "b c" in interpolation`},
		{"empty interpolation", `"[]"`,
			`test.ango:1:2: invalid variable name "" in interpolation`},
		{"lone closing bracket", `"a ] b"`,
			`test.ango:1:4: unmatched "]" in string; use "]]" for a literal "]"`},

		{"unexpected character", `set x = $`, `test.ango:1:9: unexpected character '$'`},
		{"lone bang", `if !a:`, `test.ango:1:4: unexpected character '!' (use "not" for negation)`},
		{"number followed by letters", `set x = 12abc`, `test.ango:1:9: invalid number literal "12abc"`},
		{"trailing dot in number", `set x = 1.`, `test.ango:1:10: invalid number: expected digits after "."`},
		{"invalid utf-8", "Asep: \xff", `test.ango:1:7: invalid UTF-8 encoding`},
		{"error on a later line", "Asep: \"ok\"\nAsep: $", `test.ango:2:7: unexpected character '$'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Lex("test.ango", tt.source)
			if err == nil {
				t.Fatalf("expected error, got tokens: %s", render(tokens))
			}
			if tokens != nil {
				t.Errorf("tokens should be nil on error")
			}
			if err.Error() != tt.want {
				t.Errorf("error\n got: %s\nwant: %s", err, tt.want)
			}
			var le *Error
			if !errors.As(err, &le) || le.Span.File != "test.ango" {
				t.Errorf("error should be *Error carrying the file name, got %#v", err)
			}
		})
	}
}

func TestLexErrorWithoutFileName(t *testing.T) {
	_, err := Lex("", `$`)
	if err == nil || err.Error() != "1:1: unexpected character '$'" {
		t.Errorf("got %v", err)
	}
}

func TestLexNormalization(t *testing.T) {
	base := "Asep: \"Halo.\"\nchoice:\n    \"A\":\n        jump a\n"
	want := mustLex(t, base)

	variants := map[string]string{
		"crlf":     strings.ReplaceAll(base, "\n", "\r\n"),
		"cr":       strings.ReplaceAll(base, "\n", "\r"),
		"bom":      "\uFEFF" + base,
		"bom+crlf": "\uFEFF" + strings.ReplaceAll(base, "\n", "\r\n"),
	}
	for name, src := range variants {
		t.Run(name, func(t *testing.T) {
			got := mustLex(t, src)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("tokens differ from the \\n version\n got: %s\nwant: %s", render(got), render(want))
			}
		})
	}
}

func TestLexDot(t *testing.T) {
	tokens := mustLex(t, "jump a.b\n")
	want := []TokenType{TokenJump, TokenIdentifier, TokenDot, TokenIdentifier}
	for i, w := range want {
		if tokens[i].Type != w {
			t.Errorf("token %d: got %v, want %v", i, tokens[i].Type, w)
		}
	}
	if got := mustLex(t, "set x = 1.5\n")[3].Type; got != TokenFloat {
		t.Errorf("1.5 lexed as %v, want FLOAT", got)
	}
}

func TestLexKeywords(t *testing.T) {
	words := []string{
		"and", "or", "not", "label", "scene", "show", "hide", "choice", "jump",
		"call", "return", "set", "if", "else", "default", "namespace", "flip", "end", "at", "with", "true", "false",
	}
	if len(keywords) != len(words) {
		t.Errorf("keyword table has %d entries, test lists %d", len(keywords), len(words))
	}
	for _, w := range words {
		tokens := mustLex(t, w)
		if tokens[0].Type == TokenIdentifier {
			t.Errorf("%q lexed as an identifier", w)
		}
		if tokens[0].Type.String() != w {
			t.Errorf("%q: TokenType.String() = %q", w, tokens[0].Type.String())
		}
	}
}

// specSample is the corrected v0.1 example script from the spec discussion.
const specSample = `default found_key = false
default courage = 0
default player_name = "Asep"

label start

scene "school.png"
show asep "normal" at center

Asep: "Halo, [player_name]."

if found_key and courage > 5:
    jump secret

choice:
    "Ambil kunci" if not found_key:
        set found_key = true
        jump hallway

    "Periksa meja":
        set courage = courage + 1

    "Pergi":
        call hallway

Asep: "Apa yang harus kulakukan?"

label hallway

Asep: "Aku berada di lorong."

return

label secret

Asep: "Jadi ini tempatnya."

end
`

func TestLexSpecSampleInvariants(t *testing.T) {
	tokens := mustLex(t, specSample)

	if tokens[0].Type != TokenDefault {
		t.Errorf("first token = %s, want default", tokens[0].Type)
	}
	if last := tokens[len(tokens)-1]; last.Type != TokenEOF {
		t.Errorf("last token = %s, want EOF", last.Type)
	}

	indents, dedents := 0, 0
	for i, tok := range tokens {
		switch tok.Type {
		case TokenIndent:
			indents++
			if prev := tokens[i-1].Type; prev != TokenNewline {
				t.Errorf("token %d: INDENT preceded by %s, want NEWLINE", i, prev)
			}
		case TokenDedent:
			dedents++
			if prev := tokens[i-1].Type; prev != TokenNewline && prev != TokenDedent {
				t.Errorf("token %d: DEDENT preceded by %s, want NEWLINE or DEDENT", i, prev)
			}
		}
	}
	if indents != 5 || dedents != 5 {
		t.Errorf("indents=%d dedents=%d, want 5 and 5", indents, dedents)
	}
}
