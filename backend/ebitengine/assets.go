package ebitengine

import (
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // register decoders
	_ "image/png"
	"io/fs"
	"log"
	"math"
	"path"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	_ "golang.org/x/image/webp" // register decoder (image.Decode picks it up by signature)
)

// DefaultExtensions is the order candidate extensions are tried in when a
// script names an asset without one, e.g. `scene "kelas"`. Earlier entries
// win when several files share a base name. WebP is tried first because it
// is the lightest on disk; SVG is tried last because rasterizing it costs
// more CPU than decoding a raster image, even though the file itself is
// small.
var DefaultExtensions = []string{".webp", ".png", ".jpg", ".jpeg", ".svg"}

// Decoder turns the bytes at path p (opened from fsys) into an image.Image.
// Register one with assets.RegisterDecoder to support a format this file
// does not know about (e.g. a future ".avif" or a custom sprite format)
// without editing this file again.
type Decoder func(fsys fs.FS, p string) (image.Image, error)

// assets loads images from an fs.FS rooted at the assets directory
// (backgrounds/, characters/, audio/). A nil result means "not found" and
// the caller draws a placeholder instead, so scripts run without art.
type assets struct {
	fsys  fs.FS
	cache map[string]*ebiten.Image // nil value = known missing

	// Extensions is the candidate order used when background/sprite build
	// paths from a bare name. Defaults to DefaultExtensions; override it
	// (e.g. assets.Extensions = []string{".png"}) to change priority or
	// drop formats you never ship.
	Extensions []string

	// SVGScale rasterizes SVGs at their intrinsic size times this factor.
	// 0 behaves like 1. Raise it (e.g. 2) to render crisper art for
	// higher-density displays; the result is still a plain *ebiten.Image.
	SVGScale float64

	decoders map[string]Decoder
}

func newAssets(fsys fs.FS) *assets {
	a := &assets{
		fsys:       fsys,
		cache:      map[string]*ebiten.Image{},
		Extensions: DefaultExtensions,
		decoders:   map[string]Decoder{},
	}
	a.decoders[".svg"] = decodeSVG
	return a
}

// RegisterDecoder adds or overrides how a file extension (e.g. ".svg",
// leading dot, case-insensitive) is turned into an image. Formats not
// registered here (.png, .jpg, .jpeg, .webp) fall back to image.Decode,
// which already recognizes them via the blank imports above.
func (a *assets) RegisterDecoder(ext string, dec Decoder) {
	a.decoders[strings.ToLower(ext)] = dec
}

// background: `scene "kelas"` -> backgrounds/kelas(.webp|.png|.jpg|.svg|...)
// `scene "kelas.svg"` (an explicit extension) is tried first, verbatim.
func (a *assets) background(name string) *ebiten.Image {
	return a.load(a.withExts(path.Join("backgrounds", name))...)
}

// sprite: `show asep senang` -> characters/asep_senang.*,
// characters/asep/senang.* or characters/asep.*
func (a *assets) sprite(char, expr string) *ebiten.Image {
	bases := []string{
		path.Join("characters", char+"_"+expr),
		path.Join("characters", char, expr),
		path.Join("characters", char),
	}
	var candidates []string
	for _, b := range bases {
		candidates = append(candidates, a.withExts(b)...)
	}
	return a.load(candidates...)
}

// gui loads an optional gui/<name>.* image; nil without a log line.
func (a *assets) gui(name string) *ebiten.Image {
	key := path.Join("gui", name)
	if img, ok := a.cache[key]; ok {
		return img
	}
	var img *ebiten.Image
	for _, p := range a.withExts(key) {
		if img = a.decode(p); img != nil {
			break
		}
	}
	a.cache[key] = img
	return img
}

// withExts returns base as given (so an explicit extension in a script
// still works verbatim) followed by base+ext for each configured extension.
func (a *assets) withExts(base string) []string {
	exts := a.Extensions
	if len(exts) == 0 {
		exts = DefaultExtensions
	}
	out := make([]string, 0, len(exts)+1)
	out = append(out, base)
	for _, ext := range exts {
		out = append(out, base+ext)
	}
	return out
}

