package script

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// indentWidth is the number of spaces per indentation level.
const indentWidth = 4

// Lex converts Ango source into tokens.
//
// file is used only for the File field of spans and errors. Lexing stops at
// the first error, which is returned as *Error; tokens are nil in that case.
//
// Rules (Ango v0.1):
//   - "\r\n" and "\r" are normalized to "\n"; a leading BOM is ignored.
//   - Columns are 1-based and count runes.
//   - Indentation is 4 spaces per level. Tabs, widths that are not a multiple
//     of 4, and jumps of more than one level are errors. Dropping several
//     levels yields several DEDENT tokens.
//   - Blank lines and comment-only lines ("#" to end of line) never affect
//     indentation and produce no tokens.
//   - Every logical line ends with NEWLINE, including the last one when the
//     file has no trailing newline; DEDENTs and EOF come after it.
//   - Strings support \" \\ \n, "[[" -> "[", "]]" -> "]" and [name]
//     interpolation, which the lexer splits into Token.Parts.
func Lex(file, src string) ([]Token, error) {
	lx := &lexer{file: file}
	if err := lx.run(normalizeSource(src)); err != nil {
		return nil, err
	}
	return lx.tokens, nil
}

func normalizeSource(src string) string {
	src = strings.TrimPrefix(src, "\uFEFF")
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")
	return src
}

type lexer struct {
	file   string
	tokens []Token
	level  int // current indentation level
}

func (lx *lexer) run(src string) error {
	if err := lx.checkUTF8(src); err != nil {
		return err
	}

	lines := strings.Split(src, "\n")
	for i, text := range lines {
		hasNewline := i < len(lines)-1
		if err := lx.scanLine(i+1, []rune(text), hasNewline); err != nil {
			return err
		}
	}

	last := lines[len(lines)-1]
	eof := Position{Line: len(lines), Column: utf8.RuneCountInString(last) + 1}
	sp := Span{File: lx.file, Start: eof, End: eof}
	for ; lx.level > 0; lx.level-- {
		lx.emit(TokenDedent, "", sp)
	}
	lx.emit(TokenEOF, "", sp)
	return nil
}

func (lx *lexer) checkUTF8(src string) error {
	if utf8.ValidString(src) {
		return nil
	}
	line, col := 1, 1
	for i := 0; i < len(src); {
		r, size := utf8.DecodeRuneInString(src[i:])
		if r == utf8.RuneError && size == 1 {
			return lx.errAt(line, col, col+1, "invalid UTF-8 encoding")
		}
		if r == '\n' {
			line, col = line+1, 1
		} else {
			col++
		}
		i += size
	}
	return nil
}

