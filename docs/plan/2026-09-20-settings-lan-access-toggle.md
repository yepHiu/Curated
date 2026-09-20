# Settings 局域网访问开关

日期：2026-09-20

## 设计帧

| 项 | 决定 |
| --- | --- |
| Surface and job | 设置 → 网络。用户任务是决定本机 HTTP 服务是否对局域网设备开放。 |
| Existing precedent | 通用页的开关（`launchAtLogin`）+ 网络页代理卡片。 |
| Hierarchy and density | 网络分区第一张卡「局域网访问」，其下才是出站代理。开关是主操作；地址列表与重启提示是次要状态，重启提示覆盖开启和关闭。 |
| State matrix | Web：开关可直接保存；偏好与当前监听不一致时提示完全退出后生效，包括关闭但当前仍在局域网监听；开启时展示局域网 URL。Mock：不可用。错误就地显示，不新增 Toast。 |
| System impact | 本地设置卡片 + `library-config.cfg` 键 + Settings DTO。不新增 UI 基元或 token。 |

## 当前事实

默认开发 `127.0.0.1:8080`、release `127.0.0.1:8081`。设置页「网络」可持久化 `lanEnabled`。PIN 与局域网访问互相独立。

## 目标行为

1. `GET/PATCH /api/settings` 增加 `lanEnabled`（偏好）、`lanListening`（当前进程是否非 loopback）、`lanAccessUrls`（按当前端口列出本机私网 IPv4 URL）。
2. `lanEnabled` 写入 `library-config.cfg`，启动时合并后按偏好改写实际监听地址：关闭强制 `127.0.0.1:<port>`，开启把 loopback/未指定主机改成 `0.0.0.0:<port>`，已显式指定的非 loopback 主机保持不变。
3. 开启不要求应用 PIN；开启后也可以关闭 PIN。Host/CORS 仍跟当前真实监听走，避免尚未改绑时放宽 DNS rebinding 防护。
4. 改绑需要完全退出并重新打开 Curated（托盘退出，不是只关窗口）。界面在偏好与当前监听不一致时说明这一点。
5. Mock 模式不假装已经对局域网提供服务。

## 非目标

- 不在本切片做 HTTP 热重绑。
- 不自动改 Windows 防火墙。
- 不恢复可写的「本机/LAN 分叉 PIN」策略。

## 实施状态

2026-09-20 已落地：设置 → 网络开关写入 `library-config.cfg` 的 `lanEnabled`；`GET/PATCH /api/settings` 返回偏好、当前是否非 loopback 监听、私网 IPv4 URL。不要求 PIN，也不因局域网访问禁止关闭 PIN；改绑需完全退出。Host 校验跟当前真实监听走。

2026-09-21 Review 修复：重启提示条件统一为 Web API 模式下 `lanEnabled !== lanListening`，补齐关闭后仍在监听的提示；组件回归测试覆盖偏好 / 监听的四种组合与 Mock 隐藏提示。
