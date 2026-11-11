package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/DamonZS/Jianmu/internal/agent"
	"github.com/DamonZS/Jianmu/internal/appdir"
	"github.com/DamonZS/Jianmu/internal/autostart"
	"github.com/DamonZS/Jianmu/internal/daemon"
	"github.com/DamonZS/Jianmu/internal/proc"
)

// On Linux with no desktop session (a WSL distro, a server over SSH)
// nothing opens jianmu at login, so once jianmu tui quit, the agents it had
// connected asked a gateway that was gone: Codex in WSL "连接会失败，没自动
// 启动jianmu的网关" (zaidu259434 on X, with jianmu installed in WSL and
// pointed at Windows' jianmu as a remote one). As it quits having served
// the gateway with an agent connected, the terminal hands the gateway on:
// to jianmu's service (jianmu daemon), installed now where systemd runs it
// and the user never removed it, which starts it with the system or the
// user's login from then on; else to a `jianmu serve` of its own, in the
// background until the system stops.

// keeper is what keepGateway asks and does; a var for the tests.
type keeper struct {
	goos        string
	autostartOn func() bool
	status      func() (daemon.Status, error)
	removed     func() bool
	start       func() error
	install     func() error
	linger      func() string
	background  func() (pid int, log string, err error)
}

var keep = keeper{
	goos:        runtime.GOOS,
	autostartOn: func() bool { return autostart.Enabled() != autostart.Off },
	status:      daemon.Get,
	removed:     daemon.Removed,
	start:       daemon.Start,
	install:     func() error { return daemon.Install(daemon.Options{}) },
	linger:      daemon.Linger,
	background:  serveInBackground,
}

// keepGateway hands the gateway on as the terminal quits having served it
// (see above), and is what to say of it.
func keepGateway(connected bool) string {
	k := keep
	if k.goos != "linux" || !connected {
		return gatewayGoneNote()
	}
	if k.autostartOn() {
		return atLoginNote()
	}
	st, err := k.status()
	switch {
	case err == nil && st.Installed:
		if err := k.start(); err == nil {
			return "jianmu: the gateway goes on in jianmu's service (jianmu daemon status)"
		}
	case err == nil && !k.removed():
		if err := k.install(); err == nil {
			msg := "jianmu: the gateway goes on in jianmu's service, which now starts with this system or your login, so agents connected to jianmu reach it with jianmu tui closed; jianmu daemon uninstall removes it"
			if l := k.linger(); l != "" {
				msg += " (" + l + ")"
			}
			return msg
		}
	}
	pid, log, err := k.background()
	if err != nil {
		return gatewayGoneNote() + " (couldn't start jianmu serve in the background: " + err.Error() + ")"
	}
	msg := fmt.Sprintf("jianmu: the gateway goes on in the background (jianmu serve, pid %d, log %s) until this system stops", pid, log)
	if wslHere() {
		msg += "; for it to start with WSL, turn systemd on with [boot] systemd=true in /etc/wsl.conf and run jianmu daemon install"
	} else {
		msg += "; jianmu daemon install starts it with the system"
	}
	return msg
}

// atLoginNote is said where jianmu opens at login, so the terminal left
// the gateway to the app. On Linux that can be a package's own autostart
// entry on a box no desktop logs into (NixOS-WSL, a server over SSH),
// which nothing ever starts (#3756): the note says the switch that has the
// terminal hand the gateway on instead.
func atLoginNote() string {
	return gatewayGoneNote() + "; jianmu opens at login and serves it then. Where no desktop logs in here, `jianmu autostart off` has jianmu tui hand it on as it quits"
}

// BackgroundLog is the log of a jianmu serve that jianmu tui left
// running as it quit (serveInBackground).
func BackgroundLog() string { return filepath.Join(appdir.Cache(), "serve.log") }

// serveInBackground starts `jianmu serve` on its own, outliving this
// process and its terminal, its output in a log in jianmu's cache.
func serveInBackground() (int, string, error) {
	exe, err := os.Executable()
	if err != nil {
		return 0, "", err
	}
	log := BackgroundLog()
	if err := os.MkdirAll(filepath.Dir(log), 0o700); err != nil {
		return 0, "", err
	}
	f, err := os.OpenFile(log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return 0, "", err
	}
	defer f.Close()
	cmd := proc.Command(exe, "serve")
	cmd.Stdout, cmd.Stderr = f, f
	detach(cmd)
	// reaped if it ends while this process runs: a released one stayed
	// <defunct> until it quit
	if err := proc.StartReaped(cmd); err != nil {
		return 0, "", err
	}
	return cmd.Process.Pid, log, nil
}

// wslHere: this Linux is a WSL distro.
func wslHere() bool {
	b, _ := os.ReadFile("/proc/version")
	return strings.Contains(strings.ToLower(string(b)), "microsoft")
}

// connected says one of agents is connected to jianmu.
func connected(agents []*agent.Agent) bool {
	for _, a := range agents {
		if a.Routed != nil && a.Routed() {
			return true
		}
	}
	return false
}
