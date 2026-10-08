// Command ango-launcher is a small graphical launcher for Ango stories,
// in the spirit of the Ren'Py launcher: it lists the projects found in one
// or more folders and runs the `ango` binary on them (play, check,
// auto-test), showing the output in a log panel.
//
// Project folders are not tied to one fixed place. They are resolved in
// this order (first one that is set wins):
//
//  1. folders given on the command line
//  2. the ANGO_PROJECTS environment variable (":"-separated on Linux and
//     macOS, ";"-separated on Windows)
//  3. the roots saved in the config file (launcher.json under the user config directory:
//     ~/.config/ango on Linux, %AppData%\ango on Windows)
//  4. a "projects" folder next to the launcher binary
//
// "Tambah Folder" adds a folder at runtime and saves it to the config file.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	winW, winH = 1100, 680
	faceSize   = 18
	lineH      = 24
	itemH      = 52

	infoX            = 392
	btnW, btnH       = 250, 42
	btnGapX, btnGapY = 16, 10
	btnTop           = 170
)

var (
	colBG    = color.NRGBA{20, 20, 32, 255}
	colPanel = color.NRGBA{30, 30, 50, 255}
	colItem  = color.NRGBA{44, 44, 74, 255}
	colSel   = color.NRGBA{70, 70, 140, 255}
	colBtn   = color.NRGBA{42, 42, 80, 255}
	colBtnHi = color.NRGBA{70, 70, 140, 255}
	colOff   = color.NRGBA{34, 34, 48, 255}
	colText  = color.NRGBA{255, 255, 255, 255}
	colDim   = color.NRGBA{150, 150, 175, 255}
	colOffTx = color.NRGBA{100, 100, 120, 255}

	listRect = image.Rect(24, 84, 364, 656)
	logRect  = image.Rect(392, 330, 1076, 656)
)

// ============================================================
// Config and folder resolution
// ============================================================

type config struct {
	Roots []string `json:"roots,omitempty"`
	Ango  string   `json:"ango,omitempty"`
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ango", "launcher.json"), nil
}

func loadConfig() (config, error) {
	var c config
	p, err := configPath()
	if err != nil {
		return c, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return config{}, fmt.Errorf("%s: %w", p, err)
	}
	return c, nil
}

func saveConfig(c config) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(data, '\n'), 0o644)
}

func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}

func cleanList(in []string) []string {
	var out []string
	for _, p := range in {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, expandHome(p))
		}
	}
	return out
}

// resolveRoots picks the project folders and says where they came from.
func resolveRoots(args []string, env string, cfg config, fallback string) (roots []string, source string) {
	switch {
	case len(cleanList(args)) > 0:
		return cleanList(args), "argumen"
	case len(cleanList(filepath.SplitList(env))) > 0:
		return cleanList(filepath.SplitList(env)), "ANGO_PROJECTS"
	case len(cleanList(cfg.Roots)) > 0:
		return cleanList(cfg.Roots), "konfigurasi"
	}
	return []string{fallback}, "bawaan"
}

func defaultRoot() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exe), "projects")
	}
	return "projects"
}

func summarizeRoots(roots []string) string {
	switch len(roots) {
	case 0:
		return "(tidak ada)"
	case 1:
		return roots[0]
	}
	return fmt.Sprintf("%s (+%d lainnya)", roots[0], len(roots)-1)
}

// ============================================================
// Projects
// ============================================================

type project struct {
	name  string
	dir   string // absolute
	root  string // the root folder this project was found through
	files int    // number of .ango files directly inside
}

func isDirEntry(parent string, e os.DirEntry) bool {
	if e.IsDir() {
		return true
	}
	if e.Type()&fs.ModeSymlink != 0 {
		st, err := os.Stat(filepath.Join(parent, e.Name()))
		return err == nil && st.IsDir()
	}
	return false
}

func countAngo(dir string) int {
	es, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range es {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".ango") {
			n++
		}
	}
	return n
}

// scanRoots lists the projects reachable from roots. A root that itself
// contains .ango files is a project; so is each of its sub-folders that
// does. Duplicates (same absolute folder) are listed once. Unreadable
// roots are reported in errs and skipped.
func scanRoots(roots []string) (out []project, errs []error) {
	seen := map[string]bool{}
	add := func(p project) {
		if !seen[p.dir] {
			seen[p.dir] = true
			out = append(out, p)
		}
	}
	for _, root := range roots {
		abs, err := filepath.Abs(root)
		if err != nil {
			abs = root
		}
		entries, err := os.ReadDir(abs)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if n := countAngo(abs); n > 0 {
			add(project{name: filepath.Base(abs), dir: abs, root: abs, files: n})
		}
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".") || !isDirEntry(abs, e) {
				continue
			}
			dir := filepath.Join(abs, e.Name())
			if n := countAngo(dir); n > 0 {
				add(project{name: e.Name(), dir: dir, root: abs, files: n})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := strings.ToLower(out[i].name), strings.ToLower(out[j].name)
		if a != b {
			return a < b
		}
		return out[i].dir < out[j].dir
	})
	return out, errs
}

