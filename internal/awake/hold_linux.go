package awake

import (
	"os/exec"
	"syscall"

	"github.com/DamonZS/Jianmu/internal/proc"
)

// takeHold runs systemd-inhibit holding idle and sleep off around a sleep
// that never ends; the inhibitor goes with the process, which goes with
// jianmu (Pdeathsig) should jianmu end without letting go. The idle
// inhibitor is all logind has for the display as well, so display asks for
// nothing more: whether the screen stays on is the desktop's to say.
func takeHold(display bool) (func(), error) {
	path, err := exec.LookPath("systemd-inhibit")
	if err != nil {
		return nil, err
	}
	cmd := proc.Command(path, "--what=idle:sleep", "--who=jianmu", "--why=Agents are working through jianmu", "--mode=block", "sleep", "infinity")
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return func() {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		_ = cmd.Wait()
	}, nil
}
