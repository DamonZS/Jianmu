#!/bin/sh
# jianmu installer: curl -fsSL https://usejianmu.ai/install.sh | sh
#
# macOS: puts jianmu.app in /Applications (~/Applications when that is not
# writable) and links the jianmu command into ~/.local/bin.
# Linux: puts jianmu in ~/.local/bin — the desktop app (with a menu entry)
# when GTK 3 and WebKitGTK 4.1 are installed, the terminal build otherwise.
# A computer with a desktop but without them has WebKitGTK 4.1 installed
# with its package manager (apt-get, dnf, pacman or zypper, through sudo,
# which asks for the password) when someone is at the terminal, then gets
# the desktop app; with no terminal, no sudo or a failed install, the
# terminal build and the command that installs them. --gui and --tui
# choose instead (JIANMU_CLI=1 is --tui).
# Termux: installs the Android terminal build into $PREFIX/bin.
# Every download is checked against the release's SHA-256.
#
# Options (curl -fsSL https://usejianmu.ai/install.sh | sh -s -- --proxy …):
#   --proxy <url>      a proxy for every download: http://, https://,
#                      socks5:// or socks5h:// (JIANMU_PROXY= does the same;
#                      without either, curl's own https_proxy / HTTPS_PROXY /
#                      ALL_PROXY are used as usual)
#   --mirror <prefix>  a GitHub download mirror: the release file is fetched
#                      from <prefix><its github.com URL> (JIANMU_MIRROR= does
#                      the same). None is used unless given. The release
#                      list and its SHA-256 still come from usejianmu.ai,
#                      never from the mirror, so a file the mirror changed is
#                      refused.
#   --gui              the desktop app, on Linux even when its libraries can't
#                      be installed now (it opens once they are)
#   --tui              the terminal build (TUI, web UI and CLI), on Linux even
#                      with a desktop (JIANMU_CLI=1 does the same)
#   --uninstall        `jianmu uninstall --yes --keep-data` first (every
#                      agent disconnected and put back as it was), then
#                      takes off what this script put there: the jianmu
#                      command, and the app (macOS) or its menu entry, icon
#                      and jianmu:// handler (Linux), and the service and
#                      opening at login that would start a jianmu no longer
#                      there. Your settings, providers and usage in
#                      ~/.config/jianmu stay, for when you install again.
#   --purge            the same with `jianmu uninstall --yes`: jianmu's
#                      settings, providers and usage deleted too.
#                      With jianmu deleted by hand already (or one too old
#                      to keep its files), both download the newest jianmu
#                      to a temporary folder, checked like an install, to
#                      disconnect the agents with; nothing of it stays.
set -eu

site=https://usejianmu.ai
bin="${JIANMU_BIN_DIR:-$HOME/.local/bin}"
proxy="${JIANMU_PROXY:-}"
mirror="${JIANMU_MIRROR:-}"
uninstall=
build=
nolibs=
[ -z "${JIANMU_CLI:-}" ] || build=tui

say() { printf '  %s\n' "$*"; }
die() { printf 'jianmu: %s\n' "$*" >&2; exit 1; }

while [ $# -gt 0 ]; do
  case "$1" in
    --proxy) [ $# -ge 2 ] || die "--proxy needs a value"; proxy=$2; shift 2 ;;
    --proxy=*) proxy=${1#--proxy=}; shift ;;
    --mirror) [ $# -ge 2 ] || die "--mirror needs a value"; mirror=$2; shift 2 ;;
    --mirror=*) mirror=${1#--mirror=}; shift ;;
    --uninstall) [ "$uninstall" = purge ] || uninstall=1; shift ;;
    --purge) uninstall=purge; shift ;;
    --gui) build=gui; shift ;;
    --tui|--cli) build=tui; shift ;;
    -h|--help)
      printf '%s\n' "usage: curl -fsSL $site/install.sh | sh -s -- [--gui | --tui] [--proxy <url>] [--mirror <prefix>]" \
        "       curl -fsSL $site/install.sh | sh -s -- --uninstall [--purge]" \
        "  --proxy <url>      http://, https://, socks5:// or socks5h:// proxy for the downloads (or JIANMU_PROXY=)" \
        "  --mirror <prefix>  GitHub download mirror put before the release's github.com URL (or JIANMU_MIRROR=);" \
        "                     the SHA-256 is still checked against $site's" \
        "  --gui              Linux: the desktop app, even when GTK 3 and WebKitGTK 4.1 can't be installed now" \
        "  --tui              Linux: the terminal build (TUI, web UI, CLI), even on a desktop (or JIANMU_CLI=1)" \
        "  --uninstall        disconnect every agent, putting back what each had, and take off what this script installed;" \
        "                     your settings in ~/.config/jianmu stay (a jianmu deleted already is downloaded to do it)" \
        "  --purge            with --uninstall: also delete jianmu's settings, providers and usage (jianmu uninstall)"
      exit 0 ;;
    *) die "unknown option $1 (--gui, --tui, --proxy <url>, --mirror <prefix>, --uninstall, --purge)" ;;
  esac
