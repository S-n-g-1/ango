package ebitengine

import (
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"io/fs"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ============================================================
// Theme: assets/gui/theme.json (semua field opsional)
// ============================================================

type rectFile struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type colorsFile struct {
	Text        string `json:"text"`
	Textbox     string `json:"textbox"`
	Plate       string `json:"plate"`
	Button      string `json:"button"`
	ButtonHover string `json:"button_hover"`
}

type plateFile struct {
	OffsetX int     `json:"offset_x"`
	Height  int     `json:"height"`
	Inset   float64 `json:"inset"`
}

type buttonFile struct {
	W    int     `json:"w"`
	H    int     `json:"h"`
	Gap  int     `json:"gap"`
	PadX float64 `json:"pad_x"`
}

type menuFile struct {
	Enabled    bool    `json:"enabled"`
	Title      string  `json:"title"`
	Start      string  `json:"start"`
	Resume     string  `json:"resume"`
	Quit       string  `json:"quit"`
	Pause      string  `json:"pause"`
	Top        int     `json:"top"`
	TitleScale float64 `json:"title_scale"`
}

type themeFile struct {
	FontSize   float64    `json:"font_size"`
	LineHeight float64    `json:"line_height"`
	MaxLines   int        `json:"max_lines"`
	Colors     colorsFile `json:"colors"`
	Textbox    rectFile   `json:"textbox"`
	TextPadX   float64    `json:"text_pad_x"`
	TextPadY   float64    `json:"text_pad_y"`
	Plate      plateFile  `json:"plate"`
	Button     buttonFile `json:"button"`
	Menu       menuFile   `json:"menu"`
}

// defaultThemeFile reproduces the look the backend had before themes
// existed, so a story without assets/gui/ renders exactly as before.
func defaultThemeFile() themeFile {
	return themeFile{
		FontSize:   26,
		LineHeight: 34,
		MaxLines:   4,
		Colors: colorsFile{
			Text:        "#FFFFFF",
			Textbox:     "#000000BE",
			Plate:       "#1E1E46E6",
			Button:      "#141428DC",
			ButtonHover: "#46468CF0",
		},
		Textbox:  rectFile{X: 40, Y: 520, W: 1200, H: 170},
		TextPadX: 28,
		TextPadY: 20,
		Plate:    plateFile{OffsetX: 20, Height: 44, Inset: 20},
		Button:   buttonFile{W: 720, H: 56, Gap: 16, PadX: 24},
		Menu: menuFile{
			Start:      "Mulai",
			Resume:     "Lanjut",
			Quit:       "Keluar",
			Pause:      "Jeda",
			Top:        360,
			TitleScale: 2.2,
		},
	}
}

// theme is the resolved form of themeFile used by the renderer.
type theme struct {
	fontSize, lineH float64
	maxLines        int

	text, textbox, plate, btn, btnHi color.NRGBA

	box        image.Rectangle
	padX, padY float64
	plateOffX  int
	plateH     int
	plateInset float64
	btnW, btnH int
	btnGap     int
	btnPadX    float64
	menu       menuFile
}

// loadTheme reads gui/theme.json from fsys. A missing file (or nil fsys)
// gives the defaults; a broken file is logged and also gives the defaults.
func loadTheme(fsys fs.FS) theme {
	f := defaultThemeFile()
	if fsys != nil {
		data, err := fs.ReadFile(fsys, "gui/theme.json")
		switch {
		case err == nil:
			if err := json.Unmarshal(data, &f); err != nil {
				log.Printf("ebitengine: gui/theme.json: %v (using defaults)", err)
				f = defaultThemeFile()
			}
		case !errors.Is(err, fs.ErrNotExist):
			log.Printf("ebitengine: gui/theme.json: %v (using defaults)", err)
		}
	}
	return f.resolve()
}

func (f themeFile) resolve() theme {
	d := defaultThemeFile()
	if f.FontSize <= 0 {
		f.FontSize = d.FontSize
	}
	if f.LineHeight <= 0 {
		f.LineHeight = d.LineHeight
	}
	if f.MaxLines <= 0 {
		f.MaxLines = d.MaxLines
	}
	if f.Textbox.W <= 0 || f.Textbox.H <= 0 {
		f.Textbox = d.Textbox
	}
	if f.Button.W <= 0 || f.Button.H <= 0 {
		f.Button.W, f.Button.H = d.Button.W, d.Button.H
	}
	if f.Menu.TitleScale <= 0 {
		f.Menu.TitleScale = d.Menu.TitleScale
	}
	if f.Menu.Top <= 0 {
		f.Menu.Top = d.Menu.Top
	}
	for _, p := range []struct {
		dst *string
		def string
	}{
		{&f.Menu.Start, d.Menu.Start},
		{&f.Menu.Resume, d.Menu.Resume},
		{&f.Menu.Quit, d.Menu.Quit},
		{&f.Menu.Pause, d.Menu.Pause},
	} {
		if *p.dst == "" {
			*p.dst = p.def
		}
	}

	return theme{
		fontSize:   f.FontSize,
		lineH:      f.LineHeight,
		maxLines:   f.MaxLines,
		text:       parseColor("colors.text", f.Colors.Text, d.Colors.Text),
		textbox:    parseColor("colors.textbox", f.Colors.Textbox, d.Colors.Textbox),
		plate:      parseColor("colors.plate", f.Colors.Plate, d.Colors.Plate),
		btn:        parseColor("colors.button", f.Colors.Button, d.Colors.Button),
		btnHi:      parseColor("colors.button_hover", f.Colors.ButtonHover, d.Colors.ButtonHover),
		box:        image.Rect(f.Textbox.X, f.Textbox.Y, f.Textbox.X+f.Textbox.W, f.Textbox.Y+f.Textbox.H),
		padX:       f.TextPadX,
		padY:       f.TextPadY,
		plateOffX:  f.Plate.OffsetX,
		plateH:     f.Plate.Height,
		plateInset: f.Plate.Inset,
		btnW:       f.Button.W,
		btnH:       f.Button.H,
		btnGap:     f.Button.Gap,
		btnPadX:    f.Button.PadX,
		menu:       f.Menu,
	}
}

// parseColor accepts "#RRGGBB" or "#RRGGBBAA". Empty means "use the
// default" silently; anything else malformed is logged.
func parseColor(name, s, def string) color.NRGBA {
	if s != "" {
		if c, ok := tryColor(s); ok {
			return c
		}
		log.Printf("ebitengine: gui/theme.json: bad color %q for %s (using default)", s, name)
	}
	c, _ := tryColor(def)
	return c
}

func tryColor(s string) (color.NRGBA, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 && len(s) != 8 {
		return color.NRGBA{}, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return color.NRGBA{}, false
	}
	if len(s) == 6 {
		return color.NRGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}, true
	}
	return color.NRGBA{uint8(v >> 24), uint8(v >> 16), uint8(v >> 8), uint8(v)}, true
}