// findAngo locates the ango binary: an explicit path, else next to this
// launcher, else on PATH.
func findAngo(explicit string) (string, error) {
	if explicit != "" {
		explicit = expandHome(explicit)
		if st, err := os.Stat(explicit); err != nil || st.IsDir() {
			return "", fmt.Errorf("binary ango tidak ditemukan di %q", explicit)
		}
		return explicit, nil
	}
	if exe, err := os.Executable(); err == nil {
		cand := filepath.Join(filepath.Dir(exe), angoBinaryName(runtime.GOOS))
		if st, err := os.Stat(cand); err == nil && !st.IsDir() {
			return cand, nil
		}
	}
	if p, err := exec.LookPath(angoBinaryName(runtime.GOOS)); err == nil {
		return p, nil
	}
	return "", errors.New("binary ango tidak ditemukan (taruh di samping launcher, atau pakai -ango <path> / ANGO_BIN)")
}

// ============================================================
// Launcher state
// ============================================================

type button struct {
	label   string
	enabled bool
	action  func()
}

type launcher struct {
	face  *text.GoTextFace
	pixel *ebiten.Image

	roots    []string
	source   string // where roots came from
	fallback string
	cfg      config
	ango     string

	projects []project
	sel      int
	scroll   int // first visible project index
	logOff   int // lines scrolled up from the newest log line

	lastClickIdx  int
	lastClickTime time.Time

	pickCh  chan string
	picking bool

	mu      sync.Mutex
	lines   []string
	running int
}

func newLauncher(roots []string, source, fallback string, cfg config) (*launcher, error) {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		return nil, fmt.Errorf("load font: %w", err)
	}
	l := &launcher{
		face:         &text.GoTextFace{Source: src, Size: faceSize},
		pixel:        ebiten.NewImage(1, 1),
		roots:        roots,
		source:       source,
		fallback:     fallback,
		cfg:          cfg,
		lastClickIdx: -1,
		pickCh:       make(chan string, 1),
	}
	l.pixel.Fill(color.White)
	return l, nil
}

func (l *launcher) logf(format string, args ...any) {
	s := fmt.Sprintf(format, args...)
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r", ""), "\t", "    ")
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, strings.Split(s, "\n")...)
	if len(l.lines) > 2000 {
		l.lines = append([]string(nil), l.lines[500:]...)
	}
}

func (l *launcher) snapshot() (lines []string, running int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.lines...), l.running
}

func (l *launcher) addRunning(d int) {
	l.mu.Lock()
	l.running += d
	l.mu.Unlock()
}

func (l *launcher) reload() {
	ps, errs := scanRoots(l.roots)
	for _, err := range errs {
		l.logf("tidak bisa membaca folder: %v", err)
	}
	l.projects = ps
	if l.sel >= len(ps) {
		l.sel = len(ps) - 1
	}
	if l.sel < 0 {
		l.sel = 0
	}
	l.ensureVisible()
}

func (l *launcher) selected() (project, bool) {
	if l.sel < 0 || l.sel >= len(l.projects) {
		return project{}, false
	}
	return l.projects[l.sel], true
}

func (l *launcher) visibleItems() int { return listRect.Dy() / itemH }

func (l *launcher) ensureVisible() {
	v := l.visibleItems()
	if l.sel < l.scroll {
		l.scroll = l.sel
	}
	if l.sel >= l.scroll+v {
		l.scroll = l.sel - v + 1
	}
	if l.scroll < 0 {
		l.scroll = 0
	}
}

// ============================================================
// Actions
// ============================================================

