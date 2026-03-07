package ui

import (
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// LayoutMode represents the current layout mode (desktop or mobile).
type LayoutMode int

const (
	LayoutDesktop LayoutMode = iota
	LayoutMobile
)

const mobileThreshold float32 = 600

// DetectLayoutMode returns the layout mode based on OS and available width.
// On Android/iOS, always returns LayoutMobile regardless of screen size.
// On desktop, returns LayoutMobile if width < 600dp, LayoutDesktop otherwise.
func DetectLayoutMode(size fyne.Size) LayoutMode {
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		return LayoutMobile
	}
	if size.Width < mobileThreshold {
		return LayoutMobile
	}
	return LayoutDesktop
}

// ResponsiveContainer swaps between a desktop layout and a mobile layout
// depending on the current LayoutMode. The same widgets are reused —
// only the wrapping containers change.
type ResponsiveContainer struct {
	widget.BaseWidget

	desktopBuilder func() fyne.CanvasObject
	mobileBuilder  func() fyne.CanvasObject
	currentMode    LayoutMode
	current        fyne.CanvasObject
	initialized    bool
}

// NewResponsiveContainer creates a container that switches layout based on screen size.
func NewResponsiveContainer(desktopBuilder, mobileBuilder func() fyne.CanvasObject) *ResponsiveContainer {
	rc := &ResponsiveContainer{
		desktopBuilder: desktopBuilder,
		mobileBuilder:  mobileBuilder,
	}
	rc.ExtendBaseWidget(rc)
	return rc
}

// ForceRebuild marks the container as needing a layout rebuild on next render.
func (rc *ResponsiveContainer) ForceRebuild() {
	rc.initialized = false
	rc.Refresh()
}

// CreateRenderer returns the renderer for the ResponsiveContainer.
func (rc *ResponsiveContainer) CreateRenderer() fyne.WidgetRenderer {
	return &responsiveRenderer{rc: rc}
}

type responsiveRenderer struct {
	rc *ResponsiveContainer
}

func (r *responsiveRenderer) Layout(size fyne.Size) {
	rc := r.rc
	mode := DetectLayoutMode(size)

	if !rc.initialized || mode != rc.currentMode {
		rc.currentMode = mode
		rc.initialized = true
		switch mode {
		case LayoutDesktop:
			rc.current = rc.desktopBuilder()
		case LayoutMobile:
			rc.current = rc.mobileBuilder()
		}
	}

	if rc.current != nil {
		rc.current.Resize(size)
		rc.current.Move(fyne.NewPos(0, 0))
	}
}

func (r *responsiveRenderer) MinSize() fyne.Size {
	if r.rc.current != nil {
		return r.rc.current.MinSize()
	}
	return fyne.NewSize(200, 200)
}

func (r *responsiveRenderer) Refresh() {
	if r.rc.current != nil {
		r.rc.current.Refresh()
	}
}

func (r *responsiveRenderer) Objects() []fyne.CanvasObject {
	if r.rc.current != nil {
		return []fyne.CanvasObject{r.rc.current}
	}
	return nil
}

func (r *responsiveRenderer) Destroy() {}
