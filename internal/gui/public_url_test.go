package gui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DamonZS/Jianmu/internal/fx"
	"github.com/DamonZS/Jianmu/internal/gateway"
	"github.com/DamonZS/Jianmu/internal/settings"
	"github.com/DamonZS/Jianmu/internal/testenv"
)

// Settings shows the local address agents use; the Gateway page advertises
// the public URL for its copy buttons and snippets.
func TestPublicURLInConsole(t *testing.T) {
	h := t.TempDir()
	testenv.SetHome(t, h)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(h, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(h, ".cache"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(h, ".local", "share"))
	t.Setenv("JIANMU_ADDR", "0.0.0.0:3425")
	t.Setenv("JIANMU_PUBLIC_URL", "https://jianmu.example.com/jianmu///")

	// Seed the rate the Settings page reads, so it needs no network.
	cache := fx.CachePath()
	if err := os.MkdirAll(filepath.Dir(cache), 0o755); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(fx.Rate{CNYPerUSD: 7.25, At: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cache, b, 0o644); err != nil {
		t.Fatal(err)
	}
	fx.Reset()
	t.Cleanup(fx.Reset)

	if err := settings.Save(settings.Settings{LAN: true, LANKey: "sk-jianmu-test"}); err != nil {
		t.Fatal(err)
	}
	want := "https://jianmu.example.com/jianmu"
	if got := settingsState().Gateway; got != gateway.URL() {
		t.Errorf("Settings: %q, want %q", got, gateway.URL())
	}
	g := providersState().Gateway
	if g.URL != gateway.URL() {
		t.Errorf("Gateway address: %q, want the local %q", g.URL, gateway.URL())
	}
	if len(g.LANURLs) != 1 || g.LANURLs[0] != want {
		t.Errorf("LAN addresses: %v, want [%s]", g.LANURLs, want)
	}
	if !g.OnNetwork {
		t.Error("JIANMU_ADDR 0.0.0.0 not said to be on the network")
	}
	// #1551: Connect starts at the public URL, and Settings shows it
	if g.PublicURL != want {
		t.Errorf("Gateway page's public URL: %q, want %q", g.PublicURL, want)
	}
	if got := settingsState().PublicURL; got != want {
		t.Errorf("Settings' public URL: %q, want %q", got, want)
	}
	b, err = json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	var state map[string]any
	if err := json.Unmarshal(b, &state); err != nil {
		t.Fatal(err)
	}
	if state["lan"] != true {
		t.Fatalf("LAN state missing from gateway page: %s", b)
	}
}

// A gateway JIANMU_ADDR puts on the network (the Docker image) asks other
// machines for a gateway key whether or not it is shared, so the Gateway
// page shows its keys and offers no key-less "this computer" then.
func TestGatewayOnNetworkInConsole(t *testing.T) {
	h := t.TempDir()
	testenv.SetHome(t, h)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(h, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(h, ".cache"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(h, ".local", "share"))
	t.Setenv("JIANMU_ADDR", "0.0.0.0:3425")
	t.Setenv("JIANMU_PUBLIC_URL", "")
	if err := settings.Save(settings.Settings{LAN: false}); err != nil {
		t.Fatal(err)
	}
	if g := providersState().Gateway; !g.OnNetwork || !g.LAN {
		t.Errorf("gateway on the network, not shared, shown without its keys: %+v", g)
	}
	if err := settings.Save(settings.Settings{LAN: true}); err != nil {
		t.Fatal(err)
	}
	if g := providersState().Gateway; !g.OnNetwork || !g.LAN {
		t.Errorf("gateway on the network, shared: %+v", g)
	}
	t.Setenv("JIANMU_ADDR", "127.0.0.1:3425")
	if err := settings.Save(settings.Settings{LAN: false}); err != nil {
		t.Fatal(err)
	}
	if g := providersState().Gateway; g.OnNetwork || g.LAN {
		t.Errorf("gateway on loopback, not shared: %+v", g)
	}
}

// #1551: JIANMU_PUBLIC_URL is offered where other machines are let in,
// and nowhere else: on loopback, not shared, nothing reaches the gateway
// through it without a key it can't take.
func TestPublicURLOnlyWhereOffered(t *testing.T) {
	h := t.TempDir()
	testenv.SetHome(t, h)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(h, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(h, ".cache"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(h, ".local", "share"))
	t.Setenv("JIANMU_PUBLIC_URL", "https://jianmu.example.com")
	t.Setenv("JIANMU_ADDR", "127.0.0.1:3425")
	if err := settings.Save(settings.Settings{LAN: false}); err != nil {
		t.Fatal(err)
	}
	if g := providersState().Gateway; g.PublicURL != "" {
		t.Errorf("loopback, not shared: Gateway page offers %q", g.PublicURL)
	}
	if u := settingsState().PublicURL; u != "" {
		t.Errorf("loopback, not shared: Settings shows %q", u)
	}
	if err := settings.Save(settings.Settings{LAN: true}); err != nil {
		t.Fatal(err)
	}
	if g := providersState().Gateway; g.PublicURL != "https://jianmu.example.com" {
		t.Errorf("shared: Gateway page's public URL %q", g.PublicURL)
	}
	t.Setenv("JIANMU_PUBLIC_URL", "")
	if g := providersState().Gateway; g.PublicURL != "" {
		t.Errorf("no JIANMU_PUBLIC_URL: Gateway page's public URL %q", g.PublicURL)
	}
	if u := settingsState().PublicURL; u != "" {
		t.Errorf("no JIANMU_PUBLIC_URL: Settings shows %q", u)
	}
}

// jiezhengj on Discord: jianmu in Docker on a NAS, JIANMU_PUBLIC_URL at
// the NAS's 192.168.31.218, and the page called it "Public address"
// (公网地址) as if anyone online reached it. Settings and the Gateway page
// say what kind of address it is, and a private one is the local
// network's.
func TestPublicURLSaysItsKind(t *testing.T) {
	h := t.TempDir()
	testenv.SetHome(t, h)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(h, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(h, ".cache"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(h, ".local", "share"))
	t.Setenv("JIANMU_ADDR", "127.0.0.1:3425")
	if err := settings.Save(settings.Settings{LAN: true}); err != nil {
		t.Fatal(err)
	}
	kind := func(v any) any {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}
		return m["publicKind"]
	}
	for u, want := range map[string]any{
		"http://192.168.31.218:3425": "local",
		"http://172.17.0.1:3425":     "local",
		"http://203.0.113.9:3425":    "internet",
		"https://jianmu.example.com": nil,
	} {
		t.Setenv("JIANMU_PUBLIC_URL", u)
		if got := kind(providersState().Gateway); got != want {
			t.Errorf("%s: Gateway page's publicKind %v, want %v", u, got, want)
		}
		if got := kind(settingsState()); got != want {
			t.Errorf("%s: Settings' publicKind %v, want %v", u, got, want)
		}
	}
}
