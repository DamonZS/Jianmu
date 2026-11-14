package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// A plugin written before jianmu renamed its package.json fields keeps them
// in a magpie block; jianmu reads that block the same way, and jianmu's own
// block wins where a package has both, so the community's migrated packages
// and older installs both work.
func TestLegacyMetadataKeepsWorking(t *testing.T) {
	root := t.TempDir()
	mk := func(name, pkg string, files ...string) string {
		t.Helper()
		d := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Join(d, "page"), 0o755); err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			if err := os.WriteFile(filepath.Join(d, filepath.FromSlash(f)), []byte("<p>hi</p>"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(d, "package.json"), []byte(pkg), 0o644); err != nil {
			t.Fatal(err)
		}
		return d
	}
	agent := mk("agent", `{"name":"a","magpie":{"agent":"a.agent.js"}}`)
	if f, _ := Agent(agent); filepath.Base(f) != "a.agent.js" {
		t.Errorf("agent in magpie: %q", f)
	}
	hooks := mk("hooks", `{"name":"h","magpie":{"hooks":"h.hooks.mjs"}}`)
	if f := Gateway(hooks); filepath.Base(f) != "h.hooks.mjs" {
		t.Errorf("hooks in magpie: %q", f)
	}
	mw := mk("mw", `{"name":"m","magpie":{"middleware":"m.middleware.js"}}`)
	if f, only := Middleware(mw); filepath.Base(f) != "m.middleware.js" || !only {
		t.Errorf("middleware in magpie: %q only=%v", f, only)
	}
	mk("opts", `{"name":"o","magpie":{"options":{"a":1}}}`)
	if o := OptionsExample(filepath.Join(root, "opts")); o["a"] != float64(1) {
		t.Errorf("options in magpie: %v", o)
	}
	page := mk("page", `{"name":"p","magpie":{"page":{"title":"Old","file":"page/index.html"}}}`, "page/index.html")
	if w, _ := Window(page); w == nil || w.Page == nil || w.Page.Title != "Old" {
		t.Errorf("page in magpie: %+v", w)
	}
	// both blocks: jianmu's says which file, the old one names another
	both := mk("both", `{"name":"b","magpie":{"agent":"old.agent.js"},"jianmu":{"agent":"new.agent.js"}}`)
	if f, _ := Agent(both); filepath.Base(f) != "new.agent.js" {
		t.Errorf("jianmu's block wins: %q", f)
	}
	// npm's copy: its words are in either block too
	for _, tc := range []struct{ jianmu, magpie, want string }{
		{`{"name":"New"}`, `{"name":"Old"}`, "New"},
		{``, `{"name":"Old"}`, "Old"},
		{`{}`, `{"name":"Old"}`, "Old"},
	} {
		l := npmLatest{Jianmu: hostWords{}, Magpie: hostWords{}}
		_ = json.Unmarshal([]byte(tc.jianmu), &l.Jianmu)
		_ = json.Unmarshal([]byte(tc.magpie), &l.Magpie)
		if got := l.words().Name; got != tc.want {
			t.Errorf("words(%s, %s) = %q, want %q", tc.jianmu, tc.magpie, got, tc.want)
		}
	}
}
