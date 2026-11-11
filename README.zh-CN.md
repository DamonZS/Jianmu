<div align="center">

# Jianmu（建木）

### 所有 Agent 的模型，一处搞定。

**一个本地网关，统一管理每个 Agent 的模型、供应商、额度和路由。**<br>
让 Claude Code、Codex、Gemini CLI 等都能用上你能访问的任何模型，<br>
额度用完时，它会悄悄切到你的下一个账号。

[![License](https://img.shields.io/badge/license-MIT-111111)](LICENSE) ![Windows](https://img.shields.io/badge/Windows-0078D4?logo=windows&logoColor=white) ![macOS](https://img.shields.io/badge/macOS-000000?logo=apple&logoColor=white) ![Linux](https://img.shields.io/badge/Linux-FCC624?logo=linux&logoColor=black) ![Docker](https://img.shields.io/badge/Docker-2496ED?logo=docker&logoColor=white)

**[下载](https://github.com/DamonZS/Jianmu/releases)** &nbsp;·&nbsp; **[官网](https://toporeduce.cn)** &nbsp;·&nbsp; **[完整参考](docs/reference.md)** &nbsp;·&nbsp; **[插件](https://github.com/DamonZS/jianmu-commonity)** &nbsp;·&nbsp; [English](README.md) · **简体中文**

</div>

<br>

## Jianmu 适合你吗？

每个 Agent 都把模型写在自己的配置文件里，格式各不相同，key 和 base URL 各管各的。花钱买的订阅只能在一个 Agent 里用。Jianmu 把这些收进一个应用和一个本地网关：

- **任意 Agent 用任意模型**，在一张表里选。Jianmu 按每个 Agent 自己的格式写入它的配置。
- **一个供应商处处能用**：API key、本地模型，或者你登录着的订阅。
- **额度用完也不断**：下一个账号自动接上，路由组还能换到别的模型或供应商。
- **只维护一份**：指令、MCP 服务器和技能只写一次，写进所有 Agent。

如果你用不止一个 Agent、不止一个供应商或不止一个账号，它就适合你。如果只用一个 Agent、只用它自家的订阅，而且够用，那就不需要它。

## 快速开始

**1 · 安装**。到 [Releases](https://github.com/DamonZS/Jianmu/releases) 下载适合你系统的版本。

**2 · 添加供应商**。打开 Jianmu，进入「供应商」添加。选一个预设并粘贴 key，或者登录一个订阅。我们自己的中转 `https://api.toporeduce.cn` 排在最前。

**3 · 选模型**。在 Agents 页给每个 Agent 选一个模型，开一个新会话就用上了。

其他任何能填 base URL 的工具也都能用这个网关：

```sh
export OPENAI_BASE_URL=http://127.0.0.1:3425/v1
export ANTHROPIC_BASE_URL=http://127.0.0.1:3425
```

## 功能

| | |
| :-- | :-- |
| **供应商与账号** | 多密钥、多账号、订阅桥接、额度读取与自动切换 |
| **网关路由** | 路由组、回退、九种路由策略、请求中间件与会话聚合 |
| **Agent 接线** | 按各 Agent 自己的配置格式写入网关地址，支持漂移检测、干净断开与 WSL |
| **资料库** | 指令、MCP 服务器、技能统一维护，一次写进所有 Agent |
| **插件** | 内置插件可关停；社区插件在独立的 Bun 宿主里运行，工具以 MCP 形式提供给所有 Agent |
| **共享记忆** | 跨 Agent 的短文本记忆，通过 MCP 读写 |
| **同步与备份** | 口令加密的配置包，可存到 WebDAV、S3 或 GitHub |
| **四种形态** | 桌面窗口与托盘、`jianmu web`（浏览器或手机）、`jianmu tui`、`jianmu serve`（纯网关） |

`127.0.0.1:3425` 上的网关同时支持 OpenAI Chat、OpenAI Responses、Anthropic Messages 和 Gemini，并在它们之间互相转换。

## 从源码构建

需要 Go 1.26.8 或更高。

```sh
make build            # 桌面版（本机平台）
make cli              # 无界面版，可交叉编译
make build-release    # 桌面版，前端资源打包成 zip 内嵌
make test             # go vet 与全部测试
make test-ui          # 浏览器回归测试（node --test，需要 Playwright）
make icons            # 重新生成托盘与应用图标
```

Linux 桌面版用 cgo 链接 GTK 3 与 WebKitGTK 4.1，只能在本机原生构建（Debian/Ubuntu：`libgtk-3-dev libwebkit2gtk-4.1-dev libwayland-dev libfontconfig1-dev`）。Windows 桌面版走系统 WebView2，不需要 cgo，可用 `make release-windows` 交叉编译。

## 发布与 CI

| 工作流 | 触发 | 做什么 |
| --- | --- | --- |
| `ci.yml` | push 到 `main`、pull request | 运行 `make test`（`go vet` 与 `go test ./...`） |
| `docker.yml` | push 到 `main` | 构建多架构镜像（amd64 与 arm64），推送到 `ghcr.io/damonzs/jianmu` |
| `release.yml` | push 到 `main`、`v*` 标签 | 确定下一个版本号，按提交生成更新说明，构建所有目标，生成 `SHA256SUMS` 并发布 GitHub Release |

发布产物：

- `jianmu-windows-x64.exe`、`jianmu-windows-arm64.exe`
- `jianmu-macos-x64.dmg`、`jianmu-macos-arm64.dmg`（未公证）
- `jianmu-linux-x64`、`jianmu-linux-arm64`
- Source code (zip) 与 Source code (tar.gz)

产物文件名必须与更新器一致（`internal/update/update.go` 的 `BinaryAsset` 与 `AppAsset`）。

## 更新源

自更新通过 `https://toporeduce.cn/api/latest` 获取版本信息（可用 `JIANMU_UPDATE_FEED` 覆盖）。该接口需要返回：

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

`sha256` 是必需的，缺失的资产会被拒绝安装。

## 文档

- `docs/subsystems/`：各子系统的设计与契约。
- `docs/code-standards.md`：代码约定。
- `docs/reference.md`：完整参考手册（英文）。

## 许可

[MIT](LICENSE)。版权所有 Damon，[toporeduce.cn](https://toporeduce.cn)。第三方组件的许可见 `THIRD_PARTY_NOTICES`，也可在「设置 › 关于 › 开源许可」中查看。
