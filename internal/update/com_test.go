package update

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The launcher reads the name comText writes with its own copy of the
// marker (launcher/args.go): the two are one.
func TestComMagicIsTheLaunchers(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("launcher", "args.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `const magic = "\x00jianmu-com-target"`) || comMagic != "\x00jianmu-com-target" {
		t.Fatal("jianmu.com's marker differs between the app and the launcher")
	}
}

// #3737: a shell doesn't wait for the GUI-linked jianmu.exe, so jianmu
// writes jianmu.com (run first by PATHEXT) beside it, naming the app it
// runs; written once, again only when its bytes differ (an update).
func TestWriteCom(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "jianmu-windows-amd64.exe")
	if err := os.WriteFile(exe, []byte("MZ"), 0o755); err != nil {
		t.Fatal(err)
	}
	launcher := []byte("MZ launcher v1")
	p, err := writeCom(launcher, exe)
	if err != nil || p != filepath.Join(dir, "jianmu.com") {
		t.Fatalf("writeCom: %q, %v", p, err)
	}
	b, _ := os.ReadFile(p)
	if !bytes.HasPrefix(b, launcher) || comTarget(b) != "jianmu-windows-amd64.exe" {
		t.Fatalf("jianmu.com: %q", b)
	}
	if !comRuns(p, exe) || comRuns(p, filepath.Join(dir, "jianmu.exe")) {
		t.Fatal("jianmu.com not read as running this app alone")
	}
	st1, _ := os.Stat(p)
	if _, err := writeCom(launcher, exe); err != nil {
		t.Fatal(err)
	}
	if st2, _ := os.Stat(p); !os.SameFile(st1, st2) {
		t.Fatal("an up-to-date jianmu.com written again")
	}
	// the next version's launcher
	if _, err := writeCom([]byte("MZ launcher v2"), exe); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); !bytes.HasPrefix(b, []byte("MZ launcher v2")) {
		t.Fatalf("jianmu.com not brought up to date: %q", b)
	}
	// one moved aside while it ran goes on the next run
	if err := os.WriteFile(p+".old", []byte("MZ launcher v1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := writeCom([]byte("MZ launcher v2"), exe); err != nil || exists(p+".old") {
		t.Fatalf("jianmu.com.old left: %v", err)
	}
	if left, _ := filepath.Glob(filepath.Join(dir, ".jianmu-com-*")); len(left) > 0 {
		t.Fatalf("temp files left: %q", left)
	}
}

func TestComOwner(t *testing.T) {
	dir := t.TempDir()
	dl := filepath.Join(dir, "jianmu-windows-amd64.exe")
	if !comOwner(dl) {
		t.Fatal("a lone download doesn't own jianmu.com")
	}
	if err := os.WriteFile(filepath.Join(dir, "jianmu.exe"), []byte("MZ"), 0o755); err != nil {
		t.Fatal(err)
	}
	if comOwner(dl) {
		t.Fatal("a copy beside jianmu.exe takes `jianmu` from it")
	}
	if !comOwner(filepath.Join(dir, "Jianmu.exe")) {
		t.Fatal("jianmu.exe doesn't own its jianmu.com")
	}
}

// EnsureCompanion writes where a terminal runs jianmu from, and nowhere
// else: a downloads folder off PATH gets nothing.
func TestEnsureCompanion(t *testing.T) {
	old, wasDL := comLauncher, downloadFolder
	t.Cleanup(func() { comLauncher, downloadFolder = old, wasDL })
	// on Windows a temporary folder counts as a download folder, which
	// gets one only from Add (TestNoCompanionPutBackOnTheDesktop)
	downloadFolder = func(string) bool { return false }
	sep := string(os.PathListSeparator)

	comLauncher = nil
	dir := t.TempDir()
	exe := filepath.Join(dir, "jianmu.exe")
	t.Setenv("PATH", dir)
	if err := EnsureCompanion(exe); err != nil || exists(filepath.Join(dir, comName)) {
		t.Fatalf("a build with no launcher wrote jianmu.com: %v", err)
	}

	comLauncher = []byte("MZ launcher")
	off := t.TempDir()
	t.Setenv("PATH", "/x"+sep+dir+string(filepath.Separator))
	if err := EnsureCompanion(filepath.Join(off, "jianmu.exe")); err != nil || exists(filepath.Join(off, comName)) {
		t.Fatalf("jianmu.com written off PATH: %v", err)
	}
	if err := EnsureCompanion(exe); err != nil || !comRuns(filepath.Join(dir, comName), exe) {
		t.Fatalf("jianmu.com not written in a folder on PATH: %v", err)
	}

	// off PATH, but a jianmu.cmd says a terminal runs it from there
	shim := t.TempDir()
	dl := filepath.Join(shim, "jianmu-windows-arm64.exe")
	if err := os.WriteFile(filepath.Join(shim, shimName), []byte("@echo off\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureCompanion(dl); err != nil || !comRuns(filepath.Join(shim, comName), dl) {
		t.Fatalf("jianmu.com not written beside a jianmu.cmd: %v", err)
	}
}

// findCLI names the program a shell runs: jianmu.com, before jianmu.exe
// in the same folder.
func TestFindCLIPrefersCom(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PATHEXT is Windows'")
	}
	dir := t.TempDir()
	for _, n := range []string{"jianmu.exe", "jianmu.com"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("MZ"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if got := findCLI([]string{dir}); !strings.EqualFold(got, filepath.Join(dir, "jianmu.com")) {
		t.Fatalf("findCLI: %q", got)
	}
}
