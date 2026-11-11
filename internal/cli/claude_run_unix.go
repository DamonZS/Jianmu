//go:build !windows

package cli

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/DamonZS/Jianmu/internal/proc"
)

// claudeRunOS starts claude and waits for it, to see what its session
// left in the user's settings.json (agent.ClaudeLaunchAfter). It runs in
// jianmu's process group, so the terminal's Ctrl+C and Ctrl+\ reach it
// as they would reach claude started alone, and jianmu only waits: a
// stop (Ctrl+Z) stops jianmu beside it, so the shell has the terminal
// back, and fg continues both. A SIGTERM or SIGHUP sent to jianmu alone
// is passed on. Its exit code is jianmu's.
func claudeRunOS(bin string, args, env []string) error {
	cmd := proc.Command(bin, args...)
	cmd.Env = env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	// caught, not ignored: an ignored signal stays ignored in claude
	sigs := make(chan os.Signal, 8)
	signal.Notify(sigs, os.Interrupt, syscall.SIGQUIT, syscall.SIGTSTP, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigs)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		for s := range sigs {
			if s == syscall.SIGTERM || s == syscall.SIGHUP {
				_ = cmd.Process.Signal(s)
			}
		}
	}()
	code, err := proc.WaitFollowingStops(cmd)
	if err != nil {
		return err
	}
	if code != 0 {
		return exitError{code: code}
	}
	return nil
}
