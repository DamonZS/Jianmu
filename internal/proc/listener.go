package proc

import (
	"context"
	"debug/buildinfo"
	"errors"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// The process listening on a port, and its program: how the GUI finds the
// older jianmu that keeps the gateway's port and quits it, when the user
// asks it to (leslie_luo on Discord: the page said an older jianmu served
// the gateway, with no way to find it and quit it).

// ErrDenied is a process this user may not end: another user's, or one
// run as administrator.
var ErrDenied = errors.New("not allowed to end it")

// ListeningOn is the ids of the processes listening on TCP port, that this
// user can see: lsof on a Mac, /proc on Linux, Get-NetTCPConnection on
// Windows. Another user's process may be missing.
func ListeningOn(ctx context.Context, port int) ([]int, error) { return listeningOn(ctx, port) }

// Executable is the path of the program process pid runs, as the system
// says it: the one it was started from, also when that file has been
// replaced since (an update).
func Executable(pid int) (string, error) {
	p, err := executable(pid)
	if err != nil {
		return "", err
	}
	// Linux names a replaced file so
	return strings.TrimSuffix(p, " (deleted)"), nil
}

// Terminate asks process pid to end: SIGTERM on Unix, which a jianmu quits
// on; on Windows it is ended at once (nothing else reaches a tray app).
// ErrDenied when this user may not.
func Terminate(pid int) error { return terminate(pid) }

// Kill ends process pid at once. ErrDenied when this user may not.
func Kill(pid int) error { return kill(pid) }

// RunsJianmu says whether process pid, running the program at path (as
// Executable says), is a jianmu: by the program's name (IsJianmu), or else
// by what the program is, jianmu's own main package, read from the build
// information Go writes into it. A jianmu under a name of the user's
// (acezan on Discord: jianmu.real, behind a wrapper script installed as
// jianmu) is still one, and on Linux this holds for one whose file an
// update has since replaced, read through /proc/<pid>/exe: the panel had
// called that jianmu "not jianmu" and left it holding the gateway's port.
func RunsJianmu(pid int, path string) bool {
	if IsJianmu(path) {
		return true
	}
	file := path
	if runtime.GOOS == "linux" && pid > 0 {
		file = filepath.Join("/proc", strconv.Itoa(pid), "exe")
	}
	if file == "" {
		return false
	}
	bi, err := buildinfo.ReadFile(file)
	return err == nil && bi.Path == jianmuMain
}

// jianmuMain is jianmu's main package, as a build of it names itself; a
// var for the tests, whose own binary is another package's.
var jianmuMain = "github.com/DamonZS/Jianmu"

// IsJianmu says whether the program at path is a jianmu: jianmu, its
// jianmu-* builds or dial, its name before; with .exe, and Windows' .old
// for one replaced by an update while it ran, or .old-2 when that one was
// still held; and jianmu.real, the program a wrapper script named jianmu
// runs (acezan on Discord).
func IsJianmu(path string) bool {
	if path == "" {
		return false
	}
	n := strings.ToLower(filepath.Base(filepath.Clean(strings.ReplaceAll(path, `\`, "/"))))
	// an update moves a running exe aside to exe.old, and to exe.old-2, -3
	// when that one is held too, so the tail is a number as often as not
	if i := strings.LastIndex(n, ".old"); i > 0 && isOldCount(n[i+len(".old"):]) {
		n = n[:i]
	}
	n = strings.TrimSuffix(n, ".old")
	n = strings.TrimSuffix(n, ".new")
	n = strings.TrimSuffix(n, ".exe")
	n = strings.TrimSuffix(n, ".real")
	for _, name := range []string{"jianmu", "dial"} {
		if n == name || strings.HasPrefix(n, name+"-") || strings.HasPrefix(n, name+"_") {
			return true
		}
	}
	return false
}

// isOldCount is the -2, -3 an update appends when exe.old is taken too: a
// number alone is jianmu's own naming, so another program's is left alone.
func isOldCount(s string) bool {
	s = strings.TrimPrefix(s, "-")
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// IsForwarder says whether the program at path holds a port for a container
// or a VM, which serves on it from inside: OrbStack's helper, Docker
// Desktop's backend and vpnkit, docker-proxy, Podman's gvproxy, Lima's
// (Colima's, Rancher Desktop's) host agent, WSL's relay. A jianmu answering
// there runs in that container (leslie_luo on Discord: jianmu v0.1.552 in
// OrbStack held 3425, and its helper was named as the program to quit),
// which only the container's own tools stop.
func IsForwarder(path string) bool {
	if path == "" {
		return false
	}
	p := strings.ToLower(strings.ReplaceAll(path, `\`, "/"))
	for _, app := range []string{"/orbstack.app/", "/docker.app/", "/rancher desktop.app/", "/podman desktop.app/", "/docker/resources/"} {
		if strings.Contains(p, app) {
			return true
		}
	}
	n := strings.TrimSuffix(filepath.Base(filepath.Clean(p)), ".exe")
	for _, name := range []string{"orbstack", "com.docker.", "docker-proxy", "dockerd", "vpnkit", "gvproxy", "podman", "limactl", "lima-", "wslrelay", "wslhost", "rootlesskit", "slirp4netns"} {
		if strings.HasPrefix(n, name) {
			return true
		}
	}
	return false
}
