package ui

import (
	"image/color"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var (
	tabBarHeight float32 = 52
	tabIconSize  float32 = 24
	tabFontSize  float32 = 10
)

func init() {
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		tabBarHeight = 80
		tabIconSize = 40
		tabFontSize = 14
	}
}

var (
	tabBgColor         = color.NRGBA{R: 0x1e, G: 0x1e, B: 0x1e, A: 0xff}
	tabActiveColor     = color.NRGBA{R: 0x29, G: 0x6d, B: 0xd4, A: 0xff}
	tabInactiveColor   = color.NRGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff}
	tabLabelActiveColor = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
)

// TabItem describes a single tab in the mobile tab bar.
type TabItem struct {
	Icon    fyne.Resource
	Label   string
	Content fyne.CanvasObject
}

// TabBar is a custom bottom tab bar with larger touch targets
// than Fyne's built-in AppTabs. Icons are 24dp, labels 10pt, bar 52dp.
type TabBar struct {
	tabs      []TabItem
	active    int
	bar       *fyne.Container
	content   *fyne.Container
	icons     []*canvas.Image
	labels    []*canvas.Text
	indicators []*canvas.Rectangle
}

// NewTabBar creates a custom mobile tab bar with the given tabs.
func NewTabBar(tabs ...TabItem) *TabBar {
	mtb := &TabBar{
		tabs: tabs,
	}
	mtb.buildUI()
	return mtb
}

func (mtb *TabBar) buildUI() {
	// Content stack — only the active tab's content is visible.
	mtb.content = container.NewStack()
	for i, tab := range mtb.tabs {
		if i == 0 {
			mtb.content.Add(tab.Content)
		} else {
			hidden := tab.Content
			hidden.Hide()
			mtb.content.Add(hidden)
		}
	}

	// Bar: one column per tab, each with indicator + icon + label.
	barItems := make([]fyne.CanvasObject, 0, len(mtb.tabs))
	for i, tab := range mtb.tabs {
		idx := i

		// Active indicator (thin colored bar at top).
		indicator := canvas.NewRectangle(color.Transparent)
		indicator.SetMinSize(fyne.NewSize(0, 3))
		if idx == 0 {
			indicator.FillColor = tabActiveColor
		}
		mtb.indicators = append(mtb.indicators, indicator)

		// Icon.
		ico := canvas.NewImageFromResource(tab.Icon)
		ico.FillMode = canvas.ImageFillContain
		ico.SetMinSize(fyne.NewSize(tabIconSize, tabIconSize))
		mtb.icons = append(mtb.icons, ico)

		// Label.
		lbl := canvas.NewText(tab.Label, tabInactiveColor)
		lbl.TextSize = tabFontSize
		lbl.Alignment = fyne.TextAlignCenter
		if idx == 0 {
			lbl.Color = tabLabelActiveColor
		}
		mtb.labels = append(mtb.labels, lbl)

		// Tappable container for the tab button.
		col := container.NewVBox(indicator, container.NewCenter(ico), lbl)
		btn := newTabTappable(col, func() { mtb.SelectTab(idx) })
		barItems = append(barItems, btn)
	}

	bg := canvas.NewRectangle(tabBgColor)
	bg.SetMinSize(fyne.NewSize(0, tabBarHeight))
	grid := container.NewGridWithColumns(len(mtb.tabs), barItems...)
	mtb.bar = container.NewStack(bg, container.NewPadded(grid))
}

// SelectTab switches to the tab at the given index.
func (mtb *TabBar) SelectTab(idx int) {
	if idx < 0 || idx >= len(mtb.tabs) || idx == mtb.active {
		return
	}
	// Hide old, show new.
	mtb.content.Objects[mtb.active].Hide()
	mtb.content.Objects[idx].Show()

	// Update indicators and labels.
	mtb.indicators[mtb.active].FillColor = color.Transparent
	mtb.indicators[mtb.active].Refresh()
	mtb.labels[mtb.active].Color = tabInactiveColor
	mtb.labels[mtb.active].Refresh()

	mtb.indicators[idx].FillColor = tabActiveColor
	mtb.indicators[idx].Refresh()
	mtb.labels[idx].Color = tabLabelActiveColor
	mtb.labels[idx].Refresh()

	mtb.active = idx
	mtb.content.Refresh()
}

// SetTabLabel updates the label text for a tab (e.g., after language change).
func (mtb *TabBar) SetTabLabel(idx int, label string) {
	if idx >= 0 && idx < len(mtb.labels) {
		mtb.labels[idx].Text = label
		mtb.labels[idx].Refresh()
	}
}

// Widget returns the full tab bar + content as a border layout (content center, bar bottom).
func (mtb *TabBar) Widget() fyne.CanvasObject {
	return container.NewBorder(nil, mtb.bar, nil, nil, mtb.content)
}

// tabTappable wraps a canvas object to make it tappable.
type tabTappable struct {
	widget.BaseWidget
	content fyne.CanvasObject
	onTap   func()
}

func newTabTappable(content fyne.CanvasObject, onTap func()) *tabTappable {
	t := &tabTappable{content: content, onTap: onTap}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tabTappable) Tapped(*fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *tabTappable) CreateRenderer() fyne.WidgetRenderer {
	return &tabTappableRenderer{t: t}
}

type tabTappableRenderer struct {
	t *tabTappable
}

func (r *tabTappableRenderer) Layout(size fyne.Size) {
	r.t.content.Resize(size)
	r.t.content.Move(fyne.NewPos(0, 0))
}

func (r *tabTappableRenderer) MinSize() fyne.Size {
	return r.t.content.MinSize()
}

func (r *tabTappableRenderer) Refresh() {
	r.t.content.Refresh()
}

func (r *tabTappableRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.t.content}
}

func (r *tabTappableRenderer) Destroy() {}
