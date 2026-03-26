package icon

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	pngassets "github.com/darkweaver87/courtdraw/assets/icons"
)

// New returns the standard Fyne theme icon for "new document".
// Icon functions (not vars) avoid calling fyne.CurrentApp() at init time.
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
func Share() fyne.Resource      { return theme.MailSendIcon() }

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

	// ActionDribble and other action icons are generated via cmd/genicons.
	ActionDribble = LoadPngResource("dribble-action")
	ActionPass    = LoadPngResource("pass-action")
	ActionCut     = LoadPngResource("cut-action")
	ActionScreen  = LoadPngResource("screen-action")
	ActionShot    = LoadPngResource("shot-action")
	ActionHandoffRes = LoadPngResource("handoff-action")
	BallIcon      = LoadPngResource("ball")

	AccCone   = LoadPngResource("cone")
	AccLadder = LoadPngResource("ladder")
	AccChair  = LoadPngResource("chair")

	ToolSelect = LoadPngResource("select")

	FlagEN = LoadPngResource("flag-en")
	FlagFR = LoadPngResource("flag-fr")

	ChevronDown  = LoadPngResource("chevron-down")
	ChevronUp    = LoadPngResource("chevron-up")
	ChevronRight = LoadPngResource("chevron-right")
)

// ActionHandoff returns the handoff icon.
func ActionHandoff() fyne.Resource {
	return ActionHandoffRes
}

// ChevronLeft returns the left chevron icon.
func ChevronLeft() fyne.Resource {
	return theme.NavigateBackIcon()
}
