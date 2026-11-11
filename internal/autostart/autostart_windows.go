package autostart

import (
	"errors"

	"golang.org/x/sys/windows/registry"
)

const (
	runKey = `Software\Microsoft\Windows\CurrentVersion\Run`
	name   = "jianmu"
	// where Task Manager's Startup apps keeps an entry switched off: an odd
	// first byte is off
	approvedKey = `Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`
)

// record is where the system keeps it, as a file: none, it's the registry's
func record() string { return "" }

// openRunKey is how the Run key is reached. A test hands back a key the user's
// ACL refuses, which no key this machine's own user has.
var openRunKey = registry.OpenKey

// openApprovedKey is how Task Manager's Startup apps marker is reached, for the
// same reason: read to answer switchedOff, opened for writing to clear it.
var openApprovedKey = registry.OpenKey

// readRunValue is how the jianmu Run value is read off a key that opened. A
// test has no Run value to hand back, so it stands in for this too.
var readRunValue = func(k registry.Key) (string, error) {
	s, _, err := k.GetStringValue(name)
	return s, err
}

func enabled() bool {
	k, err := openRunKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		// No Run key at all is off. A Run key that can't be read is not a
		// no: Windows may still be starting jianmu, and Set is what says so
		// when it can't reach the key to turn that off.
		return !errors.Is(err, registry.ErrNotExist)
	}
	defer k.Close()
	if _, err = readRunValue(k); err != nil {
		return !errors.Is(err, registry.ErrNotExist)
	}
	return !switchedOff()
}

// switchedOff: turned off in Task Manager, which leaves the Run value be. A
// marker that isn't there, or that can't be read, is not switched off: only a
// marker that says so is, since the Run value is what starts jianmu.
func switchedOff() bool {
	k, err := openApprovedKey(registry.CURRENT_USER, approvedKey, registry.QUERY_VALUE)
	if err != nil {
		// No marker at all is not switched off, and neither is one that
		// can't be read: Windows writes this key the first time Startup
		// apps lists an entry, so a machine whose user has never opened it
		// has no key at all while every Run value it has is on. The Run
		// value is what starts jianmu, so reporting switched off here
		// would hide a jianmu that does start.
		return false
	}
	defer k.Close()
	b, _, err := k.GetBinaryValue(name)
	return err == nil && len(b) > 0 && b[0]&1 == 1
}

// clearApprovedMarker takes Task Manager's Startup apps marker off the Run
// value, so Windows starts jianmu again at sign-in. A marker that can't be
// reached or deleted leaves the Run value switched off, so saying nothing
// about it would let Set(true) report on while Windows starts nothing.
func clearApprovedMarker() error {
	a, err := openApprovedKey(registry.CURRENT_USER, approvedKey, registry.SET_VALUE)
	if err != nil {
		if !errors.Is(err, registry.ErrNotExist) {
			return err
		}
		return nil // no marker is nothing to clear
	}
	defer a.Close()
	if err := a.DeleteValue(name); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	return nil
}

// the user's own Run value, which Windows starts at sign-in (and which
// Task Manager's Startup apps can switch off)
func enable(exe string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if err := k.SetStringValue(name, `"`+exe+`" `+Arg); err != nil {
		return err
	}
	// switched on here after Task Manager switched it off: on again there
	return clearApprovedMarker()
}

func disable() error {
	k, err := openRunKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	if err != nil {
		return nil // no Run key is nothing to turn off
	}
	defer k.Close()
	if err := k.DeleteValue(name); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	return nil
}

// before is the Run value's name in builds published under another name.
// An upgrade carries it to name so the login item the user turned on is not
// lost, switch included, and leaves only one value that starts jianmu.
const before = "magpie"

// refresh carries a Run value written under the earlier name over to this
// one; a value this version already has is left alone, and the earlier one
// removed. Nothing else an older jianmu wrote here needs writing again.
func refresh() error {
	k, err := openRunKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return err
	}
	defer k.Close()
	old, _, err := k.GetStringValue(before)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return err
	}
	if _, err := readRunValue(k); errors.Is(err, registry.ErrNotExist) {
		if err := k.SetStringValue(name, old); err != nil {
			return err
		}
		carryApproved()
	}
	if err := k.DeleteValue(before); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	dropApproved(before)
	return nil
}

// carryApproved gives name Task Manager's switch as it was for before; one
// that can't be reached leaves the value on, as a new one is.
func carryApproved() {
	a, err := openApprovedKey(registry.CURRENT_USER, approvedKey, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return
	}
	defer a.Close()
	if b, _, err := a.GetBinaryValue(before); err == nil {
		_ = a.SetBinaryValue(name, b)
	}
}

// dropApproved takes the switch kept for a value that is gone.
func dropApproved(value string) {
	a, err := openApprovedKey(registry.CURRENT_USER, approvedKey, registry.SET_VALUE)
	if err != nil {
		return
	}
	defer a.Close()
	_ = a.DeleteValue(value)
}

// heldBack: nothing here is switched off apart from the record itself
func heldBack() bool { return false }
