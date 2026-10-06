package ebitengine

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"io/fs"
	"log"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"
)

// Logical resolution. The window can be any size; Ebitengine scales it.
const (
	screenW = 1280
	screenH = 720

	fontSize = 26
	lineH    = 34
	maxLines = 4

	boxX, boxY, boxW, boxH = 40, 520, 1200, 170

	choiceW, choiceH, choiceGap = 720, 56, 16

	spriteH = 600.0 // sprites are scaled to this height

	// transitionFade is the only transition name actually animated in this
	// MVP. Anything else (or no "with" clause) is a hard cut; an unknown
	// name is logged once, the same way a missing asset is.
	transitionFade = "fade"
)

var (
	colWhite = color.NRGBA{255, 255, 255, 255}
	colBox   = color.NRGBA{0, 0, 0, 190}
	colPlate = color.NRGBA{30, 30, 70, 230}
	colBtn   = color.NRGBA{20, 20, 40, 220}
	colBtnHi = color.NRGBA{70, 70, 140, 240}
)

// Options configures Run.
type Options struct {
	Title  string
	Assets fs.FS // rooted at the assets dir; may be nil (placeholders only)
}

// Run opens a window and plays sess until the script ends or the window closes.
func Run(sess Session, opts Options) error {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		return fmt.Errorf("load font: %w", err)
	}
	th := loadTheme(opts.Assets)
	g := &Game{
		sess:   sess,
		assets: newAssets(opts.Assets),
		th:     th,
		face:   &text.GoTextFace{Source: src, Size: th.fontSize},
		pixel:  ebiten.NewImage(1, 1),
	}
	g.pixel.Fill(color.White)
	g.sync()

	title := opts.Title
	if title == "" {
		title = "Ango"
	}
	g.initMenu(title)
	ebiten.SetWindowTitle(title)
	ebiten.SetWindowSize(screenW*3/4, screenH*3/4)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	return ebiten.RunGame(g)
}

// bgAnim is the background crossfade in progress, if any. The zero value
// (dur == 0) means "not animating": drawBackground then just draws the
// current background at full opacity, exactly like before transitions
// existed.
type bgAnim struct {
	prevName string
	start    time.Time
	dur      float64 // seconds
}

// spriteAnim is one sprite's entrance or exit fade.
type spriteAnim struct {
	sprite Sprite // Char/Expr/Pos to draw while animating
	start  time.Time
	dur    float64 // seconds
}

// progressAt returns how far into [0,1] an animation starting at start
// with duration dur (seconds) currently is. dur <= 0 means "no animation":
// always fully done (1), so callers don't need a separate branch for it.
func progressAt(start time.Time, dur float64) float64 {
	if dur <= 0 {
		return 1
	}
	t := time.Since(start).Seconds() / dur
	switch {
	case t < 0:
		return 0
	case t > 1:
		return 1
	}
	return t
}

var warnedTransitions = map[string]bool{}

// warnUnknownTransition logs once per unrecognized transition name, the
// same pattern used for missing assets: the script still runs, just with
// a hard cut instead of the animation it asked for.
func warnUnknownTransition(name string) {
	if warnedTransitions[name] {
		return
	}
	warnedTransitions[name] = true
	log.Printf("ebitengine: unknown transition %q (using a hard cut)", name)
}

// fadeDuration returns t's duration in seconds if it asks for a "fade",
// warning once and returning 0 (instant) for anything else, including nil.
func fadeDuration(t *Transition) float64 {
	if t == nil {
		return 0
	}
	if t.Name != transitionFade {
		warnUnknownTransition(t.Name)
		return 0
	}
	if t.Duration <= 0 {
		return 0
	}
	return t.Duration
}

// Game implements ebiten.Game.
type Game struct {
	sess   Session
	assets *assets
	face   *text.GoTextFace
	pixel  *ebiten.Image // 1x1 white, scaled to draw rectangles

	view  View
	lines []string // wrapped dialogue for the current view

	// Presentation-only animation state. Never persisted (there is nothing
	// here the VM or a save file needs to know about); rebuilt by sync()
	// from whatever View just reported changed.
	bg       bgAnim
	entering map[string]spriteAnim // keyed by Char; entrance fade in progress
	exiting  []spriteAnim          // sprites currently fading out after a hide

	th         theme
	mode       uiMode
	items      []menuItem
	sel        int
	lastCursor image.Point
}

