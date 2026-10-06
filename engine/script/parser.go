package script

import (
	"fmt"
	"strconv"
	"strings"

	"ango/engine/value"
)

// Parse lexes and parses one Ango source file. It stops at the first error,
// which is returned as *Error (lexer errors pass through unchanged).
//
// The parser only checks syntax. Semantic checks (undeclared variables,
// unknown labels, duplicate labels, the "start" label, valid positions and
// assets) belong to the compiler and to a separate asset-validation pass.
func Parse(file, src string) (*ProgramNode, error) {
	tokens, err := Lex(file, src)
	if err != nil {
		return nil, err
	}
	return ParseTokens(tokens)
}

// ParseTokens parses a token stream produced by Lex.
func ParseTokens(tokens []Token) (prog *ProgramNode, err error) {
	p := &parser{tokens: tokens}
	defer func() {
		if r := recover(); r != nil {
			b, ok := r.(bailout)
			if !ok {
				panic(r)
			}
			prog, err = nil, b.err
		}
	}()
	return p.parseProgram(), nil
}

// bailout carries a syntax error up the recursive-descent stack.
type bailout struct{ err *Error }

type parser struct {
	tokens []Token
	pos    int

	// prevContent is the last consumed token that is not NEWLINE, INDENT or
	// DEDENT. Node spans end here.
	prevContent Token
}

// ---- token helpers ----

func (p *parser) peekAt(offset int) Token {
	if i := p.pos + offset; i < len(p.tokens) {
		return p.tokens[i]
	}
	// Past the end (or an empty stream): behave as EOF.
	var at Position
	file := ""
	if n := len(p.tokens); n > 0 {
		last := p.tokens[n-1].Span
		file, at = last.File, last.End
	}
	return Token{Type: TokenEOF, Span: Span{File: file, Start: at, End: at}}
}

func (p *parser) peek() Token { return p.peekAt(0) }

func (p *parser) at(tt TokenType) bool { return p.peek().Type == tt }

func (p *parser) next() Token {
	tok := p.peek()
	if tok.Type == TokenEOF {
		return tok // never advance past EOF
	}
	p.pos++
	switch tok.Type {
	case TokenNewline, TokenIndent, TokenDedent:
	default:
		p.prevContent = tok
	}
	return tok
}

func (p *parser) errorf(sp Span, format string, args ...any) bailout {
	return bailout{&Error{Span: sp, Msg: fmt.Sprintf(format, args...)}}
}

// expect consumes a token of the given type. want describes it for errors.
func (p *parser) expect(tt TokenType, want string) Token {
	tok := p.peek()
	if tok.Type != tt {
		panic(p.errorf(tok.Span, "expected %s, found %s", want, describe(tok)))
	}
	return p.next()
}

// expectName consumes an identifier. what names its role ("label name").
func (p *parser) expectName(what string) Token {
	tok := p.peek()
	switch {
	case tok.Type == TokenIdentifier:
		return p.next()
	case isKeyword(tok.Type):
		panic(p.errorf(tok.Span, "%q is a reserved word and cannot be used as a %s", tok.Lexeme, what))
	}
	panic(p.errorf(tok.Span, "expected %s (identifier), found %s", what, describe(tok)))
}

func (p *parser) expectNewline() {
	tok := p.peek()
	if tok.Type != TokenNewline {
		panic(p.errorf(tok.Span, "expected end of line, found %s", describe(tok)))
	}
	p.next()
}

func (p *parser) indentError(tok Token) bailout {
	return p.errorf(tok.Span,
		"unexpected indentation (label bodies are not indented; indent only inside if/else/choice blocks)")
}