done
case "$proxy" in
  ''|http://?*|https://?*|socks5://?*|socks5h://?*) ;;
  *) die "--proxy is an address like http://127.0.0.1:7890 or socks5://127.0.0.1:1080, not $proxy" ;;
esac
case "$mirror" in
  '') ;;
  http://?*|https://?*) case "$mirror" in */) ;; *) mirror="$mirror/" ;; esac ;;
  *) die "--mirror is an http(s) address put before the github.com URL, like https://mirror.example/, not $mirror" ;;
esac

# get: curl, through the proxy when one was given
get() {
  if [ -n "$proxy" ]; then curl -fsSL --proxy "$proxy" "$@"; else curl -fsSL "$@"; fi
}

case "$(uname -s)" in
  Darwin) os=darwin ;;
  Linux)
    termux="${TERMUX_VERSION:-}"
    case "${PREFIX:-}" in */com.termux/files/usr) termux=1 ;; esac
    if [ -n "$termux" ]; then
      [ -n "${PREFIX:-}" ] || die "Termux requires PREFIX to locate its bin directory"
      os=android
      bin="${JIANMU_BIN_DIR:-$PREFIX/bin}"
    else
      os=linux
    fi ;;
  *) die "unsupported system: $(uname -s); see $site" ;;
esac
share="${XDG_DATA_HOME:-$HOME/.local/share}"
config="${XDG_CONFIG_HOME:-$HOME/.config}"

case "$(uname -m)" in
  arm64|aarch64) arch=arm64 ;;
  x86_64|amd64) arch=amd64 ;;
  *) arch= ;;
esac

# apiroute <url> <file>: GitHub's API address of a release file whose
# github.com address is <url>, read from the release's description with
# sed alone (no jq); nothing when <url> isn't a GitHub release file
apiroute() {
  repo=$(printf '%s' "$1" | sed -n 's|^https://github\.com/\([^/]*/[^/]*\)/releases/download/[^/]*/[^/]*$|\1|p')
  tag=$(printf '%s' "$1" | sed -n 's|^https://github\.com/[^/]*/[^/]*/releases/download/\([^/]*\)/[^/]*$|\1|p')
  [ -n "$repo" ] && [ -n "$tag" ] || return 0
  # one asset's own fields (url, id, name) come before its first nested
  # object, so cutting at each { leaves them on one line
  get -H 'Accept: application/vnd.github+json' "https://api.github.com/repos/$repo/releases/tags/$tag" 2>/dev/null | tr -d '\n' | tr '{' '\n' |
    sed -n "s|.*\"url\": *\"\(https://api\.github\.com/repos/$repo/releases/assets/[0-9]*\)\".*\"name\": *\"$2\".*|\1|p" | head -n 1
}

