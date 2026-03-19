package ui

import "fyne.io/fyne/v2"

// flowLayout arranges objects left-to-right, wrapping to the next row
// when the available width is exceeded. Each row is as tall as its tallest element.
type flowLayout struct {
	hGap, vGap   float32
	lastHeight   float32 // height computed by last Layout call
	lastWidth    float32 // container width from last Layout call
}

func newFlowLayout(hGap, vGap float32) *flowLayout {
	return &flowLayout{hGap: hGap, vGap: vGap}
}

func (f *flowLayout) Layout(objects []fyne.CanvasObject, containerSize fyne.Size) {
	x, y := float32(0), float32(0)
	rowH := float32(0)
	for _, o := range objects {
		if !o.Visible() {
			continue
		}
		s := o.MinSize()
		if x > 0 && x+s.Width > containerSize.Width {
			x = 0
			y += rowH + f.vGap
			rowH = 0
		}
		o.Resize(s)
		o.Move(fyne.NewPos(x, y))
		x += s.Width + f.hGap
		if s.Height > rowH {
			rowH = s.Height
		}
	}
	f.lastHeight = y + rowH
	f.lastWidth = containerSize.Width
}

func (f *flowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	// Width: widest single element (minimum to avoid zero).
	maxW := float32(0)
	for _, o := range objects {
		if !o.Visible() {
			continue
		}
		if w := o.MinSize().Width; w > maxW {
			maxW = w
		}
	}
	// Height: use the last computed height if available, otherwise one row.
	h := f.lastHeight
	if h <= 0 {
		for _, o := range objects {
			if !o.Visible() {
				continue
			}
			if oh := o.MinSize().Height; oh > h {
				h = oh
			}
		}
	}
	return fyne.NewSize(maxW, h)
}
