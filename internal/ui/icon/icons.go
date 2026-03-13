package icon

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	pngassets "github.com/darkweaver87/courtdraw/assets/icons"
)

// Standard Fyne theme icons for toolbar actions.
// These are functions (not vars) to avoid calling fyne.CurrentApp() at init time.
func New() fyne.Resource       { return theme.DocumentCreateIcon() }
func Open() fyne.Resource      { return theme.FolderOpenIcon() }
func Save() fyne.Resource      { return theme.DocumentSaveIcon() }
func Duplicate() fyne.Resource { return theme.ContentCopyIcon() }
func Import() fyne.Resource    { return theme.DownloadIcon() }
func Play() fyne.Resource      { return theme.MediaPlayIcon() }
func Pause() fyne.Resource     { return theme.MediaPauseIcon() }
func Stop() fyne.Resource      { return theme.MediaStopIcon() }
func Prev() fyne.Resource      { return theme.MediaSkipPreviousIcon() }
func Next() fyne.Resource      { return theme.MediaSkipNextIcon() }
func Delete() fyne.Resource    { return theme.DeleteIcon() }
func Close() fyne.Resource     { return theme.CancelIcon() }
func Add() fyne.Resource       { return theme.ContentAddIcon() }
func Refresh() fyne.Resource   { return theme.ViewRefreshIcon() }
func Settings() fyne.Resource  { return theme.SettingsIcon() }
func Info() fyne.Resource      { return theme.InfoIcon() }
func Upload() fyne.Resource    { return theme.UploadIcon() }
func MoveUp() fyne.Resource     { return theme.MoveUpIcon() }
func MoveDown() fyne.Resource   { return theme.MoveDownIcon() }
func DragHandle() fyne.Resource { return theme.MenuIcon() }
func Preview() fyne.Resource    { return theme.VisibilityIcon() }
func Training() fyne.Resource   { return theme.MediaPlayIcon() }
func Back() fyne.Resource       { return theme.NavigateBackIcon() }
func Timer() fyne.Resource      { return theme.HistoryIcon() }

// LoadPngResource loads a PNG icon from assets/icons/ as a Fyne static resource.
func LoadPngResource(name string) fyne.Resource {
	data, err := pngassets.FS.ReadFile(name + ".png")
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource(name+".png", data)
}

// Basketball tool palette PNG icons.
var (
	PlayerAttacker = LoadPngResource("attacker")
	PlayerDefender = LoadPngResource("defender")
	PlayerCoach    = LoadPngResource("coach")
	PlayerPG       = LoadPngResource("pg")
	PlayerSG       = LoadPngResource("sg")
	PlayerSF       = LoadPngResource("sf")
	PlayerPF       = LoadPngResource("pf")
	PlayerCenter   = LoadPngResource("center")
	PlayerQueue    = LoadPngResource("queue")

	ActionPass     = LoadPngResource("pass")
	ActionDribble  = LoadPngResource("dribble")
	ActionSprint   = LoadPngResource("sprint")
	ActionShot     = LoadPngResource("shot")
	ActionScreen   = LoadPngResource("screen")
	ActionCut      = LoadPngResource("cut")
	ActionCloseOut = LoadPngResource("close-out")
	ActionContest  = LoadPngResource("contest")
	ActionReverse  = LoadPngResource("reverse")

	AccCone   = LoadPngResource("cone")
	AccLadder = LoadPngResource("ladder")
	AccChair  = LoadPngResource("chair")

	ToolSelect = LoadPngResource("select")
)