// describe renders a token for "found ..." error messages.
func describe(t Token) string {
	switch t.Type {
	case TokenIdentifier:
		return fmt.Sprintf("identifier %q", t.Lexeme)
	case TokenString:
		return "string"
	case TokenInt, TokenFloat:
		return "number " + t.Lexeme
	case TokenNewline:
		return "end of line"
	case TokenIndent:
		return "indentation"
	case TokenDedent:
		return "end of block"
	case TokenEOF:
		return "end of file"
	}
	return fmt.Sprintf("%q", t.Type.String())
}

func isKeyword(tt TokenType) bool { return tt >= TokenAnd && tt <= TokenFalse }

func join(a, b Span) Span {
	return Span{File: a.File, Start: a.Start, End: b.End}
}

// ---- program, declarations, labels ----

func (p *parser) parseProgram() *ProgramNode {
	first := p.peek()
	prog := &ProgramNode{}

	if p.at(TokenNamespace) {
		prog.Namespace, prog.NamespaceSpan = p.parseNamespace()
	}

	for p.at(TokenDefault) {
		prog.Declarations = append(prog.Declarations, p.parseDefault())
	}

	for !p.at(TokenEOF) {
		tok := p.peek()
		switch tok.Type {
		case TokenLabel:
			l := p.parseLabel()
			l.Namespace = prog.Namespace
			prog.Labels = append(prog.Labels, l)
		case TokenNamespace:
			panic(p.errorf(tok.Span,
				`"namespace" must be the first declaration in the file, and may appear only once`))
		case TokenDefault:
			panic(p.errorf(tok.Span, `"default" must come before the first label`))
		case TokenIndent:
			panic(p.indentError(tok))
		default:
			panic(p.errorf(tok.Span,
				`expected "label" (statements must be inside a label), found %s`, describe(tok)))
		}
	}

	eof := p.peek()
	prog.Span = Span{File: eof.Span.File, Start: first.Span.Start, End: eof.Span.End}
	return prog
}

func (p *parser) parseNamespace() (string, Span) {
	start := p.next() // namespace
	name := p.expectName("namespace name")
	span := join(start.Span, p.prevContent.Span)
	p.expectNewline()
	return name.Lexeme, span
}

// parseLabelRef parses `name` or `ns.name`.
func (p *parser) parseLabelRef() string {
	name := p.expectName("label name")
	if p.at(TokenDot) {
		p.next()
		member := p.expectName(`label name after "."`)
		return name.Lexeme + "." + member.Lexeme
	}
	return name.Lexeme
}
func (p *parser) parseDefault() *DefaultNode {
	start := p.next() // default
	name := p.expectName("variable name")
	p.expect(TokenEqual, `"="`)
	val := p.parseDefaultLiteral()
	end := p.prevContent.Span
	p.expectNewline()
	return &DefaultNode{Span: join(start.Span, end), Name: name.Lexeme, Value: val}
}

// parseDefaultLiteral accepts a literal, optionally preceded by "-" for
// numbers. v0.1 restricts default values to literals.
func (p *parser) parseDefaultLiteral() Expression {
	tok := p.next()
	if tok.Type == TokenMinus {
		num := p.peek()
		if num.Type != TokenInt && num.Type != TokenFloat {
			panic(p.errorf(num.Span, `expected a number after "-", found %s`, describe(num)))
		}
		p.next()
		lit := p.literal(num, true)
		lit.Span = join(tok.Span, num.Span)
		return lit
	}
	switch tok.Type {
	case TokenInt, TokenFloat, TokenString, TokenTrue, TokenFalse:
		return p.literal(tok, false)
	}
	panic(p.errorf(tok.Span,
		`"default" values must be literals (number, string, true or false), found %s`, describe(tok)))
}

func (p *parser) parseLabel() *LabelNode {
	start := p.next() // label
	name := p.expectName("label name")
	p.expectNewline()

	lbl := &LabelNode{Name: name.Lexeme}
	for !p.at(TokenLabel) && !p.at(TokenEOF) {
		lbl.Body = append(lbl.Body, p.parseStatement())
	}
	lbl.Span = join(start.Span, p.prevContent.Span)
	return lbl
}

