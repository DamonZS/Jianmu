//go:build !nogui

package gui

import (
	"path/filepath"
	"testing"

	"github.com/DamonZS/Jianmu/internal/settings"
)

func TestProductNameFollowsSavedLanguage(t *testing.T) {
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("USERPROFILE", h)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(h, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(h, ".cache"))
	for _, lang := range []string{"zh", "en", "zh-TW", "ja", "de", "zh"} {
		if err := settings.Save(settings.Settings{Lang: lang}); err != nil {
			t.Fatal(err)
		}
		want := "jianmu"
		if lang == "zh" || lang == "zh-TW" {
			want = "建木"
		}
		if got := productName(); got != want {
			t.Errorf("%s product name = %q, want %q", lang, got, want)
		}
		if got := panelOptions("windows", "").Title; got != want {
			t.Errorf("%s panel title = %q, want %q", lang, got, want)
		}
	}
}
