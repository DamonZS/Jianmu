// Package appdir decides where jianmu keeps its own files: the one place
// every other package asks (#508).
//
// Installed, that is ~/.config/jianmu ($XDG_CONFIG_HOME/jianmu) for what
// jianmu keeps and ~/.cache/jianmu ($XDG_CACHE_HOME/jianmu) for what it
// can fetch again, as it always was. Portable, when a folder named "data"
// (or a file named ".portable") sits beside jianmu, everything jianmu
// keeps goes into that data folder, its caches into data/cache and the
// Windows webview's profile into data/webview2 (no Start-menu shortcut or
// App Paths entry is made either), and
// nothing of jianmu's own is written to the user's profile, the way VS
// Code's portable mode works. "Beside jianmu" is the folder holding the
// executable, with links followed; for a Mac app it is the folder holding
// Jianmu.app (a file put inside the bundle would break its signature), and
// for an AppImage the folder holding the .AppImage file, not the folder it
// is mounted at while it runs.
//
// What isn't jianmu's own stays where its owner reads it: the agents'
// configs (~/.claude, ~/.codex…), the sign-ins they keep in the system
// keychain, the folders the OS reads for autostart and link handling, and
// temporary folders.
//
// A data folder in a folder every app or download shares (/Applications on
// a Mac; the home folder, Downloads, Desktop and Documents everywhere) is
// taken for jianmu's own only when it already holds jianmu's files and the
// installed folder doesn't, or with a ".portable" marker: a "data" folder
// there is as likely another app's or the user's own. Taking one showed a
// user who had never asked for portable an empty jianmu (#989), and wrote
// jianmu's files into a Downloads\data folder a Windows user's archiver
// empties, so a provider just added was gone (tangle778 on X). Where the
// system keeps Downloads, Desktop or Documents off home (moved to D:\ or
// OneDrive on Windows, ~/下载 on Linux) only a data folder holding none of
// jianmu's files is passed over: before v0.1.2648 those folders weren't
// judged at all, so a data folder there holding jianmu's files is one this
// user's jianmu has been keeping its files in, and an update never moves
// them back to an installed folder from long ago (jwaitq on Discord).
//
// The decision is made once, from the executable as it was when jianmu
// started: an update moving the running copy aside (into .jianmu-update on
// a Mac, to jianmu.exe.old on Windows) leaves it running from somewhere
// else, and its files must not move with it.
package appdir

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const (
	// DataName is the folder beside jianmu that makes it portable and
	// holds its files.
	DataName = "data"
	// MarkerName is a file beside jianmu that makes it portable too, its
	// files then going into a data folder made beside it.
	MarkerName = ".portable"
)

var (
	once     sync.Once
	portable string // the data folder, or "" when installed
)

// Portable is the data folder jianmu keeps everything in, or "" when it
// isn't portable.
func Portable() string {
	once.Do(func() {
		exe, err := os.Executable()
		if err != nil {
			return
		}
		portable = Resolve(exe)
		if portable == "" {
			if data := Passed(exe); data != "" {
				log.Printf("appdir: %s is beside jianmu but not taken for its portable data: it is in a shared folder and %s has jianmu's files too; a .portable file beside jianmu makes it portable", data, installed())
			}
		}
	})
	return portable
}

// UseExecutable decides again as if jianmu ran from exe ("" for an
// installed jianmu, whatever runs): for tests.
func UseExecutable(exe string) {
	once.Do(func() {})
	portable = ""
	if exe != "" {
		portable = Resolve(exe)
	}
}

// Resolve is the data folder of a jianmu run from exe, or "" when it isn't
// portable.
func Resolve(exe string) string {
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	base := Beside(exe)
	data := filepath.Join(base, DataName)
	if fi, err := os.Stat(filepath.Join(base, MarkerName)); err == nil && !fi.IsDir() {
		return absolute(data)
	}
	if fi, err := os.Stat(data); err == nil && fi.IsDir() && !shared(base, data) {
		return absolute(data)
	}
	return ""
}

// Passed is the data folder beside a jianmu run from exe that holds
// jianmu's files but that Resolve doesn't take for its own, because it is
// in a shared folder while the installed folder has jianmu's files too, or
// "". A data folder holding none of them is someone else's and isn't named.
func Passed(exe string) string {
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	base := Beside(exe)
	data := filepath.Join(base, DataName)
	if fi, err := os.Stat(data); err == nil && fi.IsDir() && hasFiles(data) && Resolve(exe) == "" {
		return absolute(data)
	}
	return ""
}