// ---- statements ----

func (p *parser) parseStatement() Statement {
	tok := p.peek()
	switch tok.Type {
	case TokenAtSign, TokenIdentifier, TokenString:
		return p.parseSay()
	case TokenScene:
		return p.parseScene()
	case TokenShow:
		return p.parseShow()
	case TokenHide:
		return p.parseHide()
	case TokenSet:
		return p.parseSet()
	case TokenJump:
		p.next()
		name := p.parseLabelRef()
		return &JumpNode{Span: p.finishLine(tok), Label: name}
	case TokenCall:
		p.next()
		name := p.parseLabelRef()
		return &CallNode{Span: p.finishLine(tok), Label: name}
	case TokenNamespace:
		panic(p.errorf(tok.Span, `"namespace" must be the first declaration in the file`))
	case TokenReturn:
		p.next()
		return &ReturnNode{Span: p.finishLine(tok)}
	case TokenEnd:
		p.next()
		return &EndNode{Span: p.finishLine(tok)}
	case TokenIf:
		return p.parseIf()
	case TokenChoice:
		return p.parseChoice()

	case TokenLabel:
		panic(p.errorf(tok.Span, `"label" is only allowed at the top level`))
	case TokenDefault:
		panic(p.errorf(tok.Span, `"default" must come before the first label`))
	case TokenElse:
		panic(p.errorf(tok.Span, `"else" without a matching "if"`))
	case TokenIndent:
		panic(p.indentError(tok))
	}

	if isKeyword(tok.Type) && p.peekAt(1).Type == TokenColon {
		panic(p.errorf(tok.Span,
			`%q is a reserved word; write the speaker as a string: "%s": "..."`, tok.Lexeme, tok.Lexeme))
	}
	panic(p.errorf(tok.Span, "expected a statement, found %s", describe(tok)))
}

// finishLine ends a simple statement that began with token start: it takes
// the span up to the last content token, then requires end of line.
func (p *parser) finishLine(start Token) Span {
	sp := join(start.Span, p.prevContent.Span)
	p.expectNewline()
	return sp
}

func (p *parser) parseSay() *SayNode {
	start := p.peek()
	id := ""
	if start.Type == TokenAtSign {
		id = p.parseIDAnnotation()
		if after := p.peek(); after.Type != TokenIdentifier && after.Type != TokenString {
			panic(p.errorf(after.Span, "expected a dialogue line after @id, found %s", describe(after)))
		}
	}

	var speaker string
	var text Token
	first := p.next()
	switch first.Type {
	case TokenIdentifier:
		if !p.at(TokenColon) {
			panic(p.errorf(p.peek().Span, `expected ":" after speaker name, found %s`, describe(p.peek())))
		}
		p.next()
		speaker = first.Lexeme
		text = p.expect(TokenString, "dialogue text (string)")
	case TokenString:
		if p.at(TokenColon) {
			speaker = p.plainString(first)
			if speaker == "" {
				panic(p.errorf(first.Span, "speaker name cannot be empty"))
			}
			p.next()
			text = p.expect(TokenString, "dialogue text (string)")
		} else {
			text = first // narration
		}
	}

	sp := join(start.Span, p.prevContent.Span)
	p.expectNewline()
	return &SayNode{Span: sp, ID: id, Speaker: speaker, Parts: p.textParts(text)}
}

// parseIDAnnotation parses @id("...") on its own line and returns the id.
func (p *parser) parseIDAnnotation() string {
	p.next() // @
	name := p.peek()
	if name.Type != TokenIdentifier && !isKeyword(name.Type) {
		panic(p.errorf(name.Span, `expected annotation name after "@", found %s`, describe(name)))
	}
	if name.Lexeme != "id" {
		panic(p.errorf(name.Span, `unknown annotation "@%s"; only @id("...") is supported`, name.Lexeme))
	}
	p.next()
	p.expect(TokenLParen, `"("`)
	str := p.expect(TokenString, "annotation value (string)")
	id := p.plainString(str)
	if id == "" {
		panic(p.errorf(str.Span, "@id value cannot be empty"))
	}
	p.expect(TokenRParen, `")"`)
	p.expectNewline()
	return id
}

