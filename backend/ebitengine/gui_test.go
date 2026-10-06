package ebitengine

import (
	"image/color"
	"testing"
	"testing/fstest"
)

func TestThemeDefaults(t *testing.T) {
	th := loadTheme(nil)
	if th.box.Dx() != 1200 || th.box.Min.Y != 520 || th.btnW != 720 || th.maxLines != 4 {
		t.Errorf("defaults changed: %+v", th)
	}
	if th.menu.Enabled {
		t.Error("menu must be opt-in")
	}
	if th.text != (color.NRGBA{255, 255, 255, 255}) || th.textbox != (color.NRGBA{0, 0, 0, 190}) {
		t.Errorf("default colors changed: text=%v textbox=%v", th.text, th.textbox)
	}
}

func TestThemeOverrideKeepsOtherDefaults(t *testing.T) {
	fsys := fstest.MapFS{"gui/theme.json": {Data: []byte(
		`{"font_size": 30, "colors": {"text": "#FF0000"}, "menu": {"enabled": true, "start": "Main"}}`)}}
	th := loadTheme(fsys)
	if th.fontSize != 30 {
		t.Errorf("fontSize = %v, want 30", th.fontSize)
	}
	if th.text != (color.NRGBA{255, 0, 0, 255}) {
		t.Errorf("text = %v, want red", th.text)
	}
	if !th.menu.Enabled || th.menu.Start != "Main" || th.menu.Quit != "Keluar" {
		t.Errorf("menu = %+v", th.menu)
	}
	if th.box.Dx() != 1200 || th.btnH != 56 {
		t.Errorf("unspecified fields must keep defaults: %+v", th)
	}
}

func TestThemeBrokenFileFallsBack(t *testing.T) {
	fsys := fstest.MapFS{"gui/theme.json": {Data: []byte(`{not json`)}}
	if th := loadTheme(fsys); th.fontSize != 26 {
		t.Errorf("broken theme should give defaults, got fontSize %v", th.fontSize)
	}
}

func TestBadColorFallsBackToDefault(t *testing.T) {
	fsys := fstest.MapFS{"gui/theme.json": {Data: []byte(`{"colors": {"text": "merah"}}`)}}
	if th := loadTheme(fsys); th.text != (color.NRGBA{255, 255, 255, 255}) {
		t.Errorf("bad color should fall back, got %v", th.text)
	}
}
