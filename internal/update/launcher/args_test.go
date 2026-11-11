package main

import (
	"os"
	"path/filepath"
	"testing"
)

// #3737: jianmu.com hands the app what was typed after `jianmu` as it
// was typed, runs the program its trailer names, and leaves a command that
// opens the window to run on its own.
func TestArgsOf(t *testing.T) {
	for line, want := range map[string]string{
		`jianmu`:              ``,
		`jianmu plugin login`: `plugin login`,
		`jianmu   version`:    `version`,
		`"C:\Program Files\m\jianmu.com" "a b" c`: `"a b" c`,
		`"C:\m\jianmu" x`:                         `x`,
		`C:\m\jianmu.com	-v`:                      `-v`,
		`"unclosed`:                               ``,
	} {
		if got := argsOf(line); got != want {
			t.Errorf("argsOf(%q) = %q, want %q", line, got, want)
		}
	}
}

func TestOpensWindow(t *testing.T) {
	for _, a := range [][]string{nil, {"app"}, {"gui"}, {"tray"}, {"panel"}, {"-Embedding"}, {"JIANMU://open"}} {
		if !opensWindow(a) {
			t.Errorf("%q waited for as a terminal command", a)
		}
	}
	for _, a := range [][]string{{"version"}, {"plugin", "login"}, {"serve"}, {"backup", "app"}} {
		if opensWindow(a) {
			t.Errorf("%q not waited for", a)
		}
	}
}

func TestTargetName(t *testing.T) {
	dir := t.TempDir()
	com := filepath.Join(dir, "jianmu.com")
	body := make([]byte, 4096)
	write := func(b []byte) {
		t.Helper()
		if err := os.WriteFile(com, b, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write(append(append(append(body, 0), "jianmu-windows-amd64 (1).exe"...), magic...))
	if got := targetName(com); got != "jianmu-windows-amd64 (1).exe" {
		t.Fatalf("trailer's name: %q", got)
	}
	// no trailer, or one naming a path: its own name with .exe
	write(body)
	if got := targetName(com); got != "jianmu.exe" {
		t.Fatalf("no trailer: %q", got)
	}
	write(append(append(append(body, 0), "../evil.exe"...), magic...))
	if got := targetName(com); got != "jianmu.exe" {
		t.Fatalf("a path in the trailer: %q", got)
	}
}
