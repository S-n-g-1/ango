package game

import (
	"ango/compiler"
	"ango/engine/value"
)

// SpriteState is a sprite currently on screen.
type SpriteState struct {
	Character  string
	Expression string
	Position   string
	Flip       bool
}

// PendingChoice is a menu waiting for the player.
// Enabled holds indices into Program.Choices[Choice].Options.
type PendingChoice struct {
	Choice  int
	Enabled []int
}

// GameState is everything needed to save and restore a game.
// It contains only plain data, so it can be serialized (e.g. encoding/json).
// The operand stack is empty at every event boundary and is not part of it.
type GameState struct {
	PC         int
	Vars       []value.Value // indexed by Program.Vars slot
	CallStack  []int         // return addresses
	Ended      bool
	Background string
	Sprites    []SpriteState
	Pending    *PendingChoice
}

func newState(prog *compiler.Program) GameState {
	vars := make([]value.Value, len(prog.Vars))
	for i, v := range prog.Vars {
		vars[i] = v.Default
	}
	return GameState{PC: prog.Entry, Vars: vars}
}

// Clone returns a deep copy.
func (s GameState) Clone() GameState {
	c := s
	c.Vars = append([]value.Value(nil), s.Vars...)
	c.CallStack = append([]int(nil), s.CallStack...)
	c.Sprites = append([]SpriteState(nil), s.Sprites...)
	if s.Pending != nil {
		p := *s.Pending
		p.Enabled = append([]int(nil), s.Pending.Enabled...)
		c.Pending = &p
	}
	return c
}

func (s *GameState) showSprite(sp SpriteState) {
	for i := range s.Sprites {
		if s.Sprites[i].Character == sp.Character {
			s.Sprites[i] = sp
			return
		}
	}
	s.Sprites = append(s.Sprites, sp)
}

func (s *GameState) hideSprite(target string) {
	kept := s.Sprites[:0]
	for _, sp := range s.Sprites {
		if sp.Character != target {
			kept = append(kept, sp)
		}
	}
	s.Sprites = kept
}