tmp=
# fetch <file>: the newest release's <file> downloaded into $tmp and
# checked against the SHA-256 the site gives, never a mirror's
fetch() {
  file=$1
  if [ -z "$tmp" ]; then
    tmp=$(mktemp -d)
    trap 'rm -rf "$tmp"' EXIT
  fi
  # the release list and its checksums: always from the site, never a mirror
  feed=$(get "$site/api/latest") || die "could not reach $site${proxy:+ through $proxy}"
  # The feed is one line of JSON; pull this file's url and hash out of it.
  entry=$(printf '%s' "$feed" | sed -n "s/.*\"$file\":{\([^}]*\)}.*/\1/p")
  url=$(printf '%s' "$entry" | sed -n 's/.*"url":"\([^"]*\)".*/\1/p')
  api=$(printf '%s' "$entry" | sed -n 's/.*"api":"\([^"]*\)".*/\1/p')
  # github: the file's github.com address when url is the site's copy (R2)
  github=$(printf '%s' "$entry" | sed -n 's/.*"github":"\([^"]*\)".*/\1/p')
  sum=$(printf '%s' "$entry" | sed -n 's/.*"sha256":"\([^"]*\)".*/\1/p')
  version=$(printf '%s' "$feed" | sed -n 's/.*"version":"\([^"]*\)".*/\1/p')
  if [ "$os" = android ] && { [ -z "$url" ] || [ -z "$sum" ]; }; then
    die "the newest release has no $file; build in Termux with go build -tags nogui (see $site/docs/start)"
  fi
  [ -n "$url" ] && [ -n "$sum" ] || die "the newest release has no $file"
  case "$url" in https://github.com/*) github=$url ;; esac

  # The tries, in order: url (the site's copy, or github.com's own);
  # github.com's address, through the mirror when one is given (a mirror
  # is only ever put before a github.com address); GitHub's API, never
  # through a mirror. via is the mirror a try went through.
  from=$url via=
  case "$url" in
    https://github.com/*) [ -z "$mirror" ] || { from="$mirror$url" via=$mirror; } ;;
  esac
  if [ -n "$via" ]; then say "downloading jianmu $version ($file) through $mirror"
  else say "downloading jianmu $version ($file)"; fi
  if ! get -o "$tmp/$file" "$from"; then
    if [ -z "$via" ] && [ -n "$github" ] && [ "$github" != "$url" ]; then
      from=$github
      [ -z "$mirror" ] || { from="$mirror$github" via=$mirror; }
      say "that failed; downloading it from github.com${via:+ through $via} instead"
      get -o "$tmp/$file" "$from" || from=
    else
      from=
    fi
    if [ -z "$from" ]; then
      # github.com's download address can fail (504 on every try, for
      # hours on 2026-11-08) while GitHub's API serves the same file:
      # asked for there, checked against the same SHA-256 below. A mirror
      # given keeps precedence: through one, the API is never asked.
      [ -z "$mirror" ] || die "could not download $mirror${github:-$url}"
      [ -n "$api" ] || api=$(apiroute "${github:-$url}" "$file")
      [ -n "$api" ] || die "could not download $url"
      say "that failed; downloading it through GitHub's API instead"
      get -H 'Accept: application/octet-stream' -o "$tmp/$file" "$api" || die "could not download $url, nor through $api"
      via=
    fi
  fi
  if command -v sha256sum >/dev/null; then got=$(sha256sum "$tmp/$file" | cut -d' ' -f1)
  else got=$(shasum -a 256 "$tmp/$file" | cut -d' ' -f1); fi
  if [ "$got" != "$sum" ]; then
    [ -z "$via" ] || die "$file from $via does not match its checksum from $site; not installed"
    die "$file does not match its checksum"
  fi
}

# keeps <exe>: whether this jianmu can disconnect the agents and keep its
# files (jianmu uninstall --keep-data, which an older one doesn't know)
keeps() {
  "$1" uninstall --help 2>&1 | grep -q -- --keep-data
}

# runs <file> <key>: the program an entry's Exec= or a unit's ExecStart=
# starts, its first word, quoted or not
runs() {
  sed -n "s/^$2=//p" "$1" 2>/dev/null | head -n 1 | sed -e 's/^"\([^"]*\)".*/\1/' -e t -e 's/ .*//'
}

