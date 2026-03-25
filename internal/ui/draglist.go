package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/ui/icon"
)

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
	OnPreview func(idx int)

	// MinimalMode hides all buttons (preview, up/down, delete) — drag-only.
	MinimalMode bool
}

// DragListItem represents one entry in the list.
type DragListItem struct {
	Text string
}

var _ fyne.Widget = (*DragList)(nil)
var _ fyne.Draggable = (*DragList)(nil)
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

// Widget returns the scrollable container (or raw box in MinimalMode).
func (dl *DragList) Widget() fyne.CanvasObject {
	if dl.MinimalMode {
		return dl.box
	}
	return dl.scroll
}

// CreateRenderer returns a simple renderer wrapping the content.
func (dl *DragList) CreateRenderer() fyne.WidgetRenderer {
	if dl.MinimalMode {
		return widget.NewSimpleRenderer(dl.box)
	}
	return widget.NewSimpleRenderer(dl.scroll)
}

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

func (dl *DragList) rowY(idx int) float32 {
	var y float32
	for i := 0; i < idx && i < len(dl.rows); i++ {
		y += dl.rows[i].container.Size().Height
	}
	return y
}

func (dl *DragList) hitRow(pos fyne.Position) int {
	y := pos.Y + dl.scroll.Offset.Y
	var cumY float32
	for i, row := range dl.rows {
		h := row.container.Size().Height
		if y < cumY+h {
			return i
		}
		cumY += h
	}
	if len(dl.rows) > 0 {
		return len(dl.rows) - 1
	}
	return -1
}

func (dl *DragList) computeDropTarget(pos fyne.Position) int {
	y := pos.Y + dl.scroll.Offset.Y
	var cumY float32
	for i, row := range dl.rows {
		h := row.container.Size().Height
		if y < cumY+h {
			return i
		}
		cumY += h
	}
	if len(dl.rows) > 0 {
		return len(dl.rows) - 1
	}
	return 0
}

func (dl *DragList) showIndicator(target int) {
	if target < 0 || len(dl.items) == 0 {
		dl.indicator.Hide()
		return
	}
	indicatorY := dl.rowY(target)
	if target > dl.dragIdx {
		indicatorY += dl.rows[target].container.Size().Height
	}
	dl.indicator.Resize(fyne.NewSize(dl.box.Size().Width, 2))
	dl.indicator.Move(fyne.NewPos(0, indicatorY))
	dl.indicator.Show()
	dl.indicator.Refresh()
}

// --- dragRow: one row in the list ---

type dragRow struct {
	container *fyne.Container
	label     *widget.Label
	bg        *canvas.Rectangle
	delBtn    *widget.Button
	idx       int
}

func newDragRow(idx int, text string, dl *DragList) *dragRow {
	r := &dragRow{idx: idx}
	r.bg = canvas.NewRectangle(color.Transparent)

	handle := canvas.NewImageFromResource(icon.DragHandle())
	handle.FillMode = canvas.ImageFillContain
	handle.SetMinSize(fyne.NewSize(16, 16))

	r.label = widget.NewLabel(text)
	r.label.Wrapping = fyne.TextWrapWord

	previewBtn := widget.NewButtonWithIcon("", icon.Preview(), func() {
		if dl.OnPreview != nil {
			dl.OnPreview(idx)
		}
	})
	previewBtn.Importance = widget.LowImportance

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

	var row *fyne.Container
	if dl.MinimalMode {
		row = container.NewBorder(nil, nil, handle, nil, r.label)
	} else {
		row = container.NewBorder(nil, nil, handle, container.NewHBox(previewBtn, upBtn, downBtn, r.delBtn), r.label)
	}
	r.container = container.NewStack(r.bg, row)
	return r
}


// tappableArea is an invisible widget that captures tap events.
type tappableArea struct {
	widget.BaseWidget

	onTap func()
}

func newTappableArea(onTap func()) *tappableArea {
	t := &tappableArea{onTap: onTap}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tappableArea) Tapped(*fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *tappableArea) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}

func (r *dragRow) setDragging(on bool) {
	if on {
		r.bg.FillColor = color.NRGBA{R: 0x29, G: 0x6d, B: 0xd4, A: 0x40}
	} else {
		r.bg.FillColor = color.Transparent
	}
	r.bg.Refresh()
}
