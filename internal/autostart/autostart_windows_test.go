package autostart

import (
	"errors"
	"testing"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// Turning Open at login off has to say so when it cannot reach the Run key.
// A Run key whose ACL refuses SET_VALUE is still readable, so nothing else
// changed: the jianmu value stays where it was and Windows starts it at the
// next sign-in, which is the opposite of what was asked for. No Run key at
// all is nothing to turn off, and is not an error.
func TestDisableReportsAKeyItCannotReach(t *testing.T) {
	refuse := func() {
		openRunKey = func(registry.Key, string, uint32) (registry.Key, error) {
			return 0, windows.ERROR_ACCESS_DENIED
		}
	}
	absent := func() {
		openRunKey = func(registry.Key, string, uint32) (registry.Key, error) {
			return 0, registry.ErrNotExist
		}
	}
	restore := openRunKey
	t.Cleanup(func() { openRunKey = restore })

	refuse()
	if err := Set(false); err == nil {
		t.Fatal("a Run key that refuses SET_VALUE was reported as turned off")
	}
	absent()
	if err := Set(false); err != nil {
		t.Fatalf("no Run key is nothing to turn off, not an error: %v", err)
	}
}

// A Run key that cannot be read is not read as off: Windows may still be
// starting jianmu, and the switch says so when it can't turn it off.
func TestEnabledUnreadableKeyIsNotOff(t *testing.T) {
	restore := openRunKey
	t.Cleanup(func() { openRunKey = restore })
	openRunKey = func(registry.Key, string, uint32) (registry.Key, error) {
		return 0, windows.ERROR_ACCESS_DENIED
	}
	if Enabled() == Off {
		t.Fatal("a Run key that cannot be read reads as off")
	}
	openRunKey = func(registry.Key, string, uint32) (registry.Key, error) {
		return 0, registry.ErrNotExist
	}
	if Enabled() != Off {
		t.Fatal("no Run key reads as on")
	}
}

// Turning Open at login on has to say so when it cannot clear Task Manager's
// Startup apps marker. The Run value is written either way, so the switch
// would read as on, but the marker says Windows is to leave jianmu switched
// off: Set(true) answering success there is the same lie Set(false) used to
// tell. No marker at all is nothing to clear, and is not an error.
func TestClearApprovedMarkerReportsOneItCannotReach(t *testing.T) {
	refuse := func() {
		openApprovedKey = func(registry.Key, string, uint32) (registry.Key, error) {
			return 0, windows.ERROR_ACCESS_DENIED
		}
	}
	absent := func() {
		openApprovedKey = func(registry.Key, string, uint32) (registry.Key, error) {
			return 0, registry.ErrNotExist
		}
	}
	restore := openApprovedKey
	t.Cleanup(func() { openApprovedKey = restore })

	refuse()
	if err := clearApprovedMarker(); err == nil {
		t.Fatal("a Startup apps marker that cannot be cleared was reported as on")
	}
	absent()
	if err := clearApprovedMarker(); err != nil {
		t.Fatalf("no marker is nothing to clear, not an error: %v", err)
	}
}

// A marker that cannot be read is not read as switched off. No marker at all
// is not either: Startup apps writes this key the first time it lists an
// entry, so a machine whose user has never opened it has no key while every
// Run value it has is on, and only a marker that says so is switched off.
func TestSwitchedOffUnreadableOrAbsentMarkerIsNotOff(t *testing.T) {
	restore := openApprovedKey
	t.Cleanup(func() { openApprovedKey = restore })
	openApprovedKey = func(registry.Key, string, uint32) (registry.Key, error) {
		return 0, windows.ERROR_ACCESS_DENIED
	}
	if switchedOff() {
		t.Fatal("a Startup apps marker that cannot be read reads as switched off")
	}
	openApprovedKey = func(registry.Key, string, uint32) (registry.Key, error) {
		return 0, registry.ErrNotExist
	}
	if switchedOff() {
		t.Fatal("no marker reads as switched off")
	}
}

// The machine that has the Run value and no Startup apps marker at all is
// the ordinary one: Startup apps writes the marker key the first time it
// lists an entry, so a user who never opened it has no key, and Windows
// still starts every Run value it has. Reading that as switched off told
// Settings "login: off" for a jianmu that does start.
func TestEnabledWithNoMarkerIsOn(t *testing.T) {
	restoreRun, restoreApproved := openRunKey, openApprovedKey
	restoreRead := readRunValue
	t.Cleanup(func() { openRunKey, openApprovedKey, readRunValue = restoreRun, restoreApproved, restoreRead })
	openRunKey = func(registry.Key, string, uint32) (registry.Key, error) {
		return 0, nil
	}
	readRunValue = func(registry.Key) (string, error) { return "C:\\jianmu.exe", nil }
	openApprovedKey = func(registry.Key, string, uint32) (registry.Key, error) {
		return 0, registry.ErrNotExist
	}
	if Enabled() == Off {
		t.Fatal("a Run value with no Startup apps marker reads as off")
	}
}

// An upgrade from the build published as magpie keeps the login item: its Run
// value (and Task Manager's switch for it) moves to jianmu's name, and the
// earlier value goes so login doesn't open two. A jianmu value already there
// wins over the earlier one.
func TestRefreshCarriesTheEarlierRunValue(t *testing.T) {
	const sub = `Software\JianmuTest\Run`
	const appr = `Software\JianmuTest\Approved`
	t.Cleanup(func() {
		registry.DeleteKey(registry.CURRENT_USER, sub)
		registry.DeleteKey(registry.CURRENT_USER, appr)
		registry.DeleteKey(registry.CURRENT_USER, `Software\JianmuTest`)
	})
	k, _, err := registry.CreateKey(registry.CURRENT_USER, sub, registry.ALL_ACCESS)
	if err != nil {
		t.Fatal(err)
	}
	defer k.Close()
	a, _, err := registry.CreateKey(registry.CURRENT_USER, appr, registry.ALL_ACCESS)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	pr, pa := openRunKey, openApprovedKey
	t.Cleanup(func() { openRunKey, openApprovedKey = pr, pa })
	openRunKey = func(_ registry.Key, _ string, acc uint32) (registry.Key, error) {
		return registry.OpenKey(registry.CURRENT_USER, sub, acc)
	}
	openApprovedKey = func(_ registry.Key, _ string, acc uint32) (registry.Key, error) {
		return registry.OpenKey(registry.CURRENT_USER, appr, acc)
	}

	cmd := `"C:\Apps\jianmu\jianmu.exe" tray`
	if err := k.SetStringValue(before, cmd); err != nil {
		t.Fatal(err)
	}
	if err := a.SetBinaryValue(before, []byte{3, 0, 0, 0}); err != nil {
		t.Fatal(err)
	}
	if err := refresh(); err != nil {
		t.Fatal(err)
	}
	if got, _, err := k.GetStringValue(name); err != nil || got != cmd {
		t.Fatalf("jianmu value %q, %v; want the earlier command", got, err)
	}
	if b, _, err := a.GetBinaryValue(name); err != nil || len(b) == 0 || b[0] != 3 {
		t.Fatalf("the switch was not carried: %v, %v", b, err)
	}
	if _, _, err := k.GetStringValue(before); !errors.Is(err, registry.ErrNotExist) {
		t.Fatalf("the earlier value is still there: %v", err)
	}
	if _, _, err := a.GetBinaryValue(before); !errors.Is(err, registry.ErrNotExist) {
		t.Fatalf("the earlier switch is still there: %v", err)
	}

	// a value of this version already there wins; the earlier one still goes
	mine := `"D:\mine\jianmu.exe" tray`
	k.SetStringValue(name, mine)
	k.SetStringValue(before, cmd)
	if err := refresh(); err != nil {
		t.Fatal(err)
	}
	if got, _, _ := k.GetStringValue(name); got != mine {
		t.Fatalf("jianmu value overwritten: %q", got)
	}
	if _, _, err := k.GetStringValue(before); !errors.Is(err, registry.ErrNotExist) {
		t.Fatalf("the earlier value is still there: %v", err)
	}
}
