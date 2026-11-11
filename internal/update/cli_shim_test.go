package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #942: the site's Windows download is jianmu-windows-amd64.exe, and Add
// to PATH put its folder on PATH, where a terminal finds no `jianmu`. Add
// writes a jianmu.cmd beside a program not named jianmu.exe that runs it by
// its folder and name, which an update keeps, and the Command line reads
// that jianmu.cmd as this app; one written for another program isn't.
func TestCLIShim(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "jianmu-windows-amd64.exe")
	if err := os.WriteFile(exe, []byte("MZ"), 0o755); err != nil {
		t.Fatal(err)
	}
	p, err := writeShim(exe)
	if err != nil {
		t.Fatal(err)
	}
	if p != filepath.Join(dir, "jianmu.cmd") {
		t.Fatalf("shim at %q", p)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "\"%~dp0jianmu-windows-amd64.exe\" %*\r\n") || !strings.HasPrefix(string(b), "@echo off\r\n") {
		t.Fatalf("jianmu.cmd:\n%q", b)
	}
	if !shimRuns(p, exe) {
		t.Fatal("the jianmu.cmd written isn't read as running this app")
	}
	if shimRuns(p, filepath.Join(dir, "jianmu-old.exe")) {
		t.Fatal("jianmu.cmd read as running another program")
	}
	// renamed, as a second download is: written again for the new name
	exe2 := filepath.Join(dir, "jianmu-windows-amd64 (1).exe")
	if _, err := writeShim(exe2); err != nil {
		t.Fatal(err)
	}
	if !shimRuns(p, exe2) || shimRuns(p, exe) {
		t.Fatal("jianmu.cmd not written again for the renamed program")
	}

	// jianmu.exe is found by its own name: nothing written
	dir2 := t.TempDir()
	if p, err := writeShim(filepath.Join(dir2, "Jianmu.exe")); err != nil || p != "" {
		t.Fatalf("jianmu.exe given a shim: %q, %v", p, err)
	}
	if _, err := os.Stat(filepath.Join(dir2, "jianmu.cmd")); !os.IsNotExist(err) {
		t.Fatal("jianmu.cmd written beside jianmu.exe")
	}
}