// ============================================================
// Menu utama dan menu jeda (opt-in lewat "menu.enabled": true)
// ============================================================

type uiMode int

const (
	modePlaying uiMode = iota // nilai nol: perilaku lama, langsung masuk cerita
	modeMenu
	modePause
)

type menuItem struct {
	label  string
	action func(g *Game) error
}

func quitAction(*Game) error { return ebiten.Termination }

func resumeAction(g *Game) error {
	g.mode = modePlaying
	return nil
}

// initMenu starts at the main menu when the theme enables it.
func (g *Game) initMenu(defaultTitle string) {
	if !g.th.menu.Enabled {
		return
	}
	if g.th.menu.Title == "" {
		g.th.menu.Title = defaultTitle
	}
	g.openMain()
}

func (g *Game) openMain() {
	g.mode, g.sel = modeMenu, 0
	g.items = []menuItem{
		{g.th.menu.Start, resumeAction},
		{g.th.menu.Quit, quitAction},
	}
}

func (g *Game) openPause() {
	g.mode, g.sel = modePause, 0
	g.items = []menuItem{
		{g.th.menu.Resume, resumeAction},
		{g.th.menu.Quit, quitAction},
	}
}

func (g *Game) menuRects() []image.Rectangle {
	n := len(g.items)
	top := g.th.menu.Top
	if g.mode == modePause {
		total := n*g.th.btnH + (n-1)*g.th.btnGap
		top = (screenH-total)/2 + 30
	}
	x := (screenW - g.th.btnW) / 2
	rects := make([]image.Rectangle, n)
	for i := range rects {
		y := top + i*(g.th.btnH+g.th.btnGap)
		rects[i] = image.Rect(x, y, x+g.th.btnW, y+g.th.btnH)
	}
	return rects
}

