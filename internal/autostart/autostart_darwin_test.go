package autostart

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// what jianmu up to now wrote for Open at login
const olderPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key><string>com.jianmu</string>
	<key>ProgramArguments</key>
	<array><string>/Applications/Tom &amp; Jerry/jianmu.app/Contents/MacOS/jianmu</string><string>tray</string></array>
	<key>RunAtLoad</key><true/>
	<key>LimitLoadToSessionType</key><string>Aqua</string>
	<key>ProcessType</key><string>Interactive</string>
</dict>
</plist>
`

// launchd kills what a job started once the job exits, so the shell a
// restart to update leaves to open the new version died with a jianmu
// opened at login: the launch agent lets them go.
func TestLaunchAgentLetsTheRelaunchLive(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := Set(true); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(record())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "<key>AbandonProcessGroup</key><true/>") {
		t.Fatalf("the launch agent doesn't abandon its process group:\n%s", b)
	}
	if out, err := exec.Command("plutil", "-lint", record()).CombinedOutput(); err != nil {
		t.Fatalf("not a valid plist: %s", out)
	}
}

// An older jianmu's launch agent is written again as this one writes it,
// for the program it named; one that's off stays off.
func TestRefreshOlderLaunchAgent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := Refresh(); err != nil || Enabled() != Off {
		t.Fatalf("Refresh turned Open at login on: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "Library")); !os.IsNotExist(err) {
		t.Fatal("Refresh made the LaunchAgents folder for one that's off")
	}

	p := record()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(olderPlist), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Refresh(); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	want := plist("/Applications/Tom & Jerry/jianmu.app/Contents/MacOS/jianmu")
	if string(b) != string(want) {
		t.Fatalf("not rewritten as this version writes it:\n%s\nwant:\n%s", b, want)
	}

	// one jianmu didn't write is left as it is
	other := "<plist><dict><key>Label</key><string>x</string></dict></plist>"
	os.WriteFile(p, []byte(other), 0o644)
	if err := Refresh(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != other {
		t.Fatalf("a launch agent jianmu didn't write was changed:\n%s", b)
	}
}

// A login item turned on in a build published under another bundle id
// (com.yetone.jianmu) is carried to this one's label rather than lost: the
// program stays the one the user turned it on for, and the old record goes,
// so login doesn't open jianmu twice.
func TestAdoptsTheEarlierLabel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	old := earlier()
	if err := os.MkdirAll(filepath.Dir(old), 0o755); err != nil {
		t.Fatal(err)
	}
	exe := "/Applications/Tom & Jerry/jianmu.app/Contents/MacOS/jianmu"
	was := strings.Replace(olderPlist, "<string>"+label+"</string>", "<string>"+before+"</string>", 1)
	if err := os.WriteFile(old, []byte(was), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Refresh(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(record())
	if err != nil {
		t.Fatalf("no launch agent under this label: %v", err)
	}
	if want := plist(exe); string(b) != string(want) {
		t.Fatalf("the carried record isn't this version's:\n%s\nwant:\n%s", b, want)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("the earlier label's record is still there: %v", err)
	}

	// a record with no program names nothing to run: nothing is carried,
	// and nothing loops on the rewrite
	bare := regexp.MustCompile(`(?s)\t<key>ProgramArguments</key>.*?</array>\n`).ReplaceAllString(was, "")
	if strings.Contains(bare, "ProgramArguments") {
		t.Fatalf("the test's record still names a program:\n%s", bare)
	}
	if err := os.WriteFile(old, []byte(bare), 0o644); err != nil {
		t.Fatal(err)
	}
	os.Remove(record())
	if err := Refresh(); err != nil {
		t.Fatalf("a record with no program: %v", err)
	}
	if _, err := os.Stat(record()); !os.IsNotExist(err) {
		t.Fatalf("a record with no program was carried over: %v", err)
	}
}

// fakeApp lays out an app bundle as a release's: Contents/MacOS/jianmu
// beside Contents/Info.plist carrying the bundle id.
func fakeApp(t *testing.T, dir, id string) string {
	t.Helper()
	app := filepath.Join(dir, "Tom & Jerry", "jianmu.app", "Contents")
	if err := os.MkdirAll(filepath.Join(app, "MacOS"), 0o755); err != nil {
		t.Fatal(err)
	}
	info := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key><string>jianmu</string>
	<key>CFBundleIdentifier</key><string>` + id + `</string>
	<key>CFBundleExecutable</key><string>jianmu</string>
</dict>
</plist>
`
	if err := os.WriteFile(filepath.Join(app, "Info.plist"), []byte(info), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(app, "MacOS", "jianmu")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return exe
}

