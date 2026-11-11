package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/DamonZS/Jianmu/internal/catalog"
	"github.com/DamonZS/Jianmu/internal/settings"
)

// A Remote jianmu's list says which jianmu listed each model (the last of
// its jianmu_via). One this jianmu listed is this jianmu: the provider is
// left out, and the record outlives the run (Self is made up again each
// one) until a list another jianmu gives, or one from before jianmu_via,
// says otherwise. An empty list or a failed fetch changes nothing, and a
// provider moved to another address is not this jianmu there.
func TestRemoteSelfRecord(t *testing.T) {
	isolate(t)
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("USERPROFILE", h)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(h, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(h, ".cache"))
	var answer atomic.Value // the list the remote gives
	var down atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if down.Load() {
			http.Error(w, "down", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, answer.Load().(string))
	}))
	t.Cleanup(srv.Close)
	list := func(lister string) string {
		via := ""
		if lister != "" {
			via = `,"jianmu_via":["` + lister + `"]`
		}
		return `{"object":"list","data":[{"id":"own/m1","object":"model"` + via + `}]}`
	}
	id, err := Add(Provider{ID: "rem", Name: "Rem", Preset: RemoteJianmuPreset, Chat: strings.TrimPrefix(srv.URL, "http://")})
	if err != nil {
		t.Fatal(err)
	}
	fetch := func() Provider {
		t.Helper()
		p, err := Find(id)
		if err != nil {
			t.Fatal(err)
		}
		p.Fetch(context.Background())
		p, _ = Find(id)
		return *p
	}
	answer.Store(list("another"))
	if p := fetch(); p.SelfAnswered() || !p.On() {
		t.Fatalf("another jianmu's list: self %v, on %v", p.SelfAnswered(), p.On())
	}
	if _, err := os.Stat(filepath.Join(settings.Dir(), "remote-self.json")); err == nil {
		t.Error("another jianmu's list wrote a record")
	}
	answer.Store(list(catalog.Self()))
	if p := fetch(); !p.SelfAnswered() || !p.SelfLeftOut() || p.On() {
		t.Fatalf("this jianmu's list: self %v, left out %v, on %v", p.SelfAnswered(), p.SelfLeftOut(), p.On())
	}
	if got := ThroughSelf("rem/own/m1"); got != "own/m1" {
		t.Errorf("ThroughSelf(rem/own/m1) = %q, want own/m1", got)
	}
	if got := ThroughSelf("rem/rem/own/m1"); got != "own/m1" {
		t.Errorf("ThroughSelf(rem/rem/own/m1) = %q, want own/m1", got)
	}
	if got := ThroughSelf("other/m1"); got != "other/m1" {
		t.Errorf("ThroughSelf(other/m1) = %q", got)
	}
	if p, ok := SwitchedOff("rem/x/y"); !ok || p.ID != "rem" {
		t.Errorf("SwitchedOff(rem/x/y) = %q %v; want the left-out remote", p.ID, ok)
	}

	// the record is read again in another run, from the file
	remoteSelf.Lock()
	remoteSelf.path = ""
	remoteSelf.Unlock()
	if p, _ := Find(id); !p.SelfAnswered() {
		t.Fatal("the record didn't outlive the run")
	}
	// an empty list and a failed fetch tell nothing
	answer.Store(`{"object":"list","data":[]}`)
	if p := fetch(); !p.SelfAnswered() {
		t.Error("an empty list took the record away")
	}
	down.Store(true)
	if p := fetch(); !p.SelfAnswered() {
		t.Error("a failed fetch took the record away")
	}
	down.Store(false)
	// moved to another address: not this jianmu there
	p, _ := Find(id)
	moved := *p
	moved.Chat = "http://192.0.2.1:3425/v1"
	if moved.SelfAnswered() {
		t.Error("a provider at another address is taken for this jianmu")
	}
	// a jianmu from before jianmu_via names none: no longer this one
	answer.Store(list(""))
	if p := fetch(); p.SelfAnswered() || !p.On() {
		t.Errorf("a list from before jianmu_via: self %v, on %v", p.SelfAnswered(), p.On())
	}
	if got := ThroughSelf("rem/own/m1"); got != "rem/own/m1" {
		t.Errorf("ThroughSelf after the record went = %q", got)
	}
}

// Use it anyway is this computer's own: sync brings the provider from the
// computer it is not this jianmu on, and a backup put back here keeps it.
func TestMirrorKeepsKeepSelf(t *testing.T) {
	isolate(t)
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("USERPROFILE", h)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(h, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(h, ".cache"))
	if err := Save(Provider{ID: "rem", Name: "Rem", Preset: RemoteJianmuPreset, Chat: "http://192.0.2.1:3425/v1", Key: "k", KeepSelf: true}); err != nil {
		t.Fatal(err)
	}
	other := Provider{ID: "rem", Name: "Rem renamed", Preset: RemoteJianmuPreset, Chat: "http://192.0.2.1:3425/v1", Key: "k"}
	fresh := Provider{ID: "rem2", Name: "Rem 2", Preset: RemoteJianmuPreset, Chat: "http://192.0.2.2:3425/v1", Key: "k", KeepSelf: true}
	if err := Mirror([]Provider{other, fresh}, nil); err != nil {
		t.Fatal(err)
	}
	ps, _ := Stored()
	if len(ps) != 2 || ps[0].Name != other.Name || !ps[0].KeepSelf || ps[1].KeepSelf {
		for _, p := range ps {
			t.Logf("%s %q KeepSelf %v", p.ID, p.Name, p.KeepSelf)
		}
		t.Fatal("mirrored: want the other's name, this computer's KeepSelf, none on the new one")
	}
	if _, _, err := Restore([]Provider{other, {ID: "rem3", Name: "Rem 3", Preset: RemoteJianmuPreset, Chat: "http://192.0.2.3:3425/v1", Key: "k", KeepSelf: true}}); err != nil {
		t.Fatal(err)
	}
	ps, _ = Stored()
	for _, p := range ps {
		if p.ID == "rem" && !p.KeepSelf || p.ID == "rem3" && p.KeepSelf {
			t.Errorf("restored %s: KeepSelf %v", p.ID, p.KeepSelf)
		}
	}
}