# --uninstall: what the install below puts there, and nothing of the
# user's. The service and login entry go too while they start this
# jianmu: left, they would start a program that is gone at each login.
if [ -n "$uninstall" ]; then
  exe="$bin/jianmu"
  # jianmu disconnects the agents it connected first, putting back what
  # each had: taking it off alone left them pointed at a gateway that is
  # gone (Codex, whimsical_wolf_15477 on Discord). The one installed does
  # it; with none (deleted by hand: a broken link, an app in the Trash)
  # or one too old to keep its files, the newest is downloaded to do it,
  # while jianmu's settings (and the record of what each agent had) are
  # still here.
  with=
  if [ -x "$exe" ]; then with=$exe
  elif [ "$os" = darwin ]; then
    for apps in /Applications "$HOME/Applications"; do
      [ -n "$with" ] || [ ! -x "$apps/jianmu.app/Contents/MacOS/jianmu" ] || with="$apps/jianmu.app/Contents/MacOS/jianmu"
    done
  fi
  if [ -n "$with" ] && [ "$uninstall" != purge ] && ! keeps "$with"; then with=; fi
  if [ -z "$with" ] && [ -d "$config/jianmu" ]; then
    other=$(command -v jianmu 2>/dev/null || true)
    if [ -n "$other" ] && [ "$other" != "$exe" ] && [ -x "$other" ]; then
      # another jianmu, not this script's, serves the agents still: it
      # is the one to take off, and its agents are its own to put back
      say "jianmu at $other isn't one this script installed, so your agents are left on it: run $other uninstall to disconnect them and take it off"
    else
      [ -n "$arch" ] || die "unsupported CPU: $(uname -m)"
      case "$os" in
        darwin) fetch "jianmu-darwin-$arch.zip"
                ditto -x -k "$tmp/$file" "$tmp/x"
                with="$tmp/x/jianmu.app/Contents/MacOS/jianmu" ;;
        android) fetch "jianmu-cli-android-$arch"; chmod 755 "$tmp/$file"; with="$tmp/$file" ;;
        *) fetch "jianmu-cli-linux-$arch"; chmod 755 "$tmp/$file"; with="$tmp/$file" ;;
      esac
      say "disconnecting your agents with it (it is deleted after)"
    fi
  fi
  if [ -n "$with" ]; then
    if [ "$uninstall" = purge ]; then
      JIANMU_UNINSTALL_SCRIPT=1 "$with" uninstall --yes || die "jianmu uninstall didn't finish (see above), so the jianmu command and app are left: run this again once it does"
    else
      JIANMU_UNINSTALL_SCRIPT=1 "$with" uninstall --yes --keep-data || die "jianmu couldn't disconnect your agents (see above), so the jianmu command and app are left: run this again once it can"
    fi
  fi
  removed=
  gone() { rm -rf "$1"; removed=1; say "removed $1"; }
  if [ "$os" = darwin ]; then
    for apps in /Applications "$HOME/Applications"; do
      [ ! -d "$apps/jianmu.app" ] || gone "$apps/jianmu.app"
    done
    case "$(readlink "$exe" 2>/dev/null)" in
      */jianmu.app/Contents/MacOS/jianmu) gone "$exe" ;;
    esac
  else
    if [ "$os" = linux ]; then
      unit="$config/systemd/user/jianmu.service"
      if [ "$(runs "$unit" ExecStart)" = "$exe" ]; then
        if ! "$exe" daemon uninstall >/dev/null 2>&1; then
          command -v systemctl >/dev/null && systemctl --user disable --now jianmu.service >/dev/null 2>&1 || true
          rm -f "$unit"
        fi
        removed=1; say "removed jianmu's service ($unit)"
      fi
      login="$config/autostart/jianmu.desktop"
      [ "$(runs "$login" Exec)" != "$exe" ] || gone "$login"
    fi
    [ ! -f "$exe" ] || gone "$exe"
    if [ "$os" = linux ]; then
      apps="$share/applications"
      # the menu entry and its icon while they start this jianmu, or one
      # that is gone; another jianmu's stays
      run=$(runs "$apps/jianmu.desktop" Exec)
      if [ -f "$apps/jianmu.desktop" ] && { [ "$run" = "$exe" ] || [ ! -e "$run" ]; }; then
        gone "$apps/jianmu.desktop"
      fi
      # with no menu entry (this one, or the one jianmu uninstall took),
      # what came with it: the icon, the jianmu:// default xdg-mime set
      # (that line only) and the menu's cache when it lists nothing else
      if [ ! -e "$apps/jianmu.desktop" ]; then
        icons="$share/icons/hicolor/256x256/apps"
        [ ! -f "$icons/jianmu.png" ] || gone "$icons/jianmu.png"
        rmdir "$icons" "$share/icons/hicolor/256x256" "$share/icons/hicolor" "$share/icons" 2>/dev/null || true
        list="$config/mimeapps.list"
        if [ -f "$list" ] && grep -qx 'x-scheme-handler/jianmu=jianmu\.desktop;\{0,1\}' "$list"; then
          grep -vx 'x-scheme-handler/jianmu=jianmu\.desktop;\{0,1\}' "$list" > "$list.jianmu-new" || true
          mv "$list.jianmu-new" "$list"
          # a list left with no entry at all (the one xdg-mime made) goes
          grep -qv -e '^\[.*\]$' -e '^[[:space:]]*$' "$list" || rm -f "$list"
          removed=1; say "removed jianmu:// from $list"
        fi
        if [ -d "$apps" ] && ! ls "$apps" | grep -q '\.desktop$'; then
          rm -f "$apps/mimeinfo.cache"
          rmdir "$apps" 2>/dev/null || true
        elif [ -d "$apps" ]; then
          command -v update-desktop-database >/dev/null && update-desktop-database "$apps" 2>/dev/null || true
        fi
      fi
    fi
  fi
  [ "$removed" ] || say "no jianmu installed by this script at $exe"
  if [ "$uninstall" != purge ]; then
    say "kept your settings, providers and usage in $config/jianmu: install again to have them back, or run with --purge to delete them"
  fi
  # jianmu uninstall said so already
  [ "$uninstall" = purge ] || say "jianmu is uninstalled"
  exit 0
