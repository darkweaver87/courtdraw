package share

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/darkweaver87/courtdraw/internal/model"
)

const sessionEntry = "session.yaml"
const exerciseDir = "exercises/"

// CreateBundle writes a .courtdraw tar.gz archive containing the session
// and all referenced exercises.
func CreateBundle(w io.Writer, session *model.Session, exercises map[string]*model.Exercise) error {
	gw := gzip.NewWriter(w)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Write session.yaml
	sData, err := yaml.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	if err := writeTarEntry(tw, sessionEntry, sData); err != nil {
		return err
	}

	// Write each exercise
	for name, ex := range exercises {
		data, err := yaml.Marshal(ex)
		if err != nil {
			return fmt.Errorf("marshal exercise %s: %w", name, err)
		}
		if err := writeTarEntry(tw, exerciseDir+name+".yaml", data); err != nil {
			return err
		}
	}
	return nil
}

// ExtractBundle reads a .courtdraw tar.gz and returns the session and exercises.
func ExtractBundle(r io.Reader) (*model.Session, map[string]*model.Exercise, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, nil, fmt.Errorf("open gzip: %w", err)
	}
	defer gr.Close()
	tr := tar.NewReader(gr)

	var session *model.Session
	exercises := make(map[string]*model.Exercise)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("read tar: %w", err)
		}

		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, nil, fmt.Errorf("read entry %s: %w", hdr.Name, err)
		}

		switch {
		case hdr.Name == sessionEntry:
			session = &model.Session{}
			if err := yaml.Unmarshal(data, session); err != nil {
				return nil, nil, fmt.Errorf("unmarshal session: %w", err)
			}
		case strings.HasPrefix(hdr.Name, exerciseDir):
			name := strings.TrimSuffix(path.Base(hdr.Name), ".yaml")
			if name == "" {
				continue
			}
			ex := &model.Exercise{}
			if err := yaml.Unmarshal(data, ex); err != nil {
				return nil, nil, fmt.Errorf("unmarshal exercise %s: %w", name, err)
			}
			exercises[name] = ex
		}
	}

	if session == nil {
		return nil, nil, errors.New("bundle missing session.yaml")
	}
	return session, exercises, nil
}

// CollectExerciseNames returns all unique exercise names referenced by a session,
// including those inside variant entries.
func CollectExerciseNames(session *model.Session) []string {
	seen := make(map[string]bool)
	var names []string
	var collect func(entries []model.ExerciseEntry)
	collect = func(entries []model.ExerciseEntry) {
		for _, e := range entries {
			if e.Exercise != "" && !seen[e.Exercise] {
				seen[e.Exercise] = true
				names = append(names, e.Exercise)
			}
			collect(e.Variants)
		}
	}
	collect(session.Exercises)
	return names
}

func writeTarEntry(tw *tar.Writer, name string, data []byte) error {
	hdr := &tar.Header{
		Name: name,
		Mode: 0644,
		Size: int64(len(data)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("write tar header %s: %w", name, err)
	}
	if _, err := io.Copy(tw, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("write tar data %s: %w", name, err)
	}
	return nil
}
