package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/ui/icon"
)

const dragRowHeight float32 = 36

// DragList is a reorderable list with drag-and-drop support.
type DragList struct {
	widget.BaseWidget

	items     []DragListItem
	rows      []*dragRow
	box       *fyne.Container
	scroll    *container.Scroll
	indicator *canvas.Rectangle

	// Drag state.
	dragging    bool
	dragIdx     int
	dragStartY  float32
	dragOffsetY float32
	dropTarget  int

	OnReorder func(from, to int)
	OnDelete  func(idx int)
	OnMove    func(idx, dir int)
}

// DragListItem represents one entry in the list.
type DragListItem struct {
	Text string
}

var _ fyne.Widget = (*DragList)(nil)
var _ fyne.Draggable = (*DragList)(nil)
var _ fyne.Tappable = (*DragList)(nil)
var _ desktop.Hoverable = (*DragList)(nil)

// NewDragList creates a new draggable list.
func NewDragList() *DragList {
	dl := &DragList{
		dragIdx:    -1,
		dropTarget: -1,
	}
	dl.box = container.NewVBox()
	dl.indicator = canvas.NewRectangle(color.NRGBA{R: 0x29, G: 0x6d, B: 0xd4, A: 0xff})
	dl.indicator.Hide()
	dl.scroll = container.NewVScroll(dl.box)
	dl.ExtendBaseWidget(dl)
	return dl
}

// SetItems updates the list content.
func (dl *DragList) SetItems(items []DragListItem) {
	dl.items = items
	dl.rebuildRows()
}

func (dl *DragList) rebuildRows() {
	dl.box.RemoveAll()
	dl.rows = make([]*dragRow, len(dl.items))
	for i, item := range dl.items {
		row := newDragRow(i, item.Text, dl)
		dl.rows[i] = row
		dl.box.Add(row.container)
	}
	dl.box.Add(dl.indicator)
	dl.box.Refresh()
}

// Widget returns the scrollable container.
func (dl *DragList) Widget() fyne.CanvasObject {
	return dl.scroll
}

// CreateRenderer returns a simple renderer wrapping the scroll container.
func (dl *DragList) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(dl.scroll)
}

func (dl *DragList) Tapped(*fyne.PointEvent) {}

func (dl *DragList) MouseIn(*desktop.MouseEvent)  {}
func (dl *DragList) MouseMoved(*desktop.MouseEvent) {}
func (dl *DragList) MouseOut()                     {}

// Dragged handles drag events for reordering.
func (dl *DragList) Dragged(ev *fyne.DragEvent) {
	if !dl.dragging {
		// Determine which row was hit.
		idx := dl.hitRow(ev.Position)
		if idx < 0 {
			return
		}
		dl.dragging = true
		dl.dragIdx = idx
		dl.dragStartY = ev.Position.Y
		dl.dragOffsetY = 0
		dl.dropTarget = idx
		// Dim the dragged row.
		if idx < len(dl.rows) {
			dl.rows[idx].setDragging(true)
		}
	}

	dl.dragOffsetY += ev.Dragged.DY
	target := dl.computeDropTarget(ev.Position)
	if target != dl.dropTarget {
		dl.dropTarget = target
		dl.showIndicator(target)
	}
}

// DragEnd finalizes the drag.
func (dl *DragList) DragEnd() {
	if !dl.dragging {
		return
	}
	// Restore dragged row appearance.
	if dl.dragIdx >= 0 && dl.dragIdx < len(dl.rows) {
		dl.rows[dl.dragIdx].setDragging(false)
	}
	dl.indicator.Hide()

	from := dl.dragIdx
	to := dl.dropTarget
	dl.dragging = false
	dl.dragIdx = -1
	dl.dropTarget = -1

	if from != to && from >= 0 && to >= 0 && dl.OnReorder != nil {
		dl.OnReorder(from, to)
	}
}

func (dl *DragList) hitRow(pos fyne.Position) int {
	// Account for scroll offset.
	y := pos.Y + dl.scroll.Offset.Y
	idx := int(y / dragRowHeight)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(dl.items) {
		idx = len(dl.items) - 1
	}
	return idx
}

func (dl *DragList) computeDropTarget(pos fyne.Position) int {
	y := pos.Y + dl.scroll.Offset.Y
	target := int(y / dragRowHeight)
	if target < 0 {
		target = 0
	}
	if target >= len(dl.items) {
		target = len(dl.items) - 1
	}
	return target
}

func (dl *DragList) showIndicator(target int) {
	if target < 0 || len(dl.items) == 0 {
		dl.indicator.Hide()
		return
	}
	indicatorY := float32(target) * dragRowHeight
	if target > dl.dragIdx {
		indicatorY += dragRowHeight
	}
	dl.indicator.Resize(fyne.NewSize(dl.box.Size().Width, 2))
	dl.indicator.Move(fyne.NewPos(0, indicatorY))
	dl.indicator.Show()
	dl.indicator.Refresh()
}

// --- dragRow: one row in the list ---

type dragRow struct {
	container *fyne.Container
	label     *canvas.Text
	bg        *canvas.Rectangle
	delBtn    *widget.Button
	idx       int
}

func newDragRow(idx int, text string, dl *DragList) *dragRow {
	r := &dragRow{idx: idx}
	r.bg = canvas.NewRectangle(color.Transparent)
	r.bg.Resize(fyne.NewSize(300, dragRowHeight))

	handle := canvas.NewImageFromResource(icon.DragHandle())
	handle.FillMode = canvas.ImageFillContain
	handle.SetMinSize(fyne.NewSize(16, 16))

	r.label = canvas.NewText(text, color.White)
	r.label.TextSize = 12

	upBtn := widget.NewButtonWithIcon("", icon.MoveUp(), func() {
		if dl.OnMove != nil {
			dl.OnMove(idx, -1)
		}
	})
	upBtn.Importance = widget.LowImportance
	downBtn := widget.NewButtonWithIcon("", icon.MoveDown(), func() {
		if dl.OnMove != nil {
			dl.OnMove(idx, 1)
		}
	})
	downBtn.Importance = widget.LowImportance
	r.delBtn = widget.NewButtonWithIcon("", icon.Delete(), func() {
		if dl.OnDelete != nil {
			dl.OnDelete(idx)
		}
	})
	r.delBtn.Importance = widget.LowImportance

	row := container.NewHBox(handle, r.label, layout.NewSpacer(), upBtn, downBtn, r.delBtn)
	r.container = container.NewStack(r.bg, row)
	r.container.Resize(fyne.NewSize(300, dragRowHeight))
	return r
}

func (r *dragRow) setDragging(on bool) {
	if on {
		r.bg.FillColor = color.NRGBA{R: 0x29, G: 0x6d, B: 0xd4, A: 0x40}
	} else {
		r.bg.FillColor = color.Transparent
	}
	r.bg.Refresh()
}
