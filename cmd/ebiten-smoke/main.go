// Command ebiten-smoke opens the Ebitengine backend with a hardcoded session,
// so the window can be checked before the real VM adapter exists.
//
//	go run ./cmd/ebiten-smoke
package main

import (
	"log"
	"os"

	"ango/backend/ebitengine"
)

type scripted struct {
	views []ebitengine.View
	i     int
}

func (s *scripted) View() ebitengine.View { return s.views[s.i] }

func (s *scripted) Advance() error {
	if s.i < len(s.views)-1 {
		s.i++
	}
	return nil
}

// Choose ignores which option was picked and just moves on.
func (s *scripted) Choose(int) error { return s.Advance() }

func main() {
	asep := ebitengine.Sprite{Char: "asep", Expr: "senang", Pos: "left"}
	sess := &scripted{views: []ebitengine.View{
		{
			Background: "kelas",
			Sprites:    []ebitengine.Sprite{asep},
			Text:       "Bel pulang sudah berbunyi. Kelas mulai sepi.",
		},
		{
			Background: "kelas",
			Sprites:    []ebitengine.Sprite{asep},
			Speaker:    "Asep",
			Text:       "Halo, aku Asep. Kunci lemari itu hilang, jadi aku belum bisa pulang.",
		},
		{
			Background: "lorong",
			Choices:    []string{"Ambil kunci", "Periksa meja", "Pulang saja"},
		},
		{
			Background: "lorong",
			Sprites:    []ebitengine.Sprite{asep},
			Speaker:    "Asep",
			Text:       "Ketemu! Kuncinya ada di bawah meja.",
		},
		{Background: "lorong", Ended: true},
	}}

	err := ebitengine.Run(sess, ebitengine.Options{
		Title:  "Ango - smoke test",
		Assets: os.DirFS("examples/basic/assets"),
	})
	if err != nil {
		log.Fatal(err)
	}
}
