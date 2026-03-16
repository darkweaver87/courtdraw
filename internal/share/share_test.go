package share

import (
	"bytes"
	"testing"

	"github.com/darkweaver87/courtdraw/internal/model"
)

func TestBundleRoundTrip(t *testing.T) {
	session := &model.Session{
		Title: "Test Session",
		Date:  "2026-03-13",
		Exercises: []model.ExerciseEntry{
			{Exercise: "drill-a"},
			{Exercise: "drill-b", Variants: []model.ExerciseEntry{
				{Exercise: "drill-b-v1"},
			}},
		},
	}
	exercises := map[string]*model.Exercise{
		"drill-a":    {Name: "Drill A", CourtType: model.HalfCourt},
		"drill-b":    {Name: "Drill B", CourtType: model.FullCourt},
		"drill-b-v1": {Name: "Drill B Variant", CourtType: model.HalfCourt},
	}

	var buf bytes.Buffer
	if err := CreateBundle(&buf, session, exercises); err != nil {
		t.Fatalf("CreateBundle: %v", err)
	}

	gotSession, gotExercises, err := ExtractBundle(&buf)
	if err != nil {
		t.Fatalf("ExtractBundle: %v", err)
	}

	if gotSession.Title != session.Title {
		t.Errorf("session title = %q, want %q", gotSession.Title, session.Title)
	}
	if len(gotSession.Exercises) != 2 {
		t.Errorf("session exercises = %d, want 2", len(gotSession.Exercises))
	}
	if len(gotExercises) != 3 {
		t.Errorf("exercises = %d, want 3", len(gotExercises))
	}
	for name, ex := range exercises {
		got, ok := gotExercises[name]
		if !ok {
			t.Errorf("missing exercise %s", name)
			continue
		}
		if got.Name != ex.Name {
			t.Errorf("exercise %s name = %q, want %q", name, got.Name, ex.Name)
		}
	}
}

func TestCollectExerciseNames(t *testing.T) {
	session := &model.Session{
		Exercises: []model.ExerciseEntry{
			{Exercise: "a"},
			{Exercise: "b", Variants: []model.ExerciseEntry{
				{Exercise: "c"},
				{Exercise: "a"}, // duplicate
			}},
		},
	}
	names := CollectExerciseNames(session)
	if len(names) != 3 {
		t.Errorf("names = %v, want 3 unique", names)
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	plaintext := []byte("hello world, this is a test bundle")
	encrypted, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	decrypted, err := Decrypt(key, encrypted)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key1, _ := GenerateKey()
	key2, _ := GenerateKey()

	encrypted, _ := Encrypt(key1, []byte("secret"))
	_, err := Decrypt(key2, encrypted)
	if err == nil {
		t.Error("expected error decrypting with wrong key")
	}
}
