package main

import (
	"log"
	"os"
	"path/filepath"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/darkweaver87/courtdraw/internal/store"
	"github.com/darkweaver87/courtdraw/internal/ui"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
)

func main() {
	go run()
	app.Main()
}

func run() {
	// init store
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("home dir: %v", err)
	}
	baseDir := filepath.Join(homeDir, ".courtdraw")
	st, err := store.NewYAMLStore(baseDir)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}

	// init theme and app
	th := theme.NewTheme()
	application := ui.NewApp(th, st)

	// load first exercise if available
	if err := application.LoadFirstExercise(); err != nil {
		log.Printf("load exercise: %v", err)
	}

	// create window
	w := new(app.Window)
	w.Option(
		app.Title("CourtDraw"),
		app.Size(unit.Dp(1200), unit.Dp(800)),
	)

	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			if e.Err != nil {
				log.Fatalf("window error: %v", e.Err)
			}
			os.Exit(0)
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			layout.Background{}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.Dimensions{Size: gtx.Constraints.Max}
				},
				func(gtx layout.Context) layout.Dimensions {
					return application.Layout(gtx)
				},
			)
			e.Frame(gtx.Ops)
		}
	}
}
