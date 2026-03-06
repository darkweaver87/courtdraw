package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/store"
	"github.com/darkweaver87/courtdraw/internal/ui"
	cdtheme "github.com/darkweaver87/courtdraw/internal/ui/theme"
)

func main() {
	// Create Fyne app first — needed for Storage() on mobile.
	a := app.New()
	a.Settings().SetTheme(&cdtheme.CourtDrawTheme{})

	// Determine base directory for data storage.
	baseDir := dataDir(a)
	st, err := store.NewYAMLStore(baseDir)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}

	// Init i18n: load locales, then apply saved or system language.
	i18n.Load()
	settings, err := st.LoadSettings()
	if err != nil {
		log.Printf("load settings: %v", err)
	}
	if settings.Language != "" {
		i18n.SetLang(i18n.Lang(settings.Language))
	} else {
		i18n.SetLang(i18n.DetectSystemLang())
	}

	// Detect library directory (alongside the executable or in repo).
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

	w := a.NewWindow("CourtDraw")
	w.Resize(fyne.NewSize(1200, 800))

	// Create and initialize the application.
	application := ui.NewApp(st, settings, libraryDir, w)
	w.SetContent(application.BuildUI())

	// Initialize exercise and session after UI is built (court widget exists).
	application.NewExercise()
	application.NewSession()
	w.ShowAndRun()
}

// dataDir returns the appropriate data directory per platform.
// On Android/iOS, uses Fyne's app storage. On desktop, uses ~/.courtdraw.
func dataDir(a fyne.App) string {
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		// Fyne's RootURI gives the app-private directory.
		uri := a.Storage().RootURI()
		return uri.Path()
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("home dir: %v", err)
	}
	return filepath.Join(homeDir, ".courtdraw")
}