// sync re-reads the view and, from what changed, decides what (if
// anything) should animate until the next sync. Call it after every
// Advance/Choose.
func (g *Game) sync() {
	prevBackground := g.view.Background
	g.view = g.sess.View()
	g.lines = wrap(g.view.Text, g.face, float64(g.th.box.Dx())-2*g.th.padX)

	now := time.Now()

	if g.view.Background != prevBackground {
		g.bg = bgAnim{
			prevName: prevBackground,
			start:    now,
			dur:      fadeDuration(g.view.BackgroundTransition),
		}
	}

	entering := make(map[string]spriteAnim, len(g.view.Sprites))
	for _, sp := range g.view.Sprites {
		if sp.Transition == nil {
			continue // carried over from an earlier step; nothing changed
		}
		if dur := fadeDuration(sp.Transition); dur > 0 {
			entering[sp.Char] = spriteAnim{sprite: sp, start: now, dur: dur}
		}
	}
	g.entering = entering

	exiting := g.exiting[:0]
	for _, h := range g.view.JustHidden {
		if dur := fadeDuration(h.Transition); dur > 0 {
			exiting = append(exiting, spriteAnim{
				sprite: Sprite{Char: h.Char, Expr: h.Expr, Pos: h.Pos, Flip: h.Flip},
				start:  now,
				dur:    dur,
			})
		}
	}
	g.exiting = exiting
}

func (g *Game) Layout(outW, outH int) (int, int) { return screenW, screenH }

func (g *Game) Update() error {
	if g.mode != modePlaying {
		return g.updateMenu()
	}
	if g.th.menu.Enabled && inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.openPause()
		return nil
	}
	switch {
	case g.view.Ended:
		if continuePressed() {
			return ebiten.Termination
		}
	case len(g.view.Choices) > 0:
		if i := g.pickedChoice(); i >= 0 {
			if err := g.sess.Choose(i); err != nil {
				return err
			}
			g.sync()
		}
	default:
		if continuePressed() {
			if err := g.sess.Advance(); err != nil {
				return err
			}
			g.sync()
		}
	}
	return nil
}

// --- input ---

// justPressedPoints returns positions (logical coords) of new clicks/touches.
func justPressedPoints() []image.Point {
	var pts []image.Point
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		pts = append(pts, image.Pt(x, y))
	}
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		pts = append(pts, image.Pt(x, y))
	}
	return pts
}

func continuePressed() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		return true
	}
	return len(justPressedPoints()) > 0
}

// pickedChoice returns the chosen index (click, touch or keys 1-9) or -1.
func (g *Game) pickedChoice() int {
	rects := g.choiceRects(len(g.view.Choices))
	for _, p := range justPressedPoints() {
		for i, r := range rects {
			if p.In(r) {
				return i
			}
		}
	}
	for i := range rects {
		if i > 8 {
			break
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyDigit1 + ebiten.Key(i)) {
			return i
		}
	}
	return -1
}

// choiceRects lays out n buttons centered in the area above the textbox.

// --- drawing ---

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{16, 16, 24, 255})
	if g.mode == modeMenu {
		g.drawMainMenu(screen)
		return
	}
	g.drawBackground(screen)
	for _, e := range g.liveExits() {
		g.drawSpriteAt(screen, e.sprite, 1-progressAt(e.start, e.dur))
	}
	for _, sp := range g.view.Sprites {
		alpha := 1.0
		if a, ok := g.entering[sp.Char]; ok {
			alpha = progressAt(a.start, a.dur)
		}
		g.drawSpriteAt(screen, sp, alpha)
	}
	g.drawDialogue(screen)

	switch {
	case g.view.Ended:
		g.drawCentered(screen, "Selesai - klik atau tekan Spasi untuk keluar", screenH/2)
	case len(g.view.Choices) > 0:
		g.drawChoices(screen)
	}
	if g.mode == modePause {
		g.drawPause(screen)
	}
}

// liveExits drops exit animations that have finished fading, so Draw
// doesn't keep rendering sprites that are long gone. Safe to call every
// frame: it filters g.exiting in place.
func (g *Game) liveExits() []spriteAnim {
	live := g.exiting[:0]
	for _, e := range g.exiting {
		if progressAt(e.start, e.dur) < 1 {
			live = append(live, e)
		}
	}
	g.exiting = live
	return live
}

// drawBackground crossfades from the previous background to the current
// one while g.bg's animation is running, and just draws the current one
// once it's done (or if there never was an animation: progressAt(dur<=0)
// is always 1).
func (g *Game) drawBackground(dst *ebiten.Image) {
	t := progressAt(g.bg.start, g.bg.dur)
	if t < 1 {
		g.drawBackgroundNamed(dst, g.bg.prevName, 1-t)
		g.drawBackgroundNamed(dst, g.view.Background, t)
		return
	}
	g.drawBackgroundNamed(dst, g.view.Background, 1)
}

