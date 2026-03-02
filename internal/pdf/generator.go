package pdf

import (
	"fmt"

	"github.com/go-pdf/fpdf"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// ExerciseLoader loads an exercise by its store name.
type ExerciseLoader func(name string) (*model.Exercise, error)

// Generate creates a PDF session sheet and writes it to the given path.
func Generate(session *model.Session, loader ExerciseLoader, outputPath string) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	// Resolve all exercises.
	var blocks []exerciseBlock
	for i, entry := range session.Exercises {
		ex, err := loader(entry.Exercise)
		if err != nil {
			// Skip unresolvable exercises but note them.
			ex = &model.Exercise{Name: entry.Exercise + " (not found)"}
		}
		blocks = append(blocks, exerciseBlock{
			entry:    entry,
			exercise: ex,
			index:    i,
		})
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(false, marginBottom)
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.AddPage()

	// Header.
	layoutHeader(pdf, tr, session)

	// Exercise blocks.
	y := marginTop + headerHeight + 4
	for _, block := range blocks {
		y = layoutExerciseBlock(pdf, tr, y, block)
	}

	// Summary table.
	y = layoutSummaryTable(pdf, tr, y, blocks)

	// Coach notes and philosophy.
	_ = layoutCoachNotes(pdf, tr, y, session)

	return pdf.OutputFileAndClose(outputPath)
}

// GenerateBytes creates a PDF and returns it as bytes.
func GenerateBytes(session *model.Session, loader ExerciseLoader) ([]byte, error) {
	if session == nil {
		return nil, fmt.Errorf("session is nil")
	}

	var blocks []exerciseBlock
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

	p := fpdf.New("P", "mm", "A4", "")
	p.SetAutoPageBreak(false, marginBottom)
	tr := p.UnicodeTranslatorFromDescriptor("")
	p.AddPage()

	layoutHeader(p, tr, session)

	y := marginTop + headerHeight + 4
	for _, block := range blocks {
		y = layoutExerciseBlock(p, tr, y, block)
	}

	y = layoutSummaryTable(p, tr, y, blocks)
	_ = layoutCoachNotes(p, tr, y, session)

	var buf []byte
	w := &bytesWriter{data: &buf}
	if err := p.Output(w); err != nil {
		return nil, err
	}
	return buf, nil
}

// bytesWriter is a simple io.Writer that appends to a byte slice.
type bytesWriter struct {
	data *[]byte
}

func (bw *bytesWriter) Write(p []byte) (int, error) {
	*bw.data = append(*bw.data, p...)
	return len(p), nil
}
