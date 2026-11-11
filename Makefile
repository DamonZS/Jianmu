# The recipes (clean's rm -rf too) name paths relative to the working
# folder, so they run only in jianmu's own, known by its go.mod: run from
# another one (make -f path/to/Makefile) they would act there. make -C
# works. go.mod is read relative to the working folder, so a path with
# spaces (or a symlinked Makefile) changes nothing.
ifneq ($(shell sed -n 's/^module //p' go.mod 2>/dev/null),github.com/DamonZS/Jianmu)
  $(error run make in jianmu's folder (make -C path/to/jianmu), not in $(CURDIR))
endif

# A tag may hold $, ( or `, which the shell lines below would run: the
# version keeps only what a version is spelled with.
VERSION ?= $(or $(shell git describe --tags --always --dirty 2>/dev/null | tr -cd 'A-Za-z0-9._+-'),dev)
LDFLAGS  = -s -w -X main.version=$(VERSION)
TAGS     = production
TARGETS  = darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64 windows/arm64 android/arm64

# The GUI links the platform webview through cgo, so it is built natively.
# `nogui` builds the terminal-only jianmu, which cross-compiles anywhere.
# macOS 12 is the oldest Go (1.25 and later) runs on; Info.plist says so too.
ifeq ($(shell uname -s),Darwin)
  export CGO_CFLAGS  = -mmacosx-version-min=12.0
  export CGO_LDFLAGS = -mmacosx-version-min=12.0
endif
# On Linux, GTK 3 and WebKitGTK 4.1: older distributions have them, where
# Wails' default GTK 4 and WebKitGTK 6 would leave them out.
ifeq ($(shell uname -s),Linux)
  TAGS += gtk3
endif
# Termux uses Android's libc and has no desktop webview. Keep the web UI
# and terminal commands, without linking GTK or Wails.
ifeq ($(shell go env GOOS),android)
  TAGS = nogui
endif

.PHONY: assets-pack build-release build cli install test test-ui app icons release release-cli release-windows release-linux clean dev dev-once

# A release build embeds the UI's files as one zip instead of the tree
# (internal/gui/assets_packed.go): a Windows or Linux download is the raw
# binary, and the scripts, styles and svgs are a quarter of their size packed.
# The zip is not kept in git; every release target makes it first.
assets-pack:
	@go run ./build/assetpack internal/gui/assets internal/gui/assets.zip

build:
	go build -tags "$(TAGS)" -trimpath -ldflags="$(LDFLAGS)" -o jianmu .

cli:
	CGO_ENABLED=0 go build -tags nogui -trimpath -ldflags="$(LDFLAGS)" -o jianmu .

build-release: assets-pack
	go build -tags "$(TAGS) packedassets" -trimpath -ldflags="$(LDFLAGS)" -o jianmu .

install:
	go install -tags "$(TAGS)" -trimpath -ldflags="$(LDFLAGS)" .

test:
	go vet -tags "$(TAGS)" ./... && go test -timeout 40m -tags "$(TAGS)" ./...

# The browser regressions under internal/gui/tests: the real pages in
# Playwright's Chromium and WebKit; see internal/gui/tests/README.md.
UI_TESTS = $(wildcard internal/gui/tests/*.test.cjs)
# Files are independent and run in parallel; 1 diagnoses a flaky one.
UI_TEST_CONCURRENCY ?= 2

test-ui:
	@test -n "$(UI_TESTS)" || { echo "no GUI tests found under internal/gui/tests" >&2; exit 1; }
	node --test --test-concurrency=$(UI_TEST_CONCURRENCY) $(UI_TESTS)

# macOS bundle: menu bar app with no Dock icon (LSUIElement).
app: build-release
	@rm -rf jianmu.app
	@mkdir -p jianmu.app/Contents/MacOS jianmu.app/Contents/Resources
	@cp jianmu jianmu.app/Contents/MacOS/jianmu
	@cp build/darwin/jianmu.icns jianmu.app/Contents/Resources/jianmu.icns
	@cp build/darwin/Assets.car jianmu.app/Contents/Resources/Assets.car
	@cp THIRD_PARTY_NOTICES jianmu.app/Contents/Resources/THIRD_PARTY_NOTICES
	@sed 's/@VERSION@/$(VERSION)/' build/darwin/Info.plist > jianmu.app/Contents/Info.plist
	@echo "  jianmu.app"

icons:
	@go run build/icon/gen.go tray internal/gui/tray.png
	@go run build/icon/gen.go tray-flap internal/gui/trayflap
	@# Windows draws an app's icon to the edges of its square: the exe's
	@# icon group (build/windows/winres.json) and the tray's .ico
	@for s in 16 20 24 32 40 48 64 128 256; do \
		go run build/icon/gen.go win $$s build/windows/icon-$$s.png; \
	done
	@go run build/icon/gen.go win-ico internal/gui/icon-windows.ico
	@go run build/icon/gen.go app 1024 internal/gui/icon-1024.png
	@go run build/icon/gen.go app 256 internal/gui/icon-256.png
	@# a page's own icon: opaque to the edges, since iOS blackens a transparent
	@# one, and the same drawing the app tiles carry
	@go run build/icon/gen.go web 512 internal/gui/assets/icons/icon-512.png
	@go run build/icon/gen.go web 180 internal/gui/assets/icons/apple-touch-icon.png
	@go run build/icon/gen.go app 256 site/public/img/icon-256.png
	@go run build/icon/gen.go app 64 site/public/favicon.png
	@rm -rf build/darwin/jianmu.iconset && mkdir -p build/darwin/jianmu.iconset
	@for s in 16 32 128 256 512; do \
		go run build/icon/gen.go app $$s build/darwin/jianmu.iconset/icon_$${s}x$${s}.png; \
		go run build/icon/gen.go app $$((s*2)) build/darwin/jianmu.iconset/icon_$${s}x$${s}@2x.png; \
	done
	@iconutil -c icns build/darwin/jianmu.iconset -o build/darwin/jianmu.icns && rm -rf build/darwin/jianmu.iconset
	@# macOS 26 draws the Dock icon from AppIcon.icon (Xcode 26's actool) in every icon
	@# style: light, dark, and clear or tinted, where the mark's luminance takes the tint;
	@# the compiled Assets.car is kept in the tree so a release needn't have it
	@go run build/icon/gen.go glyph 1024 build/darwin/AppIcon.icon/Assets/jianmu.png
	@rm -rf build/darwin/actool && mkdir -p build/darwin/actool
	@xcrun actool build/darwin/AppIcon.icon --compile build/darwin/actool --platform macosx \
		--minimum-deployment-target 11.0 --app-icon AppIcon --output-partial-info-plist build/darwin/actool/partial.plist >/dev/null
	@mv build/darwin/actool/Assets.car build/darwin/Assets.car && rm -rf build/darwin/actool

release: clean build-release
	@mkdir -p dist
	@cp jianmu dist/jianmu-$(shell go env GOOS)-$(shell go env GOARCH)
	@$(MAKE) --no-print-directory release-cli

release-cli: assets-pack
	@mkdir -p dist
	@for t in $(TARGETS); do \
		os=$${t%/*}; arch=$${t#*/}; ext=""; [ $$os = windows ] && ext=.exe; oa=$$arch; [ $$arch = amd64 ] && oa=x64; \
		echo "  $$os/$$arch (cli)"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -tags "nogui packedassets" -trimpath -ldflags="$(LDFLAGS)" -o dist/jianmu-$$([ $$os = linux ] || echo cli-)$$os-$$oa$$ext . || exit 1; \
	done

# The desktop app for Windows: the system WebView2 needs no cgo, so both
# arches cross-compile. Linked as a GUI program (no console window), with
# the icon, version and a DPI-aware manifest from build/windows. It carries
# jianmu.com, the console program a terminal runs for `jianmu` and waits
# for, which it writes beside itself (internal/update/com.go, #3737).
# The launcher is built without the commit stamped in (-buildvcs=false):
# jianmu.com is written again whenever its bytes differ, and with the
# commit in them every release wrote a new one though the launcher was the
# same (TestLauncherSameAtEveryCommit).
WINVER = $(shell echo $(VERSION) | sed -E 's/^v//; s/[^0-9.].*//')
release-windows: assets-pack
	@mkdir -p dist
	@go run github.com/tc-hib/go-winres@v0.3.3 make --in build/windows/winres.json --arch amd64,arm64 --out rsrc \
		--file-version "$(or $(WINVER),0.0.0)" --product-version "$(or $(WINVER),0.0.0)"
	@for arch in amd64 arm64; do \
		echo "  windows/$$arch (app)"; oa=$$arch; [ $$arch = amd64 ] && oa=x64; \
		CGO_ENABLED=0 GOOS=windows GOARCH=$$arch go build -trimpath -buildvcs=false -ldflags="-s -w" -o internal/update/launcher-$$arch.com ./internal/update/launcher || exit 1; \
		CGO_ENABLED=0 GOOS=windows GOARCH=$$arch go build -tags production,comlauncher,packedassets -trimpath -ldflags="$(LDFLAGS) -H windowsgui" -o dist/jianmu-windows-$$oa.exe . || exit 1; \
	done; rm -f rsrc_windows_*.syso internal/update/launcher-*.com

# The desktop app for Linux links GTK 3 and WebKitGTK 4.1 through cgo, so it
# is built natively, one arch per machine.
release-linux: assets-pack
	@mkdir -p dist
	@echo "  linux/$(shell go env GOARCH) (app)"
	@go build -tags "$(TAGS) packedassets" -trimpath -ldflags="$(LDFLAGS)" -o dist/jianmu-linux-$(shell go env GOARCH) .

clean:
	rm -rf jianmu jianmu.exe jianmu.app dist internal/gui/assets.zip rsrc_windows_*.syso internal/update/launcher-*.com

# Development: the UI is served from internal/gui/assets and the window
# reloads itself when a file there is saved. With fswatch installed, the
# windows stay up and a Go change rebuilds and restarts only the backend
# behind them (build/dev.sh); without it, dev-once runs the app as one
# process. Uses its own ports so a running jianmu keeps serving the agents.
DEV_ADDR ?= 127.0.0.1:3426
DEV_UI ?= 127.0.0.1:3427
DEV_BACKEND ?= 127.0.0.1:3428
DEV_CONTROL ?= 127.0.0.1:3429
dev:
	@if command -v fswatch >/dev/null; then \
	  JIANMU_ADDR=$(DEV_ADDR) JIANMU_DEV_UI=$(DEV_UI) JIANMU_DEV_BACKEND=$(DEV_BACKEND) JIANMU_DEV_CONTROL=$(DEV_CONTROL) build/dev.sh; \
	else $(MAKE) --no-print-directory dev-once; fi

dev-once:
	@go build -tags dev -o jianmu-dev . && echo "  jianmu-dev · UI from internal/gui/assets, reload on save · gateway $(DEV_ADDR)"
	@JIANMU_ADDR=$(DEV_ADDR) JIANMU_DEV_UI=$(DEV_UI) ./jianmu-dev app