func (p *parser) parseScene() *ShowBackgroundNode {
	start := p.next() // scene
	asset := p.expect(TokenString, "background file (string)")
	name := p.plainString(asset)
	trans := p.parseTransition()
	return &ShowBackgroundNode{Span: p.finishLine(start), Asset: name, Transition: trans}
}

func (p *parser) parseShow() *ShowSpriteNode {
	start := p.next() // show
	if tok := p.peek(); tok.Type == TokenString {
		panic(p.errorf(tok.Span,
			`expected character name (identifier), found string; use scene "..." for backgrounds`))
	}
	char := p.expectName("character name")
	expr := p.expect(TokenString, "expression name (string)")
	node := &ShowSpriteNode{Character: char.Lexeme, Expression: p.plainString(expr)}
	if p.at(TokenAt) {
		p.next()
		node.Position = p.expectName("position name").Lexeme
	}
	if p.at(TokenFlip) {
		p.next()
		node.Flip = true
	}
	node.Transition = p.parseTransition()
	node.Span = p.finishLine(start)
	return node
}

func (p *parser) parseHide() *HideNode {
	start := p.next() // hide
	target := p.expectName("character name")
	trans := p.parseTransition()
	return &HideNode{Span: p.finishLine(start), Target: target.Lexeme, Transition: trans}
}

// parseTransition parses an optional "with <name> <duration>" clause,
// e.g. "with fade 0.5". It returns nil when no "with" is present, so
// callers can assign the result straight to a node's Transition field.
func (p *parser) parseTransition() *Transition {
	if !p.at(TokenWith) {
		return nil
	}
	p.next() // with
	name := p.expectName("transition name").Lexeme
	dur := p.parseDuration()
	return &Transition{Name: name, Duration: dur}
}

// parseDuration parses a bare numeric literal (int or float) as seconds,
// reusing the same literal() conversion as ordinary expressions so "0.5"
// and "1" behave identically to numbers used anywhere else in a script.
func (p *parser) parseDuration() float64 {
	tok := p.peek()
	switch tok.Type {
	case TokenInt, TokenFloat:
		p.next()
		lit := p.literal(tok, false)
		switch lit.Value.Type {
		case value.Int:
			return float64(lit.Value.Int)
		case value.Float:
			return lit.Value.Float
		}
	}
	panic(p.errorf(tok.Span, "expected transition duration (number), found %s", describe(tok)))
}

func (p *parser) parseSet() *SetNode {
	start := p.next() // set
	name := p.expectName("variable name")
	p.expect(TokenEqual, `"="`)
	val := p.parseExpr()
	return &SetNode{Span: p.finishLine(start), Name: name.Lexeme, Value: val}
}

func (p *parser) parseIf() *IfNode {
	start := p.next() // if
	node := &IfNode{Condition: p.parseExpr()}
	node.Then = p.parseBlock(`"if"`)
	if p.at(TokenElse) {
		p.next()
		if tok := p.peek(); tok.Type == TokenIf {
			panic(p.errorf(tok.Span, `"else if" is not supported; nest an "if" inside "else:"`))
		}
		node.Else = p.parseBlock(`"else"`)
	}
	node.Span = join(start.Span, p.prevContent.Span)
	return node
}

func (p *parser) parseChoice() *ChoiceNode {
	start := p.next() // choice
	p.parseBlockHeader(`"choice"`)
	node := &ChoiceNode{}
	for !p.at(TokenDedent) {
		if p.at(TokenEOF) {
			panic(p.errorf(p.peek().Span, "unexpected end of file inside a choice"))
		}
		node.Options = append(node.Options, p.parseOption())
	}
	p.next() // dedent
	node.Span = join(start.Span, p.prevContent.Span)
	return node
}