func (g *Game) updateMenu() error {
	rects := g.menuRects()

	x, y := ebiten.CursorPosition()
	if pt := image.Pt(x, y); pt != g.lastCursor {
		g.lastCursor = pt
		for i, r := range rects {
			if pt.In(r) {
				g.sel = i
			}
		}
	}

	n := len(g.items)
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		g.sel = (g.sel + 1) % n
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		g.sel = (g.sel + n - 1) % n
	}
	if g.mode == modePause && inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.mode = modePlaying
		return nil
	}
	for _, p := range justPressedPoints() {
		for i, r := range rects {
			if p.In(r) {
				g.sel = i
				return g.items[i].action(g)
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		return g.items[g.sel].action(g)
	}
	return nil
}

func (g *Game) drawMainMenu(dst *ebiten.Image) {
	dst.Fill(color.NRGBA{16, 16, 24, 255})
	if bg := g.assets.gui("menu_bg"); bg != nil {
		g.drawCover(dst, bg)
	}
	if t := g.assets.gui("title"); t != nil {
		b := t.Bounds()
		s := 1.0
		if maxW := screenW * 0.8; float64(b.Dx()) > maxW {
			s = maxW / float64(b.Dx())
		}
		var op ebiten.DrawImageOptions
		op.GeoM.Scale(s, s)
		op.GeoM.Translate((screenW-float64(b.Dx())*s)/2, 60)
		op.Filter = ebiten.FilterLinear
		dst.DrawImage(t, &op)
	} else {
		g.drawTextCenteredScaled(dst, g.th.menu.Title, screenW/2, 150, g.th.menu.TitleScale, g.th.text)
	}
	g.drawMenuButtons(dst)
}

func (g *Game) drawPause(dst *ebiten.Image) {
	g.fillRectAlpha(dst, 0, 0, screenW, screenH, color.Black, 0.6)
	g.drawTextCenteredScaled(dst, g.th.menu.Pause, screenW/2, 200, 1.6, g.th.text)
	g.drawMenuButtons(dst)
}

func (g *Game) drawMenuButtons(dst *ebiten.Image) {
	for i, r := range g.menuRects() {
		g.drawButton(dst, r, g.items[i].label, i == g.sel, true)
	}
}

// ============================================================
// Gambar GUI bersama (dialog, pilihan, menu)
// ============================================================

// drawButton draws gui/button(.ext) if present (gui/button_hover when hot),
// otherwise a flat rectangle in the theme colors.
func (g *Game) drawButton(dst *ebiten.Image, r image.Rectangle, label string, hot, center bool) {
	img := g.assets.gui("button")
	hoverImg := g.assets.gui("button_hover")
	switch {
	case hot && hoverImg != nil:
		g.drawImageRect(dst, hoverImg, r, 1)
	case img != nil:
		g.drawImageRect(dst, img, r, 1)
		if hot {
			g.fillRectAlpha(dst, float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()), g.th.btnHi, 0.35)
		}
	default:
		c := g.th.btn
		if hot {
			c = g.th.btnHi
		}
		g.fillRect(dst, float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()), c)
	}

	tx := float64(r.Min.X) + g.th.btnPadX
	if center {
		tw, _ := text.Measure(label, g.face, 0)
		tx = float64(r.Min.X) + (float64(r.Dx())-tw)/2
	}
	g.drawText(dst, label, tx, float64(r.Min.Y)+(float64(r.Dy())-g.th.lineH)/2+2, g.th.text)
}