// shared says whether the data folder in dir can't be taken for jianmu's:
// dir is a folder every app or download shares, where a folder named data
// is as likely another app's or the user's own, and either data holds none
// of jianmu's files or the installed folder holds them too. One who keeps
// jianmu portable there has its files in data and never ran it installed,
// and keeps it portable; a ".portable" marker beside it makes it portable
// whatever is there.
//
// A data folder in the place the system keeps a user folder moved off home
// (movedFolder) is passed over only when it holds none of jianmu's files.
// Releases before v0.1.2648 took any data folder there, so one holding
// jianmu's files is where this jianmu has kept them, maybe for months,
// while an installed folder with files of its own is from before then:
// judging it as in home\Downloads moved a user back to that old setup on
// updating (jwaitq on Discord).
func shared(dir, data string) bool {
	switch {
	case sharedFolder(dir):
		if !hasFiles(data) {
			return true
		}
		in := installed()
		return in != "" && hasFiles(in)
	case movedFolder(dir):
		return !hasFiles(data)
	}
	return false
}

// hasFiles says whether dir holds jianmu's own files.
func hasFiles(dir string) bool {
	for _, f := range []string{"providers.json", "logins.json", "settings.json"} {
		if fi, err := os.Stat(filepath.Join(dir, f)); err == nil && !fi.IsDir() {
			return true
		}
	}
	return false
}

// sharedFolder says whether dir is a folder every app or download shares:
// /Applications and ~/Applications on a Mac, and the home folder, its
// Downloads, Desktop and Documents everywhere, under home. Tests change it.
var sharedFolder = func(dir string) bool {
	var dirs []string
	if runtime.GOOS == "darwin" {
		dirs = append(dirs, "/Applications")
	}
	if h, err := Home(); err == nil {
		dirs = append(dirs, h)
		for _, d := range []string{DownloadsFolder, DesktopFolder, DocumentsFolder} {
			dirs = append(dirs, filepath.Join(h, d))
		}
		if runtime.GOOS == "darwin" {
			dirs = append(dirs, filepath.Join(h, "Applications"))
		}
	}
	return sameAny(dir, dirs)
}

// movedFolder says whether dir is where the system keeps the user's
// Downloads, Desktop or Documents (userFolder), moved off home or not.
var movedFolder = func(dir string) bool {
	var dirs []string
	for _, d := range []string{DownloadsFolder, DesktopFolder, DocumentsFolder} {
		if p := userFolder(d); p != "" {
			dirs = append(dirs, p)
		}
	}
	return sameAny(dir, dirs)
}

// userFolder is UserFolder; tests change it.
var userFolder = UserFolder

func sameAny(dir string, dirs []string) bool {
	for _, d := range dirs {
		if strings.EqualFold(filepath.Clean(dir), filepath.Clean(d)) {
			return true
		}
	}
	return false
}

// installed is the folder an installed jianmu keeps its files in, or ""
// with no home folder.
func installed() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); rooted(x) {
		return filepath.Join(x, "jianmu")
	}
	h, err := Home()
	if err != nil {
		return ""
	}
	return filepath.Join(h, ".config", "jianmu")
}

// Beside is the folder a portable jianmu run from exe keeps its data folder
// in: the executable's own, the one holding the .app on a Mac, the one
// holding the .AppImage on Linux. APPIMAGE counts only for the AppImage
// jianmu runs from (exe under its mount, APPDIR): every program an
// AppImage starts inherits both, a jianmu started from an AppImage
// terminal too.
func Beside(exe string) string {
	if img := os.Getenv("APPIMAGE"); img != "" && runtime.GOOS == "linux" && within(exe, os.Getenv("APPDIR")) {
		return filepath.Dir(img)
	}
	dir := filepath.Dir(exe)
	// …/Jianmu.app/Contents/MacOS/jianmu
	if filepath.Base(dir) == "MacOS" && filepath.Base(filepath.Dir(dir)) == "Contents" {
		if app := filepath.Dir(filepath.Dir(dir)); strings.EqualFold(filepath.Ext(app), ".app") {
			return filepath.Dir(app)
		}
	}
	return dir
}