func (p *parser) parseOption() ChoiceOption {
	tok := p.peek()
	if tok.Type != TokenString {
		panic(p.errorf(tok.Span, "expected an option string, found %s", describe(tok)))
	}
	p.next()
	opt := ChoiceOption{Text: p.textParts(tok)}
	if p.at(TokenIf) {
		p.next()
		opt.Condition = p.parseExpr()
	}
	opt.Body = p.parseBlock("the option")
	opt.Span = join(tok.Span, p.prevContent.Span)
	return opt
}

// parseBlockHeader consumes ":" NEWLINE INDENT. owner names the construct
// for error messages.
func (p *parser) parseBlockHeader(owner string) {
	p.expect(TokenColon, `":"`)
	if tok := p.peek(); tok.Type != TokenNewline {
		panic(p.errorf(tok.Span, `expected end of line after ":"; blocks go on their own indented lines`))
	}
	p.next()
	if tok := p.peek(); tok.Type != TokenIndent {
		panic(p.errorf(tok.Span, "expected an indented block after %s", owner))
	}
	p.next()
}

// parseBlock parses a header followed by statements up to the matching DEDENT.
func (p *parser) parseBlock(owner string) []Statement {
	p.parseBlockHeader(owner)
	var body []Statement
	for !p.at(TokenDedent) {
		if p.at(TokenEOF) {
			panic(p.errorf(p.peek().Span, "unexpected end of file inside a block"))
		}
		body = append(body, p.parseStatement())
	}
	p.next() // dedent
	return body
}

// ---- text ----

// textParts converts a string token's parts into AST text parts.
func (p *parser) textParts(tok Token) []TextPart {
	var parts []TextPart
	for _, part := range tok.Parts {
		if part.Kind == PartVariable {
			parts = append(parts, &TextVariable{Span: part.Span, Name: part.Text})
		} else {
			parts = append(parts, &TextLiteral{Span: part.Span, Text: part.Text})
		}
	}
	return parts
}

// plainString returns the text of a string token that must not contain
// [variable] interpolation (speakers, asset names, expression strings, ...).
func (p *parser) plainString(tok Token) string {
	var sb strings.Builder
	for _, part := range tok.Parts {
		if part.Kind == PartVariable {
			panic(p.errorf(part.Span, "interpolation is only allowed in dialogue and choice text"))
		}
		sb.WriteString(part.Text)
	}
	return sb.String()
}

// ---- expressions ----
//
// Precedence, lowest to highest:
//
//	or
//	and
//	not
//	== != < <= > >=   (not chainable)
//	+ -
//	* /
//	unary -
//	literal, variable, ( expr )

func (p *parser) parseExpr() Expression { return p.parseOr() }

func (p *parser) parseOr() Expression {
	left := p.parseAnd()
	for p.at(TokenOr) {
		p.next()
		right := p.parseAnd()
		left = &BinaryExpr{Span: join(left.Pos(), right.Pos()), Left: left, Op: BinaryOr, Right: right}
	}
	return left
}

func (p *parser) parseAnd() Expression {
	left := p.parseNot()
	for p.at(TokenAnd) {
		p.next()
		right := p.parseNot()
		left = &BinaryExpr{Span: join(left.Pos(), right.Pos()), Left: left, Op: BinaryAnd, Right: right}
	}
	return left
}

func (p *parser) parseNot() Expression {
	if p.at(TokenNot) {
		tok := p.next()
		operand := p.parseNot()
		return &UnaryExpr{Span: join(tok.Span, operand.Pos()), Op: UnaryNot, Operand: operand}
	}
	return p.parseComparison()
}

