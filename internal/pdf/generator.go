package pdf

import (
	"errors"

	"github.com/go-pdf/fpdf"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// ExerciseLoader loads an exercise by its store name.
type ExerciseLoader func(name string) (*model.Exercise, error)

// Generate creates a PDF session sheet and writes it to the given path.
func Generate(session *model.Session, loader ExerciseLoader, outputPath string, layout PageLayout) error {
	if session == nil {
		return errors.New("session is nil")
	}

	blocks := resolveBlocks(session, loader)
	ctx := newLayoutContext(layout)
	p := newPDF(layout)
	tr := p.UnicodeTranslatorFromDescriptor("")

	renderSession(p, tr, session, blocks, ctx)

	return p.OutputFileAndClose(outputPath)
}

// GenerateBytes creates a PDF and returns it as bytes.
func GenerateBytes(session *model.Session, loader ExerciseLoader, layout PageLayout) ([]byte, error) {
	if session == nil {
		return nil, errors.New("session is nil")
	}

	blocks := resolveBlocks(session, loader)
	ctx := newLayoutContext(layout)
	p := newPDF(layout)
	tr := p.UnicodeTranslatorFromDescriptor("")

	renderSession(p, tr, session, blocks, ctx)

	var buf []byte
	w := &bytesWriter{data: &buf}
	if err := p.Output(w); err != nil {
		return nil, err
	}
	return buf, nil
}

// renderSession renders the full session into the PDF.
// The flow is identical for portrait and 2-up: layout functions call
// ctx.nextPage() which handles physical vs virtual page breaks.
func renderSession(p *fpdf.Fpdf, tr func(string) string, session *model.Session, blocks []exerciseBlock, ctx *layoutContext) {
	// Store session/tr in context so nextPage can redraw headers.
	ctx.session = session
	ctx.tr = tr

	p.AddPage()

	var y float64
	if ctx.mode == LayoutLandscape2Up {
		drawFoldLine(p)
		layoutLandscapeHeader(p, tr, session)
		y = marginTop + headerHeight + 4
		ctx.contentStartY = y
	} else {
		layoutHeader(p, tr, session, ctx)
		y = ctx.margin + headerHeight + 4
	}

	// Exercise blocks.
	for _, block := range blocks {
		y = layoutExerciseBlock(p, tr, y, block, ctx)
	}

	// Summary table.
	y = layoutSummaryTable(p, tr, y, blocks, ctx)

	// Coach notes and philosophy.
	layoutCoachNotes(p, tr, y, session, ctx)
}

func newPDF(layout PageLayout) *fpdf.Fpdf {
	switch layout {
	case LayoutLandscape2Up:
		p := fpdf.New("L", "mm", "A4", "")
		p.SetAutoPageBreak(false, a5Margin)
		return p
	default:
		p := fpdf.New("P", "mm", "A4", "")
		p.SetAutoPageBreak(false, marginBottom)
		return p
	}
}

func resolveBlocks(session *model.Session, loader ExerciseLoader) []exerciseBlock {
	blocks := make([]exerciseBlock, 0, len(session.Exercises))
	for i, entry := range session.Exercises {
		ex, err := loader(entry.Exercise)
		if err != nil {
			ex = &model.Exercise{Name: entry.Exercise + " (not found)"}
		}
		blocks = append(blocks, exerciseBlock{
			entry:    entry,
			exercise: ex,
			index:    i,
		})
	}
	return blocks
}

// bytesWriter is a simple io.Writer that appends to a byte slice.
type bytesWriter struct {
	data *[]byte
}

func (bw *bytesWriter) Write(p []byte) (int, error) {
	*bw.data = append(*bw.data, p...)
	return len(p), nil
}
