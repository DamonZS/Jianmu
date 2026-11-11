//go:build !darwin

// Package rosetta says whether jianmu runs under Rosetta, which only
// macOS has.
package rosetta

// Translated says this process runs under Rosetta: never off macOS.
func Translated() bool { return false }