// scanLine handles indentation and tokens for one physical line. line is
// 1-based; hasNewline reports whether the line is terminated by "\n".
func (lx *lexer) scanLine(line int, rs []rune, hasNewline bool) error {
	width, firstTab, i := 0, -1, 0
	for i < len(rs) && (rs[i] == ' ' || rs[i] == '\t') {
		if rs[i] == '\t' {
			if firstTab < 0 {
				firstTab = i
			}
		} else {
			width++
		}
		i++
	}

	// Blank and comment-only lines are invisible, whatever their indentation.
	if i == len(rs) || rs[i] == '#' {
		return nil
	}

	if firstTab >= 0 {
		return lx.errAt(line, firstTab+1, firstTab+2,
			"tabs are not allowed in indentation; use %d spaces per level", indentWidth)
	}
	if width%indentWidth != 0 {
		return lx.errAt(line, 1, width+1,
			"indentation must be a multiple of %d spaces (found %d)", indentWidth, width)
	}

	level := width / indentWidth
	switch {
	case level > lx.level+1:
		return lx.errAt(line, 1, width+1,
			"indentation jumps more than one level (found %d spaces, expected at most %d)",
			width, (lx.level+1)*indentWidth)
	case level == lx.level+1:
		lx.emit(TokenIndent, string(rs[:i]), lx.span(line, 1, width+1))
		lx.level = level
	case level < lx.level:
		for lx.level > level {
			lx.emit(TokenDedent, "", lx.span(line, width+1, width+1))
			lx.level--
		}
	}

	for i < len(rs) {
		r := rs[i]
		col := i + 1
		switch {
		case r == ' ' || r == '\t':
			i++

		case r == '#':
			i = len(rs)

		case isIdentStart(r):
			j := i + 1
			for j < len(rs) && isIdentPart(rs[j]) {
				j++
			}
			word := string(rs[i:j])
			tt := TokenIdentifier
			if kw, ok := keywords[word]; ok {
				tt = kw
			}
			lx.emit(tt, word, lx.span(line, col, j+1))
			i = j

		case isDigit(r):
			j, tt, err := lx.scanNumber(line, rs, i)
			if err != nil {
				return err
			}
			lx.emit(tt, string(rs[i:j]), lx.span(line, col, j+1))
			i = j

		case r == '"':
			j, parts, err := lx.scanString(line, rs, i)
			if err != nil {
				return err
			}
			lx.tokens = append(lx.tokens, Token{
				Type:   TokenString,
				Lexeme: string(rs[i:j]),
				Span:   lx.span(line, col, j+1),
				Parts:  parts,
			})
			i = j

		default:
			tt, n := scanOperator(rs, i)
			if n == 0 {
				if r == '!' {
					return lx.errAt(line, col, col+1, `unexpected character '!' (use "not" for negation)`)
				}
				return lx.errAt(line, col, col+1, "unexpected character %q", r)
			}
			lx.emit(tt, string(rs[i:i+n]), lx.span(line, col, col+n))
			i += n
		}
	}

	end := len(rs) + 1
	if hasNewline {
		lx.emit(TokenNewline, "\n", lx.span(line, end, end+1))
	} else {
		lx.emit(TokenNewline, "", lx.span(line, end, end))
	}
	return nil
}

// scanNumber scans an int or float starting at rs[i] and returns the index
// just past it.
func (lx *lexer) scanNumber(line int, rs []rune, i int) (int, TokenType, error) {
	j := i
	for j < len(rs) && isDigit(rs[j]) {
		j++
	}
	tt := TokenInt
	if j < len(rs) && rs[j] == '.' {
		if j+1 >= len(rs) || !isDigit(rs[j+1]) {
			return 0, 0, lx.errAt(line, j+1, j+2, `invalid number: expected digits after "."`)
		}
		j++
		for j < len(rs) && isDigit(rs[j]) {
			j++
		}
		tt = TokenFloat
	}
	if j < len(rs) && isIdentStart(rs[j]) {
		k := j
		for k < len(rs) && isIdentPart(rs[k]) {
			k++
		}
		return 0, 0, lx.errAt(line, i+1, k+1, "invalid number literal %q", string(rs[i:k]))
	}
	return j, tt, nil
}

