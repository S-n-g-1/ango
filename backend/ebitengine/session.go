// Package ebitengine is the Ebitengine presentation backend for Ango.
//
// It knows nothing about the compiler, VM or GameState. It renders a View
// and reports player input back through a Session; a small adapter (in
// cmd/ango) bridges Session to engine/game.
package ebitengine

// Transition describes an optional fade on a background or sprite change,
// mirrored from the compiler's own Transition so this package stays
// decoupled from the compiler. Nil means a hard cut (no animation).
type Transition struct {
	Name     string
	Duration float64
}

// Sprite is one character currently shown on screen.
type Sprite struct {
	Char string // character id, e.g. "asep"
	Expr string // expression, e.g. "senang"
	Pos  string // "left", "center" (default) or "right"

	// Transition is set only when this sprite was just (re)shown by the
	// most recent Advance/Choose step. A sprite merely carried over from
	// an earlier step (nothing about it changed) has Transition == nil,
	// telling the renderer not to animate it.
	Transition *Transition
	Flip       bool
}

// HiddenSprite is a sprite that disappeared during the most recent
// Advance/Choose step. It no longer appears in View.Sprites; it's kept
// here just long enough for the renderer to play its exit animation.
type HiddenSprite struct {
	Char string
	Expr string
	Pos  string

	// Transition is the transition the `hide` command specified, if any.
	Transition *Transition
	Flip       bool
}

// View is the presentation snapshot the backend renders each frame.
type View struct {
	Background string // asset name from `scene`, "" = none

	// BackgroundTransition is set only on the step where Background just
	// changed; it is nil on every other frame, even while an animation
	// started by it may still be playing (the renderer tracks that itself).
	BackgroundTransition *Transition

	Sprites []Sprite // in draw order (last is on top)

	// JustHidden lists sprites removed during the most recent step, for
	// the renderer to fade out. It is only non-empty on that one step;
	// ongoing exit animations are the renderer's own concern.
	JustHidden []HiddenSprite

	Speaker string   // "" = narration
	Text    string   // current dialogue line, "" = no textbox
	Choices []string // non-empty = waiting for a choice
	Ended   bool     // script finished
}

// Session is what the backend needs from the engine.
type Session interface {
	// View returns the current presentation state.
	View() View
	// Advance moves past the current line (called on click/space/enter).
	Advance() error
	// Choose picks choice index i (0-based) while Choices is non-empty.
	Choose(i int) error
}
