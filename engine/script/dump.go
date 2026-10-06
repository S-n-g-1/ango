package script

import (
	"fmt"
	"strconv"
	"strings"

	"ango/engine/value"
)

// Dump renders a node as a single-line S-expression without source spans.
//
// Examples:
//
//	Asep: "Halo, [name]." -> (say "Asep" "Halo, " $name ".")
//	set t = t + 1       -> (set t (+ $t 1))
//
// Conventions:
//
//	"_"       = narration / empty speaker
//	"$x"      = variable
//	"say#id"  = dialogue with explicit @id
//	"(else)"  = else block when present
func Dump(n Node) string {
	switch n := n.(type) {
	case *ProgramNode:
		items := []string{"program"}

		for _, d := range n.Declarations {
			items = append(items, Dump(d))
		}

		for _, l := range n.Labels {
			items = append(items, Dump(l))
		}

		return sexp(items...)

	case *DefaultNode:
		return sexp(
			"default",
			n.Name,
			Dump(n.Value),
		)

	case *LabelNode:
		items := []string{
			"label",
			n.Name,
		}

		items = append(items, dumpAll(n.Body)...)

		return sexp(items...)

	case *SayNode:
		head := "say"

		if n.ID != "" {
			head += "#" + n.ID
		}

		speaker := "_"

		if n.Speaker != "" {
			speaker = strconv.Quote(n.Speaker)
		}

		items := []string{
			head,
			speaker,
		}

		items = append(items, dumpParts(n.Parts)...)

		return sexp(items...)

	case *ShowBackgroundNode:
		return sexp(
			"scene",
			strconv.Quote(n.Asset),
		)

	case *ShowSpriteNode:
		items := []string{
			"show",
			n.Character,
			strconv.Quote(n.Expression),
		}

		if n.Position != "" {
			items = append(
				items,
				"at",
				n.Position,
			)
		}

		return sexp(items...)

	case *HideNode:
		return sexp(
			"hide",
			n.Target,
		)

	case *SetNode:
		return sexp(
			"set",
			n.Name,
			Dump(n.Value),
		)

	case *JumpNode:
		return sexp(
			"jump",
			n.Label,
		)

	case *CallNode:
		return sexp(
			"call",
			n.Label,
		)

	case *ReturnNode:
		return sexp("return")

	case *EndNode:
		return sexp("end")

	case *IfNode:
		items := []string{
			"if",
			Dump(n.Condition),
		}

		thenItems := []string{"then"}
		thenItems = append(
			thenItems,
			dumpAll(n.Then)...,
		)

		items = append(
			items,
			sexp(thenItems...),
		)

		if len(n.Else) > 0 {
			elseItems := []string{"else"}

			elseItems = append(
				elseItems,
				dumpAll(n.Else)...,
			)

			items = append(
				items,
				sexp(elseItems...),
			)
		}

		return sexp(items...)

	case *ChoiceNode:
		items := []string{"choice"}

		for _, option := range n.Options {
			items = append(
				items,
				dumpChoiceOption(option),
			)
		}

		return sexp(items...)

	case *TextLiteral:
		return strconv.Quote(n.Text)

	case *TextVariable:
		return "$" + n.Name

	case *LiteralExpr:
		return dumpValue(n.Value)

	case *VariableExpr:
		return "$" + n.Name

	case *BinaryExpr:
		return sexp(
			n.Op.String(),
			Dump(n.Left),
			Dump(n.Right),
		)

	case *UnaryExpr:
		return sexp(
			n.Op.String(),
			Dump(n.Operand),
		)
	}

	return fmt.Sprintf("<%T>", n)
}

// ============================================================
// Choice Option
// ============================================================

// ChoiceOption bukan Node, jadi tidak bisa dikirim langsung ke Dump().
//
// Helper ini khusus untuk merender isi ChoiceOption.
func dumpChoiceOption(n ChoiceOption) string {
	items := []string{"option"}

	items = append(
		items,
		dumpParts(n.Text)...,
	)

	if n.Condition != nil {
		items = append(
			items,
			sexp(
				"when",
				Dump(n.Condition),
			),
		)
	}

	items = append(
		items,
		dumpAll(n.Body)...,
	)

	return sexp(items...)
}

// ============================================================
// Statement Helpers
// ============================================================

func dumpAll(stmts []Statement) []string {
	out := make([]string, len(stmts))

	for i, stmt := range stmts {
		out[i] = Dump(stmt)
	}

	return out
}

// ============================================================
// Text Helpers
// ============================================================

func dumpParts(parts []TextPart) []string {
	out := make([]string, len(parts))

	for i, part := range parts {
		out[i] = Dump(part)
	}

	return out
}

// ============================================================
// Value
// ============================================================

func dumpValue(v value.Value) string {
	switch v.Type {
	case value.Nil:
		return "nil"

	case value.Bool:
		return strconv.FormatBool(v.Bool)

	case value.Int:
		return strconv.FormatInt(
			v.Int,
			10,
		)

	case value.Float:
		s := strconv.FormatFloat(
			v.Float,
			'g',
			-1,
			64,
		)

		// Preserve the distinction between:
		//
		// 2   -> int
		// 2.0 -> float
		//
		if !strings.ContainsAny(
			s,
			".eE",
		) {
			s += ".0"
		}

		return s

	case value.String:
		return strconv.Quote(v.String)

	default:
		return "<unknown-value>"
	}
}

// ============================================================
// S-Expression
// ============================================================

func sexp(items ...string) string {
	return "(" + strings.Join(items, " ") + ")"
}