// start runs the ango binary with args, streaming its output to the log.
func (l *launcher) start(title string, args ...string) {
	cmd := exec.Command(l.ango, args...)
	hideWindow(cmd)
	pr, pw := io.Pipe()
	cmd.Stdout, cmd.Stderr = pw, pw
	l.logf("> %s: %s %s", title, filepath.Base(l.ango), strings.Join(args, " "))
	if err := cmd.Start(); err != nil {
		l.logf("gagal memulai: %v", err)
		return
	}
	l.addRunning(1)

	done := make(chan struct{})
	go func() {
		sc := bufio.NewScanner(pr)
		for sc.Scan() {
			l.logf("%s", sc.Text())
		}
		close(done)
	}()
	go func() {
		err := cmd.Wait()
		pw.Close()
		<-done
		if err != nil {
			l.logf("< %s selesai dengan error: %v", title, err)
		} else {
			l.logf("< %s selesai", title)
		}
		l.addRunning(-1)
	}()
}

func (l *launcher) actLaunch() {
	if p, ok := l.selected(); ok && l.ango != "" {
		l.start("Jalankan "+p.name, "-window", p.dir)
	}
}

func (l *launcher) actCheck() {
	if p, ok := l.selected(); ok && l.ango != "" {
		l.start("Periksa "+p.name, "-check", p.dir)
	}
}

func (l *launcher) actAuto() {
	if p, ok := l.selected(); ok && l.ango != "" {
		l.start("Uji otomatis "+p.name, "-auto", p.dir)
	}
}

func (l *launcher) actOpen() {
	p, ok := l.selected()
	if !ok {
		return
	}
	if err := openFolder(p.dir); err != nil {
		l.logf("membuka folder gagal: %v", err)
	}
}

func (l *launcher) actRefresh() {
	name, dir := "", ""
	if p, ok := l.selected(); ok {
		name, dir = p.name, p.dir
	}
	l.reload()
	for i, p := range l.projects {
		if p.name == name && p.dir == dir {
			l.sel = i
		}
	}
	l.ensureVisible()
	l.logf("Daftar proyek dimuat ulang (%d proyek)", len(l.projects))
}

// actAddFolder opens a native folder chooser without blocking the UI; the
// result arrives on pickCh and is handled in Update.
func (l *launcher) actAddFolder() {
	if l.picking {
		return
	}
	l.picking = true
	l.logf("memilih folder...")
	go func() {
		p, err := pickFolder()
		if errors.Is(err, errNoPicker) {
			hint := "jalankan: ango-launcher <folder>"
			if cp, cerr := configPath(); cerr == nil {
				hint += ", atau tambahkan folder ke " + cp
			}
			l.logf("%v; %s", err, hint)
		} else if err != nil {
			l.logf("pemilih folder gagal: %v", err)
		}
		l.pickCh <- p // "" = batal atau gagal
	}()
}