var comparisonOps = map[TokenType]BinaryOp{
	TokenEqualEqual:   BinaryEqual,
	TokenNotEqual:     BinaryNotEqual,
	TokenLess:         BinaryLess,
	TokenLessEqual:    BinaryLessEqual,
	TokenGreater:      BinaryGreater,
	TokenGreaterEqual: BinaryGreaterEqual,
}

func (p *parser) parseComparison() Expression {
	left := p.parseAdditive()
	op, ok := comparisonOps[p.peek().Type]
	if !ok {
		return left
	}
	p.next()
	right := p.parseAdditive()
	if tok := p.peek(); isComparison(tok.Type) {
		panic(p.errorf(tok.Span, `comparison operators cannot be chained; use "and" to combine comparisons`))
	}
	return &BinaryExpr{Span: join(left.Pos(), right.Pos()), Left: left, Op: op, Right: right}
}

func isComparison(tt TokenType) bool {
	_, ok := comparisonOps[tt]
	return ok
}

func (p *parser) parseAdditive() Expression {
	left := p.parseMultiplicative()
	for p.at(TokenPlus) || p.at(TokenMinus) {
		op := BinaryAdd
		if p.next().Type == TokenMinus {
			op = BinarySub
		}
		right := p.parseMultiplicative()
		left = &BinaryExpr{Span: join(left.Pos(), right.Pos()), Left: left, Op: op, Right: right}
	}
	return left
}

func (p *parser) parseMultiplicative() Expression {
	left := p.parseUnary()
	for p.at(TokenStar) || p.at(TokenSlash) {
		op := BinaryMul
		if p.next().Type == TokenSlash {
			op = BinaryDiv
		}
		right := p.parseUnary()
		left = &BinaryExpr{Span: join(left.Pos(), right.Pos()), Left: left, Op: op, Right: right}
	}
	return left
}

func (p *parser) parseUnary() Expression {
	if p.at(TokenMinus) {
		tok := p.next()
		operand := p.parseUnary()
		return &UnaryExpr{Span: join(tok.Span, operand.Pos()), Op: UnaryNeg, Operand: operand}
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() Expression {
	tok := p.peek()
	switch tok.Type {
	case TokenInt, TokenFloat, TokenString, TokenTrue, TokenFalse:
		p.next()
		return p.literal(tok, false)
	case TokenIdentifier:
		p.next()
		return &VariableExpr{Span: tok.Span, Name: tok.Lexeme}
	case TokenLParen:
		p.next()
		expr := p.parseExpr()
		p.expect(TokenRParen, `")"`)
		return expr
	}
	panic(p.errorf(tok.Span, "expected an expression, found %s", describe(tok)))
}

// literal builds a LiteralExpr from a literal token. negate applies a
// preceding "-" to a number (used by default declarations).
func (p *parser) literal(tok Token, negate bool) *LiteralExpr {
	sp := tok.Span
	switch tok.Type {
	case TokenInt:
		lex := tok.Lexeme
		if negate {
			lex = "-" + lex
		}
		n, err := strconv.ParseInt(lex, 10, 64)
		if err != nil {
			panic(p.errorf(sp, "integer literal out of range"))
		}
		return &LiteralExpr{Span: sp, Value: value.OfInt(n)}
	case TokenFloat:
		lex := tok.Lexeme
		if negate {
			lex = "-" + lex
		}
		f, err := strconv.ParseFloat(lex, 64)
		if err != nil {
			panic(p.errorf(sp, "float literal out of range"))
		}
		return &LiteralExpr{Span: sp, Value: value.OfFloat(f)}
	case TokenString:
		return &LiteralExpr{Span: sp, Value: value.OfString(p.plainString(tok))}
	case TokenTrue:
		return &LiteralExpr{Span: sp, Value: value.OfBool(true)}
	case TokenFalse:
		return &LiteralExpr{Span: sp, Value: value.OfBool(false)}
	}
	panic(fmt.Sprintf("script: literal called with %s", tok.Type)) // internal error
}