// load tries each candidate path in order; the first is the cache key.
func (a *assets) load(candidates ...string) *ebiten.Image {
	if len(candidates) == 0 {
		return nil
	}
	key := candidates[0]
	if img, ok := a.cache[key]; ok {
		return img
	}
	var img *ebiten.Image
	for _, p := range candidates {
		if img = a.decode(p); img != nil {
			break
		}
	}
	if img == nil {
		log.Printf("ebitengine: asset not found: %s (using placeholder)", key)
	}
	a.cache[key] = img
	return img
}

// decode dispatches by file extension: a registered Decoder (SVG by
// default) if one matches, otherwise the standard image.Decode, which
// already handles PNG, JPEG and WebP via the blank imports above.
func (a *assets) decode(p string) *ebiten.Image {
	if a.fsys == nil {
		return nil
	}
	ext := strings.ToLower(path.Ext(p))
	if dec, ok := a.decoders[ext]; ok {
		src, err := dec(a.fsys, p)
		if err != nil {
			if !fsNotExist(err) {
				log.Printf("ebitengine: decode %s: %v", p, err)
			}
			return nil
		}
		return ebiten.NewImageFromImage(src)
	}

	f, err := a.fsys.Open(p)
	if err != nil {
		return nil // candidate simply doesn't exist; not an error
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		log.Printf("ebitengine: decode %s: %v", p, err)
		return nil
	}
	return ebiten.NewImageFromImage(src)
}

// fsNotExist reports whether err is just "the candidate path doesn't
// exist", which load() treats as silent (every extension but the real one
// is expected to miss) rather than worth a log line. This has to be the
// real errors.Is: os.DirFS reports a missing file as a wrapped
// syscall.ENOENT, which only matches fs.ErrNotExist through the special
// Is(error) bool method syscall.Errno implements, not a plain Unwrap chain.
func fsNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

// decodeSVG rasterizes an SVG at its intrinsic size (its viewBox, or its
// width/height attributes), scaled by assets.SVGScale. SVGs have no fixed
// pixel size the way PNG/JPEG/WebP do, so a size has to be picked somehow;
// intrinsic size keeps "one script name -> one image" simple. If you need
// a specific asset larger (crisper on a hi-dpi screen, say), either bump
// SVGScale globally or give that file real width/height attributes.
func decodeSVG(fsys fs.FS, p string) (image.Image, error) {
	f, err := fsys.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	icon, err := oksvg.ReadIconStream(f, oksvg.WarnErrorMode)
	if err != nil {
		return nil, fmt.Errorf("parse svg: %w", err)
	}
	return rasterizeSVG(icon, defaultSVGScale)
}

// defaultSVGScale is used by the package-level decodeSVG registered in
// newAssets, since a bare Decoder func has no access to an *assets
// receiver. assets.decode calls a.decoders[ext] directly today; if you
// want per-instance SVGScale honored, register your own decoder in
// newAssets' caller via RegisterDecoder, e.g.:
//
//	a := newAssets(fsys)
//	a.SVGScale = 2
//	a.RegisterDecoder(".svg", func(fsys fs.FS, p string) (image.Image, error) {
//		f, err := fsys.Open(p)
//		if err != nil { return nil, err }
//		defer f.Close()
//		icon, err := oksvg.ReadIconStream(f, oksvg.WarnErrorMode)
//		if err != nil { return nil, err }
//		return rasterizeSVG(icon, a.SVGScale)
//	})
const defaultSVGScale = 1

// rasterizeSVG renders icon into an RGBA image at its intrinsic size
// times scale (scale <= 0 behaves like 1).
func rasterizeSVG(icon *oksvg.SvgIcon, scale float64) (image.Image, error) {
	if scale <= 0 {
		scale = 1
	}
	w := int(math.Ceil(icon.ViewBox.W * scale))
	h := int(math.Ceil(icon.ViewBox.H * scale))
	if w <= 0 {
		w = 1
	}
	if h <= 0 {
		h = 1
	}

	icon.SetTarget(0, 0, float64(w), float64(h))
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	scanner := rasterx.NewScannerGV(w, h, rgba, rgba.Bounds())
	dasher := rasterx.NewDasher(w, h, scanner)
	icon.Draw(dasher, 1.0)
	return rgba, nil
}
