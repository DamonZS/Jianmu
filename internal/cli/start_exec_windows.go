package cli

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"

	"github.com/DamonZS/Jianmu/internal/proc"
)

// startExecOS starts the agent at path and waits for it, its exit code
// jianmu's: Windows has no exec. Ctrl+C is the agent's to handle, so
// jianmu lets it pass.
func startExecOS(path, _ string, args, env []string) error {
	cmd := proc.Command(path, args...)
	cmd.Env = env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	signal.Ignore(os.Interrupt)
	err := cmd.Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return exitError{code: ee.ExitCode()}
	}
	return err
}
