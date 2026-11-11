package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// pacman's local database as the AUR's jianmu-bin 0.1.2342-1 leaves it
// (its PKGBUILD installs /usr/bin/jianmu), beside a package that doesn't
// own jianmu.
func pacmanRoot(t *testing.T) string {
	root := t.TempDir()
	local := filepath.Join(root, "var/lib/pacman/local")
	writeFile(t, filepath.Join(local, "ALPM_DB_VERSION"), "9\n")
	writeFile(t, filepath.Join(local, "bash-5.3.3-2/files"), "%FILES%\nusr/\nusr/bin/\nusr/bin/bash\nusr/bin/sh\n\n%BACKUP%\netc/bash.bashrc\t0b5ab6a3\n")
	writeFile(t, filepath.Join(local, "bash-5.3.3-2/desc"), "%NAME%\nbash\n\n%VERSION%\n5.3.3-2\n")
	writeFile(t, filepath.Join(local, "jianmu-bin-0.1.2342-1/files"), "%FILES%\nusr/\nusr/bin/\nusr/bin/jianmu\nusr/share/\nusr/share/applications/\nusr/share/applications/jianmu.desktop\nusr/share/icons/hicolor/256x256/apps/jianmu.png\nusr/share/licenses/jianmu-bin/LICENSE\n\n")
	writeFile(t, filepath.Join(local, "jianmu-bin-0.1.2342-1/desc"), "%NAME%\njianmu-bin\n\n%VERSION%\n0.1.2342-1\n\n%BASE%\njianmu-bin\n")
	return root
}

func TestPackagedByPacman(t *testing.T) {
	root := pacmanRoot(t)
	p := packagedIn(root, "linux", "/usr/bin/jianmu")
	if p == nil || p.Manager != "pacman" || p.Package != "jianmu-bin" {
		t.Fatalf("/usr/bin/jianmu from jianmu-bin: %+v", p)
	}
	if c := p.Command(); c != "yay -S jianmu-bin" {
		t.Fatalf("command %q", c)
	}
	if s := p.Say(); !strings.Contains(s, "pacman") || !strings.Contains(s, "yay -S jianmu-bin") {
		t.Fatalf("said %q", s)
	}
	// /bin is /usr/bin on Arch: the same file
	if p := packagedIn(root, "linux", "/bin/jianmu"); p == nil || p.Package != "jianmu-bin" {
		t.Fatalf("/bin/jianmu: %+v", p)
	}
	// a copy in /usr/local, ~/.local/bin or /opt that no package lists is
	// the user's own, and updates itself
	for _, exe := range []string{"/usr/local/bin/jianmu", "/home/u/.local/bin/jianmu", "/opt/jianmu/jianmu", "/usr/lib/jianmu/jianmu"} {
		if p := packagedIn(root, "linux", exe); p != nil {
			t.Fatalf("%s: %+v", exe, p)
		}
	}
	// not Linux: pacman's files aren't read
	if p := packagedIn(root, "darwin", "/usr/bin/jianmu"); p != nil {
		t.Fatalf("darwin: %+v", p)
	}
}

// A package named otherwise is found too, its name from the folder when
// its desc can't be read.
func TestPackagedByPacmanOtherName(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "var/lib/pacman/local/ai-tools-2.0-3/files"), "%FILES%\nopt/ai/bin/jianmu\n")
	p := packagedIn(root, "linux", "/opt/ai/bin/jianmu")
	if p == nil || p.Package != "ai-tools" {
		t.Fatalf("%+v", p)
	}
}

// dpkg lists absolute paths, and on a merged /usr may list /bin for the
// /usr/bin the binary resolves to.
func TestPackagedByDpkg(t *testing.T) {
	root := t.TempDir()
	info := filepath.Join(root, "var/lib/dpkg/info")
	writeFile(t, filepath.Join(info, "coreutils.list"), "/.\n/bin\n/bin/ls\n")
	writeFile(t, filepath.Join(info, "jianmu:amd64.list"), "/.\n/usr\n/usr/bin\n/usr/bin/jianmu\n")
	p := packagedIn(root, "linux", "/usr/bin/jianmu")
	if p == nil || p.Manager != "dpkg" || p.Package != "jianmu" || p.Name() != "apt" || p.Command() != "sudo apt install --only-upgrade jianmu" {
		t.Fatalf("%+v", p)
	}
	if p := packagedIn(root, "linux", "/usr/bin/ls"); p == nil || p.Package != "coreutils" {
		t.Fatalf("merged /usr: %+v", p)
	}
	if p := packagedIn(root, "linux", "/usr/sbin/jianmu"); p != nil {
		t.Fatalf("not listed: %+v", p)
	}
}

func TestPackagedByRpm(t *testing.T) {
	root := t.TempDir()
	was := rpmOwner
	t.Cleanup(func() { rpmOwner = was })
	asked := ""
	rpmOwner = func(path string) string { asked = path; return "jianmu" }
	// no rpm database: rpm isn't asked
	if p := packagedIn(root, "linux", "/usr/bin/jianmu"); p != nil || asked != "" {
		t.Fatalf("no rpm here: %+v, asked %q", p, asked)
	}
	if err := os.MkdirAll(filepath.Join(root, "var/lib/rpm"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := packagedIn(root, "linux", "/usr/bin/jianmu")
	if p == nil || p.Manager != "rpm" || asked != "/usr/bin/jianmu" || p.Command() != "sudo dnf upgrade jianmu" {
		t.Fatalf("%+v, asked %q", p, asked)
	}
	rpmOwner = func(string) string { return "" }
	if p := packagedIn(root, "linux", "/usr/bin/jianmu"); p != nil {
		t.Fatalf("not owned: %+v", p)
	}
}

func TestPackagedByHomebrewAndNix(t *testing.T) {
	root := t.TempDir()
	for exe, want := range map[string]string{
		"/opt/homebrew/Cellar/jianmu/0.1.126/bin/jianmu":              "homebrew jianmu brew upgrade jianmu",
		"/home/linuxbrew/.linuxbrew/Cellar/jianmu/0.1.126/bin/jianmu": "homebrew jianmu brew upgrade jianmu",
		"/nix/store/abc123-jianmu-0.1.2342/bin/jianmu":                "nix  ",
	} {
		goos := "linux"
		if strings.HasPrefix(exe, "/opt/homebrew") {
			goos = "darwin"
		}
		p := packagedIn(root, goos, exe)
		if p == nil || p.Manager+" "+p.Package+" "+p.Command() != want {
			t.Fatalf("%s: %+v", exe, p)
		}
	}
	if p := packagedIn(root, "darwin", "/Applications/jianmu.app/Contents/MacOS/jianmu"); p != nil {
		t.Fatalf("the app: %+v", p)
	}
}
