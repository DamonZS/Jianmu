# Jianmu

一个本地网关，把每个 Agent 的模型、供应商、额度和路由收在一个地方管。

Jianmu 是一个本地网关，把每个 Agent 的模型、供应商、额度和路由收在一个地方管。兼容 `@magpie-community/*` 社区插件。

## 它能做什么

- **供应商与账号**：多密钥、多账号、订阅桥接、额度读取与自动切换。
- **网关路由**：路由组、回退、九种路由策略、请求中间件与会话聚合。
- **Agent 接线**：按各 Agent 自己的配置格式写入网关地址，支持漂移检测、断开与 WSL。
- **Library**：指令、MCP 服务器、Skills 统一维护，一次写进所有 Agent。
- **插件**：内置插件可关停；社区插件在独立的 Bun 宿主里运行，工具以 MCP 形式提供给所有 Agent。
- **共享记忆**：跨 Agent 的短文本记忆，通过 MCP 读写。
- **同步与备份**：口令加密的配置包，可存到 WebDAV、S3 或 GitHub。
- **四种形态**：桌面窗口与托盘、`jianmu web`（浏览器/手机）、`jianmu tui`、`jianmu serve`（纯网关）。

## 安装

Windows、macOS、Linux 的发布产物见 [Releases](https://github.com/DamonZS/Jianmu/releases)。

## 构建与发布

需要 Go 1.26.8 或更高。

```bash
make build            # 桌面版（本机平台）
make cli              # 无界面版，可交叉编译
make build-release    # 桌面版，前端资源打包成 zip 内嵌
make assets-pack      # 单独打包前端资源为 internal/gui/assets.zip
make test             # vet + 全部测试（-tags "$(TAGS)"）
make test-ui          # 浏览器回归测试（node --test，需 Playwright 的 Chromium/WebKit）
make icons            # 重新生成托盘与应用图标
```

Linux 桌面版用 cgo 链接 GTK 3 与 WebKitGTK 4.1，只能在本机原生构建（Debian/Ubuntu：`libgtk-3-dev libwebkit2gtk-4.1-dev libwayland-dev libfontconfig1-dev`）；Windows 桌面版走系统 WebView2，不需要 cgo，可以交叉编译。

Windows 桌面版的完整发布产物：

```bash
make release-windows  # 生成 dist/jianmu-windows-<arch>.exe
```

它先用 `go build ./internal/update/launcher` 产出 `internal/update/launcher-<arch>.com`（`jianmu.com` 的载荷），再链接带 `comlauncher` 标签的主程序。

### CI

| 工作流 | 触发 | 做什么 |
| --- | --- | --- |
| `.github/workflows/ci.yml` | push 到 `main`、pull request | 装 GTK/WebKitGTK/Xvfb，跑 `make test`（`go vet` + `go test ./...`，用 Makefile 的 tags） |
| `.github/workflows/release.yml` | 推送 `v*` 标签 | 矩阵构建 Windows 桌面版（amd64/arm64）与全平台 CLI 版，生成 `SHA256SUMS`，上传到 GitHub Release |

发布 workflow 只用 Makefile 已有的目标（`release-windows`、`release-cli`），并在构建前先跑 `assets-pack`。版本号从标签去掉前导 `v` 后注入 `-X main.version=`，与本地 `make release` 一致。

macOS 桌面版（`jianmu-darwin-<arch>.zip`，已签名并公证）不在 CI 里构建，需要 Mac 与 Apple 签名密钥；Linux 桌面版同理不做交叉构建，CI 不产出 `jianmu-linux-<arch>`，Linux 用户默认拿 CLI 版。

发布资源的文件名必须与更新器一致（`internal/update/update.go` 的 `BinaryAsset` / `AppAsset`）：

- 桌面版：`jianmu-windows-x64.exe`、`jianmu-windows-arm64.exe`、`jianmu-darwin-arm64.zip`
- CLI 版：`jianmu-cli-<os>-<arch>[.exe]`

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

`sha256` 是必需的，缺失会被拒绝安装。

## 文档

- `docs/subsystems/`：各子系统的设计与契约。
- `docs/code-standards.md`：代码约定。
- `docs/reference.md`：完整参考手册。

## 许可

MIT。见 [LICENSE](LICENSE)。第三方组件的许可见 `THIRD_PARTY_NOTICES`，构建产物在「设置 › 关于 › 开源许可」中显示同一份内容。