// within says whether path lies in the absolute folder dir.
func within(path, dir string) bool {
	if !filepath.IsAbs(dir) {
		return false
	}
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func absolute(p string) string {
	if a, err := filepath.Abs(p); err == nil {
		return a
	}
	return p
}

// Config is the folder jianmu keeps its settings, providers, usage and the
// rest of its own state in.
func Config() string {
	if p := Portable(); p != "" {
		return p
	}
	if x := os.Getenv("XDG_CONFIG_HOME"); rooted(x) {
		return Join(x, "jianmu")
	}
	return Join(Join(mustHome(), ".config"), "jianmu")
}

// Redirected is the folder jianmu keeps its files in when it is opened
// from the Dock, the Start menu or a desktop launcher (~/.config/jianmu),
// while this jianmu reads them from another that XDG_CONFIG_HOME, set in
// the shell it was run from, names; "" when they are the same, or jianmu
// is portable. A jianmu run in such a shell sees none of what the app's
// window shows.
func Redirected() string {
	if Portable() != "" || os.Getenv("XDG_CONFIG_HOME") == "" {
		return ""
	}
	h, err := Home()
	if err != nil {
		return ""
	}
	if def := filepath.Join(h, ".config", "jianmu"); filepath.Clean(def) != filepath.Clean(Config()) {
		return def
	}
	return ""
}

// Cache is the folder for what jianmu can fetch again (the models.dev
// catalog, exchange rates, market lists).
func Cache() string {
	if p := Portable(); p != "" {
		return Join(p, "cache")
	}
	if x := os.Getenv("XDG_CACHE_HOME"); rooted(x) {
		return Join(x, "jianmu")
	}
	return Join(Join(mustHome(), ".cache"), "jianmu")
}

// Logs is the folder the desktop app writes its log in, where each system
// keeps an app's logs: ~/Library/Logs/jianmu on a Mac (beside the launch
// agent's serve.log, `jianmu daemon install`), %LOCALAPPDATA%\jianmu\logs
// on Windows, and $XDG_STATE_HOME/jianmu (~/.local/state/jianmu) elsewhere;
// portable, data/logs. A user asked for jianmu's log finds it there
// (rayem1w on X: the app opened from the Finder had written none).
func Logs() string {
	return logsOn(runtime.GOOS)
}

// logsOn is Logs as it is on goos, for tests of every system's folder.
func logsOn(goos string) string {
	if p := Portable(); p != "" {
		return filepath.Join(p, "logs")
	}
	switch goos {
	case "darwin":
		return filepath.Join(mustHome(), "Library", "Logs", "jianmu")
	case "windows":
		local := os.Getenv("LOCALAPPDATA")
		if !rooted(local) {
			local = filepath.Join(mustHome(), "AppData", "Local")
		}
		return filepath.Join(local, "jianmu", "logs")
	}
	if x := os.Getenv("XDG_STATE_HOME"); rooted(x) {
		return filepath.Join(x, "jianmu")
	}
	return filepath.Join(mustHome(), ".local", "state", "jianmu")
}

// WebView is the folder the Windows webview (WebView2) keeps its profile
// in — its cache, cookies and the pages' storage — or "" for its own
// default. Installed, that default is %APPDATA%\<exe name>, as it always
// was, so nothing kept there is lost; portable, it is data\webview2, so a
// portable jianmu leaves no jianmu.exe folder in the user's AppData (#508).
// A Windows jianmu installed from a download under another name
// (update.Settle) keeps the profile it had there: the WebViewNote beside
// it names that folder.
func WebView() string {
	if p := Portable(); p != "" {
		return filepath.Join(p, "webview2")
	}
	if runtime.GOOS != "windows" {
		return ""
	}
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return noted(filepath.Join(filepath.Dir(exe), WebViewNote))
}

// WebViewNote is the file beside an installed Windows jianmu naming the
// webview profile it keeps, the one it had where it was downloaded.
const WebViewNote = "webview-profile"

// noted is the absolute folder the note at path names, "" when there is
// no such note.
func noted(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if p := strings.TrimSpace(string(b)); filepath.IsAbs(p) {
		return p
	}
	return ""
}

// SystemCache is the cache folder the OS names for jianmu
// (~/Library/Caches/jianmu on a Mac, %LocalAppData%\jianmu on Windows),
// which a few caches have always used; portable, it is Cache.
func SystemCache() (string, error) {
	if p := Portable(); p != "" {
		return filepath.Join(p, "cache"), nil
	}
	d, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "jianmu"), nil
}
