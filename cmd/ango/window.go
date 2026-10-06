package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"ango/backend/ebitengine"
	"ango/compiler"
	"ango/engine/game"
)

// playWindow builds path (a single script or a story directory, see
// buildPath) and plays it in a graphical window instead of the terminal.
// It returns the process exit code. assetsDir overrides the default when
// non-empty; a wrong or missing directory still runs, since the backend
// falls back to placeholder art.
//
// -check and -auto have no effect here: the window always waits for real
// clicks, taps or key presses.
func playWindow(path, assetsDir string, errOut io.Writer) int {
	prog, err := buildPath(path)
	if err != nil {
		report(errOut, err)
		return 1
	}
	if assetsDir == "" {
		assetsDir = defaultAssetsDir(path)
	}
	title := "Ango — " + filepath.Base(filepath.Clean(path))
	if err := runWindow(game.New(prog), assetsDir, title); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return 0
}

// defaultAssetsDir is the assets folder for path: "<path>/assets" when
// path is a story directory (so a multi-file story keeps one shared
// assets/ folder at its root), or "<dir of path>/assets" when path is a
// single script, e.g. "examples/basic/demo.ango" -> "examples/basic/assets".
func defaultAssetsDir(path string) string {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return filepath.Join(path, "assets")
	}
	return filepath.Join(filepath.Dir(path), "assets")
}

// session implements ebitengine.Session over a running *game.VM. It never
// pauses on EventScene/EventShow/EventHide: those just update state, and
// the VM is pumped forward until a line, a choice, or the end is reached.
//
// Along the way it collects the transition metadata carried by those
// events (see engine/game.Event.Transition) and attaches it to the View
// it produces, so the renderer (game.go) knows what, if anything, to
// animate since the previous View.
type session struct {
	vm   *game.VM
	view ebitengine.View
}

// newSession wraps vm and pumps it to the first line, choice, or end.
func newSession(vm *game.VM) (*session, error) {
	s := &session{vm: vm}
	if err := s.pump(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *session) View() ebitengine.View { return s.view }

// Advance is a no-op while a choice is pending or the script has ended;
// Game.Update never calls it then, but staying safe costs nothing.
func (s *session) Advance() error {
	if s.view.Ended || len(s.view.Choices) > 0 {
		return nil
	}
	return s.pump()
}

func (s *session) Choose(i int) error {
	if err := s.vm.Choose(i); err != nil {
		return err
	}
	return s.pump()
}

// pump drives the VM with Next until it produces a line, a choice, or the
// end. Scene/Show/Hide events don't pause it, but it tracks what they did
// so the View it finally builds carries enough transition metadata for
// game.go to animate exactly what changed since the last pause:
//
//   - sceneChanged/sceneTrans: whether the background changed this step,
//     and the transition of the *last* `scene` command if it changed more
//     than once before the next pause (only the final one is ever seen).
//   - justShown: the transition each (re)shown sprite was given, keyed by
//     character; a sprite the player already saw and that wasn't touched
//     again has no entry, so it won't be re-animated.
//   - hidden: sprites removed this step, captured with their pre-hide
//     Expr/Pos (from the state just before the hide ran) so the renderer
//     can still draw them while they fade out.
func (s *session) pump() error {
	justShown := map[string]*ebitengine.Transition{}
	var hidden []ebitengine.HiddenSprite
	var sceneTrans *ebitengine.Transition
	sceneChanged := false

	for {
		before := s.vm.State() // cheap clone; only used if this step hides someone
		ev, err := s.vm.Next()
		if err != nil {
			return err
		}
		switch ev.Kind {
		case game.EventScene:
			sceneChanged = true
			sceneTrans = toViewTransition(ev.Transition)

		case game.EventShow:
			justShown[ev.Sprite.Character] = toViewTransition(ev.Transition)

		case game.EventHide:
			delete(justShown, ev.Target)
			for _, sp := range before.Sprites {
				if sp.Character == ev.Target {
					hidden = append(hidden, ebitengine.HiddenSprite{
						Char:       sp.Character,
						Expr:       sp.Expression,
						Pos:        sp.Position,
						Flip:       sp.Flip,
						Transition: toViewTransition(ev.Transition),
					})
					break
				}
			}

		case game.EventSay:
			s.finish(sceneChanged, sceneTrans, justShown, hidden)
			s.view.Speaker, s.view.Text, s.view.Choices = ev.Speaker, ev.Text, nil
			return nil

		case game.EventChoice:
			s.finish(sceneChanged, sceneTrans, justShown, hidden)
			s.view.Speaker, s.view.Text, s.view.Choices = "", "", ev.Options
			return nil

		case game.EventEnd:
			s.finish(sceneChanged, sceneTrans, justShown, hidden)
			s.view.Speaker, s.view.Text, s.view.Choices = "", "", nil
			s.view.Ended = true
			return nil
		}
		// EventScene, EventShow, EventHide: tracked above, keep pumping.
	}
}

// finish refreshes background/sprites from the VM's current state and
// attaches the transition metadata collected over this pump.
func (s *session) finish(
	sceneChanged bool,
	sceneTrans *ebitengine.Transition,
	justShown map[string]*ebitengine.Transition,
	hidden []ebitengine.HiddenSprite,
) {
	st := s.vm.State()

	if sceneChanged {
		s.view.Background = st.Background
		s.view.BackgroundTransition = sceneTrans
	} else {
		s.view.BackgroundTransition = nil
	}

	sprites := make([]ebitengine.Sprite, 0, len(st.Sprites))
	for _, sp := range st.Sprites {
		sprites = append(sprites, ebitengine.Sprite{
			Char:       sp.Character,
			Expr:       sp.Expression,
			Pos:        sp.Position,
			Flip:       sp.Flip,
			Transition: justShown[sp.Character],
		})
	}
	s.view.Sprites = sprites
	s.view.JustHidden = hidden
}

// toViewTransition copies a compiler.Transition into the backend's own
// Transition type, so this package doesn't depend on the compiler's.
func toViewTransition(t *compiler.Transition) *ebitengine.Transition {
	if t == nil {
		return nil
	}
	return &ebitengine.Transition{Name: t.Name, Duration: t.Duration}
}

// runWindow opens the Ebitengine window and plays vm until the script ends
// or the window is closed. assetsDir is the folder holding backgrounds/ and
// characters/; missing files fall back to placeholders, so an empty or
// wrong dir still runs.
func runWindow(vm *game.VM, assetsDir, title string) error {
	sess, err := newSession(vm)
	if err != nil {
		return fmt.Errorf("start session: %w", err)
	}
	if err := ebitengine.Run(sess, ebitengine.Options{
		Title:  title,
		Assets: os.DirFS(assetsDir),
	}); err != nil {
		return fmt.Errorf("open window: %w", err)
	}
	return nil
}