func sameDir(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

// addRoot adds a project folder for this session and saves it to the config.
func (l *launcher) addRoot(path string) {
	abs, err := filepath.Abs(expandHome(path))
	if err != nil {
		abs = path
	}
	if st, err := os.Stat(abs); err != nil || !st.IsDir() {
		l.logf("bukan folder: %s", path)
		return
	}
	for _, r := range l.roots {
		if abs2, err := filepath.Abs(r); err == nil && sameDir(abs2, abs) {
			l.logf("folder sudah ada di daftar: %s", abs)
			return
		}
	}
	l.roots = append(l.roots, abs)

	// Keep the folder list the user had before (the default one) when it
	// is the first time the config gets written.
	if len(l.cfg.Roots) == 0 && l.source == "bawaan" {
		if st, err := os.Stat(l.fallback); err == nil && st.IsDir() {
			l.cfg.Roots = append(l.cfg.Roots, l.fallback)
		}
	}
	l.cfg.Roots = append(l.cfg.Roots, abs)
	if err := saveConfig(l.cfg); err != nil {
		l.logf("folder ditambahkan, tapi konfigurasi gagal disimpan: %v", err)
	} else {
		l.logf("folder ditambahkan dan disimpan: %s", abs)
	}

	l.reload()
	for i, p := range l.projects {
		if sameDir(p.root, abs) {
			l.sel = i
			break
		}
	}
	l.ensureVisible()
	if len(l.projects) == 0 {
		l.logf("(belum ada folder berisi .ango di %s)", abs)
	}
}

func (l *launcher) buttons() []button {
	hasProj := len(l.projects) > 0
	canRun := hasProj && l.ango != ""
	return []button{
		{"Jalankan", canRun, l.actLaunch},
		{"Periksa Skrip", canRun, l.actCheck},
		{"Uji Otomatis", canRun, l.actAuto},
		{"Buka Folder", hasProj, l.actOpen},
		{"Tambah Folder", !l.picking, l.actAddFolder},
		{"Segarkan (F5)", true, l.actRefresh},
	}
}

func btnRect(i int) image.Rectangle {
	col, row := i/3, i%3
	x := infoX + col*(btnW+btnGapX)
	y := btnTop + row*(btnH+btnGapY)
	return image.Rect(x, y, x+btnW, y+btnH)
}

// ============================================================
// ebiten.Game
// ============================================================

func (l *launcher) Layout(outW, outH int) (int, int) { return winW, winH }

func pressedPoints() []image.Point {
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

func (l *launcher) Update() error {
	select {
	case p := <-l.pickCh:
		l.picking = false
		if p != "" {
			l.addRoot(p)
		}
	default:
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	mx, my := ebiten.CursorPosition()
	if _, wy := ebiten.Wheel(); wy != 0 {
		switch pt := image.Pt(mx, my); {
		case pt.In(listRect):
			l.scroll -= int(wy)
			maxScroll := len(l.projects) - l.visibleItems()
			if maxScroll < 0 {
				maxScroll = 0
			}
			if l.scroll > maxScroll {
				l.scroll = maxScroll
			}
			if l.scroll < 0 {
				l.scroll = 0
			}
		case pt.In(logRect):
			l.logOff += int(wy)
		}
	}

	if n := len(l.projects); n > 0 {
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) && l.sel < n-1 {
			l.sel++
			l.ensureVisible()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) && l.sel > 0 {
			l.sel--
			l.ensureVisible()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			l.actLaunch()
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		l.actRefresh()
	}

	btns := l.buttons()
	for _, p := range pressedPoints() {
		if p.In(listRect) {
			idx := l.scroll + (p.Y-listRect.Min.Y)/itemH
			if idx >= 0 && idx < len(l.projects) {
				l.sel = idx
				// klik dua kali pada proyek yang sama = jalankan
				if idx == l.lastClickIdx && time.Since(l.lastClickTime) < 400*time.Millisecond {
					l.actLaunch()
					l.lastClickIdx = -1
				} else {
					l.lastClickIdx, l.lastClickTime = idx, time.Now()
				}
			}
			continue
		}
		for i, b := range btns {
			if b.enabled && p.In(btnRect(i)) {
				b.action()
				break
			}
		}
	}
	return nil
}

func (l *launcher) Draw(dst *ebiten.Image) {
	dst.Fill(colBG)
	lines, running := l.snapshot()

	// header
	l.drawTextScaled(dst, "Ango Launcher", 24, 12, 1.4, colText)
	l.drawText(dst, l.fit("Folder proyek: "+summarizeRoots(l.roots), 800), 24, 52, colDim)
	if running > 0 {
		l.drawText(dst, fmt.Sprintf("Proses berjalan: %d", running), 900, 24, colText)
	}

	// project list
	l.fillRect(dst, listRect, colPanel)
	if len(l.projects) == 0 {
		x := float64(listRect.Min.X) + 14
		y := float64(listRect.Min.Y) + 14
		l.drawText(dst, "Belum ada proyek.", x, y, colText)
		l.drawText(dst, "Klik 'Tambah Folder' untuk memilih", x, y+lineH*2, colDim)
		l.drawText(dst, "folder yang berisi game (.ango).", x, y+lineH*3, colDim)
	}
	mx, my := ebiten.CursorPosition()
	for row := 0; row < l.visibleItems(); row++ {
		idx := l.scroll + row
		if idx >= len(l.projects) {
			break
		}
		p := l.projects[idx]
		r := image.Rect(listRect.Min.X, listRect.Min.Y+row*itemH, listRect.Max.X, listRect.Min.Y+(row+1)*itemH)
		switch {
		case idx == l.sel:
			l.fillRect(dst, r, colSel)
		case image.Pt(mx, my).In(r):
			l.fillRect(dst, r, colItem)
		}
		maxW := float64(r.Dx() - 28)
		l.drawText(dst, l.fit(p.name, maxW), float64(r.Min.X)+14, float64(r.Min.Y)+5, colText)
		l.drawTextScaled(dst, l.fit(p.root, maxW/0.8), float64(r.Min.X)+14, float64(r.Min.Y)+29, 0.8, colDim)
	}

	// info
	if p, ok := l.selected(); ok {
		l.drawTextScaled(dst, l.fit(p.name, 600/1.5), infoX, 80, 1.5, colText)
		l.drawText(dst, l.fit(p.dir, 680), infoX, 124, colDim)
		l.drawText(dst, fmt.Sprintf("%d berkas .ango", p.files), infoX, 146, colDim)
	}
	if l.ango == "" {
		l.drawText(dst, "Binary ango tidak ditemukan; tombol jalan dinonaktifkan.", infoX, 300, colDim)
	}

	// buttons
	for i, b := range l.buttons() {
		r := btnRect(i)
		c, tc := colBtn, colText
		switch {
		case !b.enabled:
			c, tc = colOff, colOffTx
		case image.Pt(mx, my).In(r):
			c = colBtnHi
		}
		l.fillRect(dst, r, c)
		tw, _ := text.Measure(b.label, l.face, 0)
		l.drawText(dst, b.label, float64(r.Min.X)+(float64(r.Dx())-tw)/2, float64(r.Min.Y)+(btnH-lineH)/2+2, tc)
	}

	// log
	l.fillRect(dst, logRect, colPanel)
	visible := (logRect.Dy() - 16) / lineH
	maxOff := len(lines) - visible
	if maxOff < 0 {
		maxOff = 0
	}
	if l.logOff > maxOff {
		l.logOff = maxOff
	}
	if l.logOff < 0 {
		l.logOff = 0
	}
	end := len(lines) - l.logOff
	start := end - visible
	if start < 0 {
		start = 0
	}
	for i := start; i < end; i++ {
		y := float64(logRect.Min.Y) + 8 + float64(i-start)*lineH
		l.drawText(dst, l.fit(lines[i], float64(logRect.Dx()-24)), float64(logRect.Min.X)+12, y, colText)
	}
}

// ---- drawing helpers ----

func (l *launcher) fillRect(dst *ebiten.Image, r image.Rectangle, c color.Color) {
	var op ebiten.DrawImageOptions
	op.GeoM.Scale(float64(r.Dx()), float64(r.Dy()))
	op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
	op.ColorScale.ScaleWithColor(c)
	dst.DrawImage(l.pixel, &op)
}

func (l *launcher) drawText(dst *ebiten.Image, s string, x, y float64, c color.Color) {
	l.drawTextScaled(dst, s, x, y, 1, c)
}

func (l *launcher) drawTextScaled(dst *ebiten.Image, s string, x, y, scale float64, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, l.face, op)
}

// fit shortens s with "..." so it is at most maxW pixels wide.
func (l *launcher) fit(s string, maxW float64) string {
	if w, _ := text.Measure(s, l.face, 0); w <= maxW {
		return s
	}
	rs := []rune(s)
	for len(rs) > 0 {
		rs = rs[:len(rs)-1]
		if w, _ := text.Measure(string(rs)+"...", l.face, 0); w <= maxW {
			return string(rs) + "..."
		}
	}
	return ""
}

// ============================================================
// main
// ============================================================

func main() {
	angoFlag := flag.String("ango", "", "path ke binary ango (juga: env ANGO_BIN; bawaan: di samping launcher, lalu PATH)")
	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintf(out, "usage: ango-launcher [-ango path] [folder-proyek ...]\n\n")
		fmt.Fprintf(out, "Folder proyek diambil dari (yang pertama ada): argumen, env ANGO_PROJECTS,\n")
		fmt.Fprintf(out, "konfigurasi (launcher.json di folder konfigurasi pengguna), lalu folder 'projects' di samping launcher.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	cfg, cfgErr := loadConfig()
	roots, source := resolveRoots(flag.Args(), os.Getenv("ANGO_PROJECTS"), cfg, defaultRoot())

	l, err := newLauncher(roots, source, defaultRoot(), cfg)
	if err != nil {
		log.Fatal(err)
	}
	if cfgErr != nil {
		l.logf("konfigurasi diabaikan: %v", cfgErr)
	}

	for _, c := range []string{*angoFlag, os.Getenv("ANGO_BIN"), cfg.Ango} {
		if c == "" {
			continue
		}
		bin, err := findAngo(c)
		if err != nil {
			l.logf("%v", err)
			continue
		}
		l.ango = bin
		break
	}
	if l.ango == "" {
		bin, err := findAngo("")
		if err != nil {
			l.logf("%v", err)
		} else {
			l.ango = bin
		}
	}
	if l.ango != "" {
		l.logf("ango: %s", l.ango)
	}

	l.reload()
	l.logf("%d proyek ditemukan (sumber folder: %s)", len(l.projects), source)
	if len(l.projects) == 0 {
		l.logf("tambahkan folder lewat 'Tambah Folder', atau jalankan: ango-launcher <folder>")
	}

	ebiten.SetWindowTitle("Ango Launcher")
	ebiten.SetWindowSize(winW*3/4, winH*3/4)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(l); err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatal(err)
	}
}