// Login Items lists a launch agent under the name its program is signed
// with, the developer's own, unless the agent names the app it belongs to
// (#1512, ysicing: "Xipeng Guan" instead of jianmu). The launch agent for
// a program inside an app names that app's bundle id, an older jianmu's is
// rewritten with it, and a bare program names none.
func TestLaunchAgentNamesItsApp(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	exe := fakeApp(t, home, "com.jianmu")

	key := "<key>AssociatedBundleIdentifiers</key>\n\t<array><string>com.jianmu</string></array>"
	if b := string(plist(exe)); !strings.Contains(b, key) {
		t.Fatalf("the launch agent doesn't name jianmu's app:\n%s", b)
	}

	p := record()
	os.MkdirAll(filepath.Dir(p), 0o755)
	older := strings.Replace(olderPlist, "/Applications/Tom &amp; Jerry/jianmu.app/Contents/MacOS/jianmu", strings.ReplaceAll(exe, "&", "&amp;"), 1)
	if err := os.WriteFile(p, []byte(older), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Refresh(); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if !strings.Contains(string(b), key) {
		t.Fatalf("an older launch agent isn't rewritten to name the app:\n%s", b)
	}
	if out, err := exec.Command("plutil", "-lint", p).CombinedOutput(); err != nil {
		t.Fatalf("not a valid plist: %s", out)
	}
	out, err := exec.Command("plutil", "-extract", "AssociatedBundleIdentifiers.0", "raw", p).CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "com.jianmu" {
		t.Fatalf("plutil reads AssociatedBundleIdentifiers as %q (%v)", out, err)
	}

	// the app's Info.plist not reading is no reason to drop the name
	os.Rename(filepath.Join(filepath.Dir(filepath.Dir(exe)), "Info.plist"), filepath.Join(home, "Info.plist.aside"))
	if err := Refresh(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); !strings.Contains(string(b), key) {
		t.Fatalf("the app's name was dropped over an unread Info.plist:\n%s", b)
	}

	// a program outside any app has nothing to name
	if b := string(plist(filepath.Join(home, "bin", "jianmu"))); strings.Contains(b, "AssociatedBundleIdentifiers") {
		t.Fatalf("a bare program's launch agent names an app:\n%s", b)
	}
}

// A record a write cut short is not one that opens jianmu at login: launchd
// refuses a job with no program to run, and one with nothing in it at all,
// so Open at login reads off rather than saying on for a record that never
// starts anything.
func TestSetThenTruncatedRecordReadsAsOff(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := Set(true); err != nil {
		t.Fatal(err)
	}
	if Enabled() == Off {
		t.Fatal("not on after Set(true)")
	}
	p := record()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	// the head of the launch agent, cut off before the program it names,
	// which is what a write that goes no further leaves behind
	head := b[:strings.Index(string(b), "<key>ProgramArguments</key>")]
	if err := os.WriteFile(p, head, 0o644); err != nil {
		t.Fatal(err)
	}
	if Enabled() != Off {
		t.Errorf("a launch agent with no ProgramArguments reads as on:\n%s", head)
	}
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if Enabled() != Off {
		t.Error("an empty launch agent reads as on")
	}
	// a program named as nothing is no program to run either, and launchd
	// refuses the job
	empty := strings.Replace(string(b), "<array><string>", "<array><string></string><string>", 1)
	if err := os.WriteFile(p, []byte(empty), 0o644); err != nil {
		t.Fatal(err)
	}
	if Enabled() != Off {
		t.Errorf("a launch agent with no program reads as on:\n%s", empty)
	}
}

// Switched off under System Settings › Login Items › Allow in the
// Background, the launch agent stays where it was and opens nothing, so
// Open at login says it needs allowing, not on (#2120). Only the system's
// requiresApproval says that: no answer, or any other, leaves the record's
// word, and no record is off whatever the system says.
func TestEnabledFollowsTheSystemsSwitch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	prev := legacyStatus
	t.Cleanup(func() { legacyStatus = prev })
	var asked string
	status := smRequiresApproval
	legacyStatus = func(p string) int { asked = p; return status }

	if got := Enabled(); got != Off {
		t.Fatalf("no launch agent reads %q, want off", got)
	}
	if err := Set(true); err != nil {
		t.Fatal(err)
	}
	if got := Enabled(); got != NeedsAllowing {
		t.Fatalf("switched off in System Settings reads %q, want %q", got, NeedsAllowing)
	}
	if asked != record() {
		t.Fatalf("asked the system about %q, not the launch agent %q", asked, record())
	}
	for _, s := range []int{smEnabled, smNotRegistered, smNotFound, smUnknown, 7} {
		status = s
		if got := Enabled(); got != On {
			t.Errorf("the system answering %d reads %q, want on", s, got)
		}
	}
	status = smRequiresApproval
	if err := Set(false); err != nil {
		t.Fatal(err)
	}
	if got := Enabled(); got != Off {
		t.Fatalf("turned off reads %q", got)
	}
}

// The system's switch is kept by the launch agent's label, wherever its
// plist is: a test's record under a sandbox home is com.jianmu
// too, and the system would answer for the user's own. The tests ask a
// stand-in that knows nothing; askSystem is the real question.
var askSystem func(string) int

func keepToTheSandbox() {
	askSystem = legacyStatus
	legacyStatus = func(string) int { return smUnknown }
}

// The real question reaches ServiceManagement: a launch agent under a
// label Background Task Management has never seen is notFound, and a path
// with no file at all is notRegistered. The user's own label is never
// asked about.
func TestLegacyStatusAsksTheSystem(t *testing.T) {
	out, err := exec.Command("sw_vers", "-productVersion").Output()
	if err != nil {
		t.Skip("no sw_vers")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "com.jianmu-test.plist")
	b := strings.Replace(string(plist("/usr/bin/true")), "<string>"+label+"</string>", "<string>com.jianmu-test-"+filepath.Base(dir)+"</string>", 1)
	if err := os.WriteFile(p, []byte(b), 0o644); err != nil {
		t.Fatal(err)
	}
	got := askSystem(p)
	if major := strings.SplitN(strings.TrimSpace(string(out)), ".", 2)[0]; major == "11" || major == "12" {
		if got != smUnknown {
			t.Fatalf("macOS %s answered %d, want unknown", out, got)
		}
		return
	}
	if !cgoBuilt {
		t.Skip("built without cgo: the system can't be asked")
	}
	if got != smNotFound {
		t.Errorf("a label BTM never saw answered %d, want notFound (%d)", got, smNotFound)
	}
	if got := askSystem(filepath.Join(dir, "none.plist")); got != smNotRegistered {
		t.Errorf("no file answered %d, want notRegistered (%d)", got, smNotRegistered)
	}
}
