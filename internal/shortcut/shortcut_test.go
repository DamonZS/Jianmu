package shortcut

import "testing"

func TestTarget(t *testing.T) {
	for in, want := range map[string]string{
		`C:\Tools\jianmu-windows-amd64.exe`:          `C:\Tools\jianmu-windows-amd64.exe`,
		`C:\Tools\jianmu.exe.old`:                    `C:\Tools\jianmu.exe`,
		`C:\Tools\jianmu.exe.old-3`:                  `C:\Tools\jianmu.exe`,
		`C:\Tools\jianmu.exe.new`:                    `C:\Tools\jianmu.exe`,
		`C:\Tools\JIANMU.EXE.OLD`:                    `C:\Tools\JIANMU.EXE`,
		`C:\old\jianmu.exe`:                          `C:\old\jianmu.exe`,
		`C:\Tools\jianmu.exe.older`:                  `C:\Tools\jianmu.exe.older`,
		`/usr/local/bin/jianmu`:                      `/usr/local/bin/jianmu`,
		`C:\Users\me\Downloads\jianmu (1).exe.old-2`: `C:\Users\me\Downloads\jianmu (1).exe`,
	} {
		if got := Target(in); got != want {
			t.Errorf("Target(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStale(t *testing.T) {
	exe := "/tools/jianmu.exe"
	for _, c := range []struct {
		target, dir string
		stale       bool
	}{
		{"", "", true},                            // no shortcut yet
		{"/tools/jianmu.exe", "/tools", false},    // already this one
		{"/TOOLS/Jianmu.EXE", "/Tools/", false},   // the same, in another case
		{"/downloads/jianmu.exe", "/tools", true}, // the exe was moved
		{"/tools/jianmu.exe", "/downloads", true}, // starts in another folder
		{"/tools/jianmu.exe", "", true},           // in no folder
	} {
		if got := Stale(c.target, c.dir, exe); got != c.stale {
			t.Errorf("Stale(%q, %q) = %v, want %v", c.target, c.dir, got, c.stale)
		}
	}
}

// Installed apps runs the exe it names, quoted, with the arguments the
// CLI opens the window on (the path has spaces under a user's name).
func TestUninstallCommand(t *testing.T) {
	got := UninstallCommand(`C:\Users\Ann Lee\AppData\Local\Programs\jianmu\jianmu.exe`)
	want := `"C:\Users\Ann Lee\AppData\Local\Programs\jianmu\jianmu.exe" uninstall --window`
	if got != want {
		t.Fatalf("UninstallCommand = %s, want %s", got, want)
	}
}