// drawImageRect stretches img to fill r.
func (g *Game) drawImageRect(dst, img *ebiten.Image, r image.Rectangle, alpha float64) {
	b := img.Bounds()
	var op ebiten.DrawImageOptions
	op.GeoM.Scale(float64(r.Dx())/float64(b.Dx()), float64(r.Dy())/float64(b.Dy()))
	op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
	op.Filter = ebiten.FilterLinear
	op.ColorScale.ScaleAlpha(float32(alpha))
	dst.DrawImage(img, &op)
}

// drawCover scales img to cover the whole screen, keeping aspect ratio.
func (g *Game) drawCover(dst, img *ebiten.Image) {
	b := img.Bounds()
	s := math.Max(screenW/float64(b.Dx()), screenH/float64(b.Dy()))
	var op ebiten.DrawImageOptions
	op.GeoM.Scale(s, s)
	op.GeoM.Translate((screenW-float64(b.Dx())*s)/2, (screenH-float64(b.Dy())*s)/2)
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(img, &op)
}

func (g *Game) drawTextCenteredScaled(dst *ebiten.Image, s string, cx, y, scale float64, c color.Color) {
	tw, _ := text.Measure(s, g.face, 0)
	op := &text.DrawOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(cx-tw*scale/2, y)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, g.face, op)
}

// ============================================================
// Dialog dan pilihan (menggantikan versi di game.go)
// ============================================================

func (g *Game) drawDialogue(dst *ebiten.Image) {
	if g.view.Text == "" && g.view.Speaker == "" {
		return
	}
	th := g.th
	box := th.box
	if img := g.assets.gui("textbox"); img != nil {
		g.drawImageRect(dst, img, box, 1)
	} else {
		g.fillRect(dst, float64(box.Min.X), float64(box.Min.Y), float64(box.Dx()), float64(box.Dy()), th.textbox)
	}

	if sp := g.view.Speaker; sp != "" {
		tw, _ := text.Measure(sp, g.face, 0)
		pw := int(tw) + 2*int(th.plateInset)
		pr := image.Rect(box.Min.X+th.plateOffX, box.Min.Y-th.plateH, box.Min.X+th.plateOffX+pw, box.Min.Y)
		if img := g.assets.gui("namebox"); img != nil {
			g.drawImageRect(dst, img, pr, 1)
		} else {
			g.fillRect(dst, float64(pr.Min.X), float64(pr.Min.Y), float64(pr.Dx()), float64(pr.Dy()), th.plate)
		}
		g.drawText(dst, sp, float64(pr.Min.X)+th.plateInset, float64(pr.Min.Y)+5, th.text)
	}

	for i, ln := range g.lines {
		if i >= th.maxLines {
			break
		}
		g.drawText(dst, ln, float64(box.Min.X)+th.padX, float64(box.Min.Y)+th.padY+float64(i)*th.lineH, th.text)
	}
}

// choiceRects lays out n buttons centered in the area above the textbox.
func (g *Game) choiceRects(n int) []image.Rectangle {
	th := g.th
	total := n*th.btnH + (n-1)*th.btnGap
	top := (th.box.Min.Y - total) / 2
	if top < 8 {
		top = 8
	}
	x := (screenW - th.btnW) / 2
	rects := make([]image.Rectangle, n)
	for i := range rects {
		y := top + i*(th.btnH+th.btnGap)
		rects[i] = image.Rect(x, y, x+th.btnW, y+th.btnH)
	}
	return rects
}

func (g *Game) drawChoices(dst *ebiten.Image) {
	mx, my := ebiten.CursorPosition()
	for i, r := range g.choiceRects(len(g.view.Choices)) {
		g.drawButton(dst, r, g.view.Choices[i], image.Pt(mx, my).In(r), false)
	}
}

func (g *Game) drawCentered(dst *ebiten.Image, s string, y float64) {
	tw, _ := text.Measure(s, g.face, 0)
	g.fillRect(dst, (screenW-tw)/2-20, y-10, tw+40, g.th.lineH+20, g.th.textbox)
	g.drawText(dst, s, (screenW-tw)/2, y, g.th.text)
}
