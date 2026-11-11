<div align="center">

# Jianmu

### Every agent's model. One place.

**One local gateway for every agent's models, providers, quotas and routing.**<br>
Point Claude Code, Codex, Gemini CLI and more at any model you can reach,<br>
and when a quota runs out, it quietly moves on to your next account.

[![License](https://img.shields.io/badge/license-MIT-111111)](LICENSE) ![Windows](https://img.shields.io/badge/Windows-0078D4?logo=windows&logoColor=white) ![macOS](https://img.shields.io/badge/macOS-000000?logo=apple&logoColor=white) ![Linux](https://img.shields.io/badge/Linux-FCC624?logo=linux&logoColor=black) ![Docker](https://img.shields.io/badge/Docker-2496ED?logo=docker&logoColor=white)

**[Download](https://github.com/DamonZS/Jianmu/releases)** &nbsp;·&nbsp; **[Website](https://toporeduce.cn)** &nbsp;·&nbsp; **[Reference](docs/reference.md)** &nbsp;·&nbsp; **[Plugins](https://github.com/DamonZS/jianmu-commonity)** &nbsp;·&nbsp; **English** · [简体中文](README.zh-CN.md)

</div>

<br>

## Is Jianmu for you?

Each agent keeps its model in its own file, in its own format, with its own keys and base URLs. A subscription you pay for works in one agent and nowhere else. Jianmu puts all of that behind one app and one local gateway:

- **Pick any model for any agent** from one list. Jianmu writes the agent's own config in its own format.
- **Use one provider everywhere**: an API key, a local model, or a subscription you are signed in to.
- **Keep going when a quota runs out**: the next account takes over, and a routing group can move on to another model or provider.
- **Keep one source of truth**: instructions, MCP servers and skills are written once and put into every agent.

It is for you if you use more than one agent, more than one provider, or more than one account. If one agent on its vendor's own plan is enough, you don't need it.

## Quick start

**1 · Install.** Download the build for your system from [Releases](https://github.com/DamonZS/Jianmu/releases).

**2 · Add a provider.** Open Jianmu, go to Providers and add one. Pick a preset and paste a key, or sign in with a subscription. Our own relay, `https://api.toporeduce.cn`, is listed first.

**3 · Pick a model** for each agent on the Agents page. A new session is on the new model.

Anything else that takes a base URL can use the gateway too:

```sh
export OPENAI_BASE_URL=http://127.0.0.1:3425/v1
export ANTHROPIC_BASE_URL=http://127.0.0.1:3425
```

## What it does

| | |
| :-- | :-- |
| **Providers and accounts** | Many keys, many accounts, subscription bridges, quota reading and automatic switching |
| **Gateway routing** | Routing groups, fallbacks, nine routing strategies, request middleware and session aggregation |
| **Agent wiring** | Writes the gateway address into each agent's own config format, detects drift, disconnects cleanly, and supports WSL |
| **Library** | Instructions, MCP servers and skills maintained in one place and written to every agent |
| **Plugins** | Built-in plugins can be switched off. Community plugins run in an isolated Bun host and offer their tools to every agent over MCP |
| **Shared memory** | Short notes shared across agents, read and written over MCP |
| **Sync and backup** | Passphrase-encrypted config bundles stored on WebDAV, S3 or GitHub |
| **Four forms** | Desktop window and tray, `jianmu web` (browser or phone), `jianmu tui`, and `jianmu serve` (gateway only) |

The gateway at `127.0.0.1:3425` speaks OpenAI Chat, OpenAI Responses, Anthropic Messages and Gemini, and translates between them.

## Build from source

Requires Go 1.26.8 or newer.

```sh
make build            # desktop build for this platform
make cli              # headless build, can be cross-compiled
make build-release    # desktop build with the UI assets packed into a zip
make test             # go vet and all tests
make test-ui          # browser regression tests (node --test, needs Playwright)
make icons            # regenerate tray and app icons
```

The Linux desktop build links GTK 3 and WebKitGTK 4.1 through cgo and must be built natively (Debian or Ubuntu: `libgtk-3-dev libwebkit2gtk-4.1-dev libwayland-dev libfontconfig1-dev`). The Windows desktop build uses the system WebView2, needs no cgo, and can be cross-compiled with `make release-windows`.

## Releases and CI

| Workflow | Trigger | What it does |
| --- | --- | --- |
| `ci.yml` | push to `main`, pull request | Runs `make test` (`go vet` and `go test ./...`) |
| `docker.yml` | push to `main` | Builds a multi-arch image (amd64 and arm64) for `ghcr.io/damonzs/jianmu` |
| `release.yml` | push to `main`, `v*` tags | Picks the next version, writes the update notes from the commits, builds every target, writes `SHA256SUMS` and publishes the GitHub release |

Release assets:

- `jianmu-windows-x64.exe`, `jianmu-windows-arm64.exe`
- `jianmu-macos-x64.dmg`, `jianmu-macos-arm64.dmg` (not notarized)
- `jianmu-linux-x64`, `jianmu-linux-arm64`
- Source code (zip) and Source code (tar.gz)

Asset names must match what the updater expects (`BinaryAsset` and `AppAsset` in `internal/update/update.go`).

## Update feed

Self-update reads `https://toporeduce.cn/api/latest` (override with `JIANMU_UPDATE_FEED`). It must return:

```json
{
  "version": "0.2.0",
  "notes": "...",
  "url": "https://github.com/DamonZS/Jianmu/releases/tag/v0.2.0",
  "assets": {
    "jianmu-windows-x64.exe": { "url": "...", "size": 123, "sha256": "..." }
  }
}
```

`sha256` is required; an asset without it is refused.

## Documentation

- `docs/subsystems/`: design and contracts of each subsystem.
- `docs/code-standards.md`: code conventions.
- `docs/reference.md`: the full reference manual.

## License

[MIT](LICENSE). Copyright (c) Damon, [toporeduce.cn](https://toporeduce.cn). Third-party licenses are in `THIRD_PARTY_NOTICES`, shown as well in Settings › About › Open source licenses.