func (g *Game) drawBackgroundNamed(dst *ebiten.Image, name string, alpha float64) {
	if name == "" || alpha <= 0 {
		return
	}
	img := g.assets.background(name)
	if img == nil {
		bg := tint(name, 255)
		bg.R, bg.G, bg.B = bg.R/3, bg.G/3, bg.B/3
		g.fillRectAlpha(dst, 0, 0, screenW, screenH, bg, alpha)
		g.drawTextAlpha(dst, "[background: "+name+"]", 24, 20, colWhite, alpha)
		return
	}
	b := img.Bounds()
	s := math.Max(screenW/float64(b.Dx()), screenH/float64(b.Dy())) // cover
	var op ebiten.DrawImageOptions
	op.GeoM.Scale(s, s)
	op.GeoM.Translate((screenW-float64(b.Dx())*s)/2, (screenH-float64(b.Dy())*s)/2)
	op.Filter = ebiten.FilterLinear
	op.ColorScale.ScaleAlpha(float32(alpha))
	dst.DrawImage(img, &op)
}

func spriteCenterX(pos string) float64 {
	switch pos {
	case "left":
		return screenW * 0.2
	case "right":
		return screenW * 0.8
	default:
		return screenW * 0.5
	}
}

func (g *Game) drawSpriteAt(dst *ebiten.Image, sp Sprite, alpha float64) {
	if alpha <= 0 {
		return
	}
	cx := spriteCenterX(sp.Pos)
	if img := g.assets.sprite(sp.Char, sp.Expr); img != nil {
		b := img.Bounds()
		s := spriteH / float64(b.Dy())
		var op ebiten.DrawImageOptions
		op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())) // anchor bottom-center
		if sp.Flip {
			op.GeoM.Scale(-1, 1) // cermin horizontal terhadap titik tengah
		}
		op.GeoM.Scale(s, s)
		op.GeoM.Translate(cx, screenH)
		op.Filter = ebiten.FilterLinear
		op.ColorScale.ScaleAlpha(float32(alpha))
		dst.DrawImage(img, &op)
		return
	}
	// placeholder: colored block labeled with character and expression
	const w, h = 260.0, 520.0
	g.fillRectAlpha(dst, cx-w/2, screenH-h, w, h, tint(sp.Char, 235), alpha)
	g.drawTextAlpha(dst, sp.Char, cx-w/2+16, screenH-h+16, colWhite, alpha)
	g.drawTextAlpha(dst, sp.Expr, cx-w/2+16, screenH-h+52, colWhite, alpha)
}

// --- helpers ---

func (g *Game) fillRect(dst *ebiten.Image, x, y, w, h float64, c color.Color) {
	g.fillRectAlpha(dst, x, y, w, h, c, 1)
}

func (g *Game) fillRectAlpha(dst *ebiten.Image, x, y, w, h float64, c color.Color, alpha float64) {
	var op ebiten.DrawImageOptions
	op.GeoM.Scale(w, h)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	op.ColorScale.ScaleAlpha(float32(alpha))
	dst.DrawImage(g.pixel, &op)
}

func (g *Game) drawText(dst *ebiten.Image, s string, x, y float64, c color.Color) {
	g.drawTextAlpha(dst, s, x, y, c, 1)
}

func (g *Game) drawTextAlpha(dst *ebiten.Image, s string, x, y float64, c color.Color, alpha float64) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	op.ColorScale.ScaleAlpha(float32(alpha))
	text.Draw(dst, s, g.face, op)
}

// wrap breaks s into lines no wider than maxW, on spaces. Explicit "\n" is kept.
func wrap(s string, face text.Face, maxW float64) []string {
	if s == "" {
		return nil
	}
	var lines []string
	for _, para := range strings.Split(s, "\n") {
		cur := ""
		for _, word := range strings.Fields(para) {
			try := word
			if cur != "" {
				try = cur + " " + word
			}
			if w, _ := text.Measure(try, face, 0); w > maxW && cur != "" {
				lines = append(lines, cur)
				cur = word
			} else {
				cur = try
			}
		}
		lines = append(lines, cur)
	}
	return lines
}

// tint gives a stable color per name, for placeholders.
func tint(name string, alpha uint8) color.NRGBA {
	h := fnv.New32a()
	h.Write([]byte(name))
	v := h.Sum32()
	return color.NRGBA{
		R: 60 + uint8(v%140),
		G: 60 + uint8((v>>8)%140),
		B: 60 + uint8((v>>16)%140),
		A: alpha,
	}
}