// scanString scans a string literal whose opening quote is rs[start] and
// returns the index just past the closing quote.
func (lx *lexer) scanString(line int, rs []rune, start int) (int, []StringPart, error) {
	var (
		parts    []StringPart
		text     strings.Builder
		segStart = start + 1 // rune index where the current text segment began
	)
	flush := func(end int) {
		if text.Len() > 0 {
			parts = append(parts, StringPart{
				Kind: PartText,
				Text: text.String(),
				Span: lx.span(line, segStart+1, end+1),
			})
			text.Reset()
		}
	}
	unterminated := func() error {
		return lx.errAt(line, start+1, start+2, "unterminated string")
	}

	k := start + 1
	for k < len(rs) {
		r := rs[k]
		next := rune(0)
		if k+1 < len(rs) {
			next = rs[k+1]
		}

		switch r {
		case '"':
			flush(k)
			return k + 1, parts, nil

		case '\\':
			if k+1 >= len(rs) {
				return 0, nil, unterminated()
			}
			switch next {
			case '"':
				text.WriteRune('"')
			case '\\':
				text.WriteRune('\\')
			case 'n':
				text.WriteRune('\n')
			default:
				return 0, nil, lx.errAt(line, k+1, k+3, `unknown escape sequence "\%c"`, next)
			}
			k += 2

		case '[':
			if next == '[' {
				text.WriteRune('[')
				k += 2
				continue
			}
			m := k + 1
			for m < len(rs) && rs[m] != ']' && rs[m] != '"' {
				m++
			}
			if m >= len(rs) || rs[m] != ']' {
				return 0, nil, lx.errAt(line, k+1, k+2,
					`unclosed "[" in string; use "[[" for a literal "["`)
			}
			name := string(rs[k+1 : m])
			if !isIdentifier(name) {
				return 0, nil, lx.errAt(line, k+1, m+2,
					"invalid variable name %q in interpolation", name)
			}
			flush(k)
			parts = append(parts, StringPart{
				Kind: PartVariable,
				Text: name,
				Span: lx.span(line, k+1, m+2),
			})
			k = m + 1
			segStart = k

		case ']':
			if next == ']' {
				text.WriteRune(']')
				k += 2
				continue
			}
			return 0, nil, lx.errAt(line, k+1, k+2,
				`unmatched "]" in string; use "]]" for a literal "]"`)

		default:
			text.WriteRune(r)
			k++
		}
	}
	return 0, nil, unterminated()
}

// scanOperator returns the operator token at rs[i] and its length in runes,
// or n == 0 if rs[i] does not start an operator.
func scanOperator(rs []rune, i int) (tt TokenType, n int) {
	next := rune(0)
	if i+1 < len(rs) {
		next = rs[i+1]
	}
	switch rs[i] {
	case ':':
		return TokenColon, 1
	case ',':
		return TokenComma, 1
	case '.':
		return TokenDot, 1
	case '(':
		return TokenLParen, 1
	case ')':
		return TokenRParen, 1
	case '@':
		return TokenAtSign, 1
	case '+':
		return TokenPlus, 1
	case '-':
		return TokenMinus, 1
	case '*':
		return TokenStar, 1
	case '/':
		return TokenSlash, 1
	case '=':
		if next == '=' {
			return TokenEqualEqual, 2
		}
		return TokenEqual, 1
	case '!':
		if next == '=' {
			return TokenNotEqual, 2
		}
	case '<':
		if next == '=' {
			return TokenLessEqual, 2
		}
		return TokenLess, 1
	case '>':
		if next == '=' {
			return TokenGreaterEqual, 2
		}
		return TokenGreater, 1
	}
	return 0, 0
}

func (lx *lexer) emit(tt TokenType, lexeme string, sp Span) {
	lx.tokens = append(lx.tokens, Token{Type: tt, Lexeme: lexeme, Span: sp})
}

func (lx *lexer) span(line, startCol, endCol int) Span {
	return Span{
		File:  lx.file,
		Start: Position{Line: line, Column: startCol},
		End:   Position{Line: line, Column: endCol},
	}
}

func (lx *lexer) errAt(line, startCol, endCol int, format string, args ...any) error {
	return &Error{Span: lx.span(line, startCol, endCol), Msg: fmt.Sprintf(format, args...)}
}

func isDigit(r rune) bool { return r >= '0' && r <= '9' }

func isIdentStart(r rune) bool { return r == '_' || unicode.IsLetter(r) }

func isIdentPart(r rune) bool { return isIdentStart(r) || isDigit(r) }

func isIdentifier(s string) bool {
	for i, r := range s {
		if i == 0 && !isIdentStart(r) {
			return false
		}
		if !isIdentPart(r) {
			return false
		}
	}
	return s != ""
}
