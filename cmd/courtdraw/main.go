package main

import (
	"log"
	"os"
	"path/filepath"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/store"
	"github.com/darkweaver87/courtdraw/internal/ui"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
)

func main() {
	go run()
	app.Main()
}

func run() {
	log.Println("courtdraw: run() started")
	// init store
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("home dir: %v", err)
	}
	log.Printf("courtdraw: homeDir=%s", homeDir)
	baseDir := filepath.Join(homeDir, ".courtdraw")
	st, err := store.NewYAMLStore(baseDir)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}
	log.Println("courtdraw: store initialized")

	// init i18n: load locales, then apply saved or system language
	i18n.Load()
	settings, err := st.LoadSettings()
	if err != nil {
		log.Printf("load settings: %v", err)
	}
	if settings.Language != "" {
		i18n.SetLang(i18n.Lang(settings.Language))
	} else {
		// No saved preference — detect from system locale
		i18n.SetLang(i18n.DetectSystemLang())
	}

	// detect library directory (alongside the executable or in repo)
	exePath, _ := os.Executable()
	libraryDir := ""
	candidates := []string{
		filepath.Join(filepath.Dir(exePath), "library"),
		filepath.Join(".", "library"),
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			libraryDir = c
			break
		}
	}

	log.Printf("courtdraw: libraryDir=%q", libraryDir)
	// init theme and app
	th := theme.NewTheme()
	application := ui.NewApp(th, st, libraryDir)
	log.Println("courtdraw: app created")

	// start with a blank exercise and blank session
	application.NewExercise()
	application.NewSession()

	// create window
	w := new(app.Window)
	application.SetWindow(w)
	w.Option(
		app.Title("CourtDraw"),
		app.Size(unit.Dp(1200), unit.Dp(800)),
	)

	log.Println("courtdraw: entering event loop")
	var ops op.Ops
	frameCount := 0
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			if e.Err != nil {
				log.Fatalf("window error: %v", e.Err)
			}
			os.Exit(0)
		case app.FrameEvent:
			frameCount++
			if frameCount <= 3 {
				log.Printf("courtdraw: frame %d, size=%v", frameCount, e.Size)
			}
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