fi

# has_gui_libs: GTK 3 and WebKitGTK 4.1 (which brings libsoup 3 and
# JavaScriptCore) can be loaded, which is all the desktop app links beyond
# glibc. ldconfig is in /sbin, off a Debian user's PATH, so it is looked for
# there too; with no cache to ask (NixOS, Guix), the usual folders are.
# musl: the desktop app is built against glibc and can't start on musl
# (Alpine); the terminal build is static and runs anywhere
musl() {
  for l in /lib/ld-musl-*; do [ ! -e "$l" ] || return 0; done
  return 1
}
has_gui_libs() {
  libs=
  for l in "$(command -v ldconfig 2>/dev/null)" /sbin/ldconfig /usr/sbin/ldconfig; do
    [ -n "$l" ] && [ -x "$l" ] || continue
    libs=$("$l" -p 2>/dev/null) && [ -n "$libs" ] && break
    libs=
  done
  if [ -n "$libs" ]; then
    printf '%s\n' "$libs" | grep -q 'libwebkit2gtk-4\.1\.so\.0 ' &&
      printf '%s\n' "$libs" | grep -q 'libgtk-3\.so\.0 '
    return
  fi
  for want in libwebkit2gtk-4.1.so.0 libgtk-3.so.0; do
    found=
    for d in /usr/lib/*-linux-gnu /lib/*-linux-gnu /usr/lib64 /lib64 /usr/lib /lib; do
      [ ! -e "$d/$want" ] || { found=1; break; }
    done
    [ -n "$found" ] || return 1
  done
}

# has_desktop: a graphical session here (DISPLAY, WAYLAND_DISPLAY, a
# desktop's variables), a graphical login of this user that an ssh shell
# doesn't inherit (logind), or a desktop installed to log in to
has_desktop() {
  [ -z "${DISPLAY:-}${WAYLAND_DISPLAY:-}${XDG_CURRENT_DESKTOP:-}${XDG_SESSION_DESKTOP:-}${DESKTOP_SESSION:-}" ] || return 0
  case "${XDG_SESSION_TYPE:-}" in x11|wayland|mir) return 0 ;; esac
  if command -v loginctl >/dev/null 2>&1; then
    # Display falls back to a tty login when there is no graphical one,
    # so the session's own type is what tells
    s=$(loginctl show-user "$(id -un 2>/dev/null)" -p Display --value 2>/dev/null) || s=
    if [ -n "$s" ]; then
      case "$(loginctl show-session "$s" -p Type --value 2>/dev/null)" in
        x11|wayland|mir) return 0 ;;
      esac
    fi
  fi
  dirs=${XDG_DATA_DIRS:-/usr/local/share:/usr/share}
  old=$IFS; IFS=:
  for d in $dirs; do
    for f in "$d"/xsessions/*.desktop "$d"/wayland-sessions/*.desktop; do
      [ ! -e "$f" ] || { IFS=$old; return 0; }
    done
  done
  IFS=$old
  return 1
}

# gui_packages: sets pkg, the package that brings WebKitGTK 4.1 (and GTK 3
# with it) on this distribution, pm, the package manager this script runs
# to install it, and hint, the command to tell the user; each is empty when
# the distribution isn't one we know (pm too where installing it isn't
# quick: Gentoo builds it from source)
gui_packages() {
  pkg='' pm='' hint=''
  id='' like=''
  # JIANMU_OS_RELEASE: another os-release, for the installer's tests
  rel=${JIANMU_OS_RELEASE:-/etc/os-release}
  [ -r "$rel" ] || rel=/usr/lib/os-release
  if [ -r "$rel" ]; then
    id=$(sed -n 's/^ID=//p' "$rel" | tr -d "\"'" | head -n 1)
    like=$(sed -n 's/^ID_LIKE=//p' "$rel" | tr -d "\"'" | head -n 1)
  fi
  for d in $id $like; do
    case "$d" in
      debian|ubuntu|linuxmint|pop|elementary|zorin|kali|raspbian|deepin|uos)
        pm=apt-get pkg=libwebkit2gtk-4.1-0 hint="sudo apt install libwebkit2gtk-4.1-0" ;;
      fedora|rhel|centos|rocky|almalinux|nobara|openeuler|anolis)
        pm=dnf pkg=webkit2gtk4.1 hint="sudo dnf install webkit2gtk4.1" ;;
      arch|manjaro|endeavouros|cachyos|garuda)
        pm=pacman pkg=webkit2gtk-4.1 hint="sudo pacman -S --needed webkit2gtk-4.1" ;;
      opensuse*|suse|sles)
        pm=zypper pkg=libwebkit2gtk-4_1-0 hint="sudo zypper install libwebkit2gtk-4_1-0" ;;
      void)
        hint="sudo xbps-install -S libwebkit2gtk41" ;;
      gentoo)
        hint="sudo emerge net-libs/webkit-gtk:4.1" ;;
      *) continue ;;
    esac
    return 0
  done
}

# install_gui_libs: installs WebKitGTK 4.1 with the distribution's package
# manager, saying first what and why, when someone is at the terminal to
# answer sudo's password (or the script runs as root). curl | sh gives the
# script the pipe for stdin, so the terminal is /dev/tty, and every command
# here reads it. Fails, and the caller falls back to the terminal build,
# with no terminal, no sudo, a package manager we don't run, or a failed
# install. JIANMU_TTY: another terminal, for the installer's tests.
install_gui_libs() {
  tried=
  gui_packages
  [ -n "$pm" ] && command -v "$pm" >/dev/null 2>&1 || return 1
  tty=${JIANMU_TTY:-/dev/tty}
  # a subshell: a redirection that fails would end the script itself
  (: <"$tty") 2>/dev/null || return 1
  as=
  if [ "$(id -u 2>/dev/null)" != 0 ]; then
    command -v sudo >/dev/null 2>&1 || return 1
    as=sudo
  fi
  case "$pm" in
    apt-get) set -- apt-get install -y "$pkg" ;;
    dnf) set -- dnf install -y "$pkg" ;;
    pacman) set -- pacman -S --needed --noconfirm "$pkg" ;;
    zypper) set -- zypper --non-interactive install "$pkg" ;;
  esac
  tried=1
  say "the desktop app needs WebKitGTK 4.1, so installing it: ${as:+$as }$*${as:+ (sudo may ask for your password)}"
  # apt's package lists can be missing (a fresh system) or out of date
  # (a 404 for a version gone from the mirror): read them again and retry
  if $as "$@" <"$tty"; then :
  elif [ "$pm" = apt-get ] && $as apt-get update <"$tty" && $as "$@" <"$tty"; then :
  else return 1; fi
  has_gui_libs
}

[ -n "$arch" ] || die "unsupported CPU: $(uname -m)"
case "$os" in
  darwin) file="jianmu-darwin-$arch.zip" ;;
  android) file="jianmu-cli-android-$arch" ;;
  *)
    file="jianmu-cli-linux-$arch"
    case "$build" in
      gui) ! musl || die "the desktop app is built for glibc and can't run on musl (Alpine): install the terminal build with --tui"
           file="jianmu-linux-$arch"
           has_gui_libs || install_gui_libs || true ;;
      tui) ;;
      *) if musl; then :
         elif has_gui_libs; then file="jianmu-linux-$arch"
         elif has_desktop; then
           if install_gui_libs; then file="jianmu-linux-$arch"; else nolibs=1; fi
         fi ;;
    esac ;;
esac

fetch "$file"

mkdir -p "$bin"
if [ "$os" = darwin ]; then
  apps=/Applications
  [ -w "$apps" ] || { apps="$HOME/Applications"; mkdir -p "$apps"; }
  ditto -x -k "$tmp/$file" "$tmp/x"
  # the new app beside the old one first: a move failing (a full disk)
  # leaves the old app in place
  rm -rf "$apps/.jianmu.app.new"
  mv "$tmp/x/jianmu.app" "$apps/.jianmu.app.new"
  rm -rf "$apps/jianmu.app"
  mv "$apps/.jianmu.app.new" "$apps/jianmu.app"
  ln -sf "$apps/jianmu.app/Contents/MacOS/jianmu" "$bin/jianmu"
  say "installed $apps/jianmu.app"
else
  install -m 755 "$tmp/$file" "$bin/jianmu"
  if [ "$file" = "jianmu-linux-$arch" ]; then
    mkdir -p "$share/applications" "$share/icons/hicolor/256x256/apps"
    get -o "$share/icons/hicolor/256x256/apps/jianmu.png" "$site/img/icon-256.png" || true
    cat > "$share/applications/jianmu.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=jianmu
Comment=Every agent's model. One place.
Exec=$bin/jianmu %u
Icon=jianmu
StartupWMClass=jianmu
Categories=Development;Utility;
MimeType=x-scheme-handler/jianmu;
Terminal=false
EOF
    # jianmu://import links open in jianmu
    command -v update-desktop-database >/dev/null && update-desktop-database "$share/applications" 2>/dev/null || true
    command -v xdg-mime >/dev/null && xdg-mime default jianmu.desktop x-scheme-handler/jianmu 2>/dev/null || true
    say "installed the jianmu desktop app (menu entry: jianmu)"
  elif [ "$os" = android ]; then
    say "installed the Termux build; use jianmu web for the browser UI or jianmu tui for the terminal UI"
  else
    say "installed the terminal build (TUI, web UI and CLI)"
  fi
fi
say "installed $bin/jianmu"
case ":$PATH:" in
  *":$bin:"*) ;;
  *) say "add $bin to your PATH to run jianmu from a terminal" ;;
esac
case "$file" in
  *.zip) say "open it: open -a jianmu" ;;
  jianmu-cli-android-*) say "run it: jianmu web" ;;
  jianmu-linux-*) say "open it from your app menu, or run: jianmu app" ;;
  *) say "run it: jianmu" ;;
esac
# the desktop app without what it needs: what installs it, said last so it
# is what the user reads
if [ "$file" = "jianmu-linux-$arch" ] && ! has_gui_libs; then
  gui_packages
  say ""
  if [ -n "${tried:-}" ]; then say "installing GTK 3 and WebKitGTK 4.1 didn't work (see above); the desktop app opens once they are installed:"
  else say "the desktop app needs GTK 3 and WebKitGTK 4.1, which aren't installed; it opens once they are:"; fi
  say "  ${hint:-install WebKitGTK 4.1 (webkit2gtk 4.1) from your distribution}"
elif [ -n "${nolibs:-}" ]; then
  gui_packages
  say ""
  if [ -n "${tried:-}" ]; then say "this computer has a desktop, but installing GTK 3 and WebKitGTK 4.1 for the desktop app didn't work (see above),"
  else say "this computer has a desktop, but the desktop app needs GTK 3 and WebKitGTK 4.1, which aren't installed,"; fi
  say "so the terminal build was installed. For the desktop app, install them and run this again:"
  say "  ${hint:-install WebKitGTK 4.1 (webkit2gtk 4.1) from your distribution}"
  say "  curl -fsSL $site/install.sh | sh"
  say "or install the desktop app now and the libraries after: curl -fsSL $site/install.sh | sh -s -- --gui"
fi
