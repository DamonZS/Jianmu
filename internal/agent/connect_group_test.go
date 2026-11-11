package agent

import (
	"path/filepath"
	"testing"

	"github.com/DamonZS/Jianmu/internal/provider"
	"github.com/DamonZS/Jianmu/internal/settings"
)

// An agent whose only models through jianmu are routing groups is
// connected on the first of them (#939: the switch said jianmu had no
// models Codex could use while `jianmu codex group/folia-1` connected it).
func TestConnectOnARoutingGroupAlone(t *testing.T) {
	home, _ := codexHome(t, "", "")
	if err := provider.Save(provider.Provider{ID: "aaa", Name: "AAA", Chat: "https://a.example/v1", Key: "k", Models: []string{"m1"}}); err != nil {
		t.Fatal(err)
	}
	if err := provider.SaveGroup(provider.Group{ID: "folia-1", Name: "folia-1", Members: []string{"aaa/m1"}}); err != nil {
		t.Fatal(err)
	}
	s := settings.Load()
	s.HiddenModels = map[string][]string{"claude": {"aaa/m1", "fake/m1"}}
	if err := settings.Save(s); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".claude", "settings.json")
	writeFile(t, path, `{}`)
	c := claude(home)
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	if m := writtenModel(path); m != "group/folia-1" {
		t.Fatalf("connected on %q:\n%s", m, readFile(path))
	}
}
