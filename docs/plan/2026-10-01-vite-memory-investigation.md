# 5173 开发服务内存耗尽调查

调查日期：2026-10-01（Asia/Shanghai）。

## 结论

已验证的异常增长来源是 **Vite 文件监听通过挂载的 DMG 内的符号链接越过项目边界，递归遍历系统应用目录**。

链路如下：

```text
Vite 监听仓库根目录
  → .workspace/dmg-preview-mount（仍挂载的 DMG）
  → Applications → /Applications
  → 系统应用及其内部资源目录
  → 大量路径、文件状态与监听对象进入 Node 进程
```

之前收窄监听与依赖扫描范围的提交为 `dc0c7fff`。本次对照实验将主要原因进一步定位到文件监听范围；限制 HTML 依赖入口属于额外防护，不能把它单独当作已证明的根因。

## 崩溃证据

- `.workspace/dev-logs/frontend.err.log` 记录原 Vite PID `33860` 的 `FATAL ERROR: Ineffective mark-compacts near heap limit Allocation failed - JavaScript heap out of memory`。
- 最后一轮 GC 在进程启动约 769 秒后发生：堆从约 4091 MB 回收至约 4081 MB，接近默认约 4 GB 堆上限。
- 这证明 Node 堆达到上限且 GC 无法回收足够空间。它不等价于整台电脑物理内存耗尽。
- 8080 Go 后端持续正常，崩溃对象是 5173 对应的 Vite/Node 进程。

## 文件系统与依赖实现

- `mount` 确认 `.workspace/dmg-preview-mount` 为实际挂载的只读 HFS DMG。
- 其中 `Applications` 是指向 `/Applications` 的符号链接。
- Vite 8.1.5 的 `resolveChokidarOptions` 默认排除 `.git`、`node_modules`、测试输出、缓存和构建输出，没有自动排除 `.workspace`；Git 忽略规则不等于 Vite 监听忽略规则。
- 该版本 Vite 的监听器以仓库根目录为入口，Chokidar 默认跟随符号链接。实际实验确认监听路径进入了该链接下的系统应用目录。
- 工作区本身约 3.4 GB，测试影片库为 1776 个文件；文件的磁盘体积不能直接换算成 Node 堆用量。

## 受限对照实验

使用本仓库同一 Node/Vite/插件创建独立开发服务器对象，**关闭依赖自动发现、不打开浏览器、不触发页面转换，也不启动新的影片扫描或刮削**，只观察文件监听。

Node 设置 512 MB 堆上限；监测脚本在约 300 MB 已用堆、目录数量阈值或时间上限处主动关闭。未尝试再次耗尽 4 GB，未中断 5173 服务。

| 配置 | 观测时间 | 监听目录 | 监听条目 | 链接下的系统应用目录 | Node 已用堆 | RSS |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 原监听配置 | 45 秒 | 89,910 | 439,807 | 85,625 | 305 MiB | 581 MiB |
| 只排除 DMG 的 Applications 链接 | 27 秒 | 4,630 | 14,747 | 0 | 32 MiB | 214 MiB |
| 当前修复后的排除规则 | 27 秒 | 217 | 2,392 | 0 | 37 MiB | 192 MiB |

原配置的目录数从 3 秒时的 15,328 增长至 45 秒时的 89,910，仍未稳定，达到阈值后主动停止。后两组的监听目录与条目数从首个采样开始保持稳定。堆数值受 GC 时机影响，不能只用单个瞬时数值判断泄漏。

另外试过单独设置 `followSymlinks: false`，本机运行结果仍进入链接下的系统应用目录，因此没有把该开关当作本次修复依据；显式路径排除已经验证有效。

本地复现脚本与原始采样记录位于 `.workspace/vite-memory-investigation/`，不纳入源码提交。所有探测进程均已关闭。

## 已落地处理与边界

- `vite.config.ts` 的 `server.watch.ignored` 排除 `.workspace`、`.playwright-cli`、`backend/runtime`、`release`。
- `optimizeDeps.entries` 限定为 `index.html`，避免自动发现无关 HTML；本次实验关闭了依赖发现仍可复现异常，所以此项不是必要触发条件。
- 没有通过提高正式开发进程的 Node 堆上限掩盖问题，没有删除测试库或改动用户数据。
- 重启后的实际首页已完成加载并显示推荐卡，HTTP 与后端代理均返回成功；实际 Vite 进程 RSS 从早期约 380–400 MiB 回落至运行约 5 分钟时的 251 MiB。
- 原崩溃进程没有保留堆快照，不能精确枚举当时 4 GB 堆内每类对象，也未证明所有可能的长期内存问题均已消失。已复现并隔离的是此次主要的异常监听与内存增长来源。

后续开发产物和安装器挂载目录继续放在已排除的本地目录中；若引入新的产物位置，检查其中指向仓库外部的符号链接及监听范围。
