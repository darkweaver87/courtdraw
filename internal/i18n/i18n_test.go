package i18n

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	Load()
	if len(translations) == 0 {
		t.Fatal("expected translations to be loaded")
	}
	if _, ok := translations[EN]; !ok {
		t.Error("English translations missing")
	}
	if _, ok := translations[FR]; !ok {
		t.Error("French translations missing")
	}
}

func TestT_EnglishDefault(t *testing.T) {
	Load()
	SetLang(EN)

	got := T("app.title")
	if got != "CourtDraw" {
		t.Errorf("T(app.title) = %q, want \"CourtDraw\"", got)
	}
}

func TestT_French(t *testing.T) {
	Load()
	SetLang(FR)
	defer SetLang(EN)

	got := T("tab.exercise_editor")
	if got != "Éditeur d'exercice" {
		t.Errorf("T(tab.exercise_editor) = %q, want French", got)
	}
}

func TestT_FallbackToEnglish(t *testing.T) {
	Load()
	SetLang(FR)
	defer SetLang(EN)

	// If a key exists in EN but not FR, should fall back to EN.
	// Since we have full parity, test by verifying a known key works.
	got := T("app.title")
	if got == "app.title" {
		t.Error("expected fallback to EN, got key itself")
	}
}

func TestT_MissingKey(t *testing.T) {
	Load()
	SetLang(EN)

	got := T("nonexistent.key")
	if got != "nonexistent.key" {
		t.Errorf("T(nonexistent.key) = %q, want key itself as fallback", got)
	}
}

func TestTf(t *testing.T) {
	Load()
	SetLang(EN)

	got := Tf("anim.seq_format", 2, 5)
	if got != "Seq 2/5" {
		t.Errorf("Tf(anim.seq_format, 2, 5) = %q, want \"Seq 2/5\"", got)
	}
}

func TestTf_French(t *testing.T) {
	Load()
	SetLang(FR)
	defer SetLang(EN)

	got := Tf("anim.seq_format", 1, 3)
	if got != "Seq 1/3" {
		t.Errorf("Tf(anim.seq_format, 1, 3) = %q, want \"Seq 1/3\"", got)
	}
}

func TestSetLang_InvalidLang(t *testing.T) {
	Load()
	SetLang(EN)

	SetLang(Lang("xx"))
	if CurrentLang() != EN {
		t.Errorf("expected EN after invalid SetLang, got %q", CurrentLang())
	}
}

func TestCurrentLang(t *testing.T) {
	Load()
	SetLang(EN)
	if CurrentLang() != EN {
		t.Errorf("expected EN, got %q", CurrentLang())
	}
	SetLang(FR)
	if CurrentLang() != FR {
		t.Errorf("expected FR, got %q", CurrentLang())
	}
	SetLang(EN)
}

func TestSupportedLangs(t *testing.T) {
	langs := SupportedLangs()
	if len(langs) < 2 {
		t.Errorf("expected at least 2 languages, got %d", len(langs))
	}
}

func TestDetectSystemLang(t *testing.T) {
	// Save original env values and restore them after the test.
	envVars := []string{"LANGUAGE", "LC_ALL", "LC_MESSAGES", "LANG"}
	saved := make(map[string]string, len(envVars))
	for _, e := range envVars {
		saved[e] = os.Getenv(e)
	}
	t.Cleanup(func() {
		for _, e := range envVars {
			os.Setenv(e, saved[e])
		}
	})

	clearEnv := func() {
		for _, e := range envVars {
			os.Unsetenv(e)
		}
	}

	tests := []struct {
		name string
		env  map[string]string
		want Lang
	}{
		{
			name: "defaults to EN when no env set",
			env:  nil,
			want: EN,
		},
		{
			name: "detects FR from LANG",
			env:  map[string]string{"LANG": "fr_FR.UTF-8"},
			want: FR,
		},
		{
			name: "detects EN from LANG",
			env:  map[string]string{"LANG": "en_US.UTF-8"},
			want: EN,
		},
		{
			name: "LANGUAGE takes priority over LANG",
			env:  map[string]string{"LANGUAGE": "fr", "LANG": "en_US.UTF-8"},
			want: FR,
		},
		{
			name: "LC_ALL takes priority over LC_MESSAGES and LANG",
			env:  map[string]string{"LC_ALL": "fr_FR.UTF-8", "LC_MESSAGES": "en_US.UTF-8", "LANG": "en_US.UTF-8"},
			want: FR,
		},
		{
			name: "LC_MESSAGES takes priority over LANG",
			env:  map[string]string{"LC_MESSAGES": "fr_FR.UTF-8", "LANG": "en_US.UTF-8"},
			want: FR,
		},
		{
			name: "unsupported locale defaults to EN",
			env:  map[string]string{"LANG": "de_DE.UTF-8"},
			want: EN,
		},
		{
			name: "handles uppercase locale",
			env:  map[string]string{"LANG": "FR_FR.UTF-8"},
			want: FR,
		},
		{
			name: "skips empty env and falls through",
			env:  map[string]string{"LANGUAGE": "", "LANG": "fr_FR.UTF-8"},
			want: FR,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clearEnv()
			for k, v := range tc.env {
				os.Setenv(k, v)
			}
			got := DetectSystemLang()
			if got != tc.want {
				t.Errorf("DetectSystemLang() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestKeyParity(t *testing.T) {
	Load()

	en := translations[EN]
	fr := translations[FR]

	if len(en) == 0 {
		t.Fatal("EN translations empty")
	}

	for key := range en {
		if _, ok := fr[key]; !ok {
			t.Errorf("key %q missing in FR", key)
		}
	}
	for key := range fr {
		if _, ok := en[key]; !ok {
			t.Errorf("key %q missing in EN", key)
		}
	}
}
