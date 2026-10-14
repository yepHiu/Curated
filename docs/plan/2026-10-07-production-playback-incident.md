# 2026-10-07 生产播放故障调查

## 范围与结论

本机已安装 Server 1.7.10（buildStamp 20261006.190051）、Desktop 0.2.5，本机 playback-settings.json 开启 preferNative。调查为只读，不修改生产配置、数据库、安装文件，不发布版本。

主要故障是 F: 文件访问阻塞，Windows 同时明确报告 USB 存储 I/O 错误。尚不能区分硬盘本体、硬盘盒、USB 线材、供电或控制器故障。软件播放准备与回退机制会放大等待，不能依靠切换播放引擎解决底层文件不可读。

## 证据（北京时间）

- 生产 Server 日志：2026-10-07 22:48:03 ROE-392 direct playback-session 请求 15194 ms；22:48:12 为 18229 ms；22:48:28 同时有 9424 ms、44383 ms 的请求完成。
- 多次 GET playback 在 HLS 启动阶段收到 context canceled，耗时 14202–48741 ms；普通影片详情请求约 1–2 ms。
- 生产数据库只读查询：ROE-392 文件为 `F:\Curated-Nas\ROE-392\ROE-392.mp4`，无持久媒体探测缓存；JUQ-711 为同盘 MP4，有 h264/aac、7083.136 秒、schema 2 的探测缓存。
- 独立 Python 文件访问：ROE-392 的 os.stat 长时间不返回；JUQ-711 的 os.stat 约 0.000055 秒返回，大小 5218349136 字节，但读取前 64 KiB 长时间不返回。三个调查读取进程已终止，未继续扫描该盘。
- Windows System 日志：故障时间段持续出现 disk Event 153（磁盘 3 的 I/O 重试）；22:49:52 的 Microsoft-Windows-Ntfs Event 140 明确指向 F: / HarddiskVolume7，事务日志刷新失败，I/O 设备错误，USB 设备 ASMT ASMT105x。22:53 仍在重复。
- Get-Volume 返回 Healthy/OK，但不能推翻上述实际 I/O 错误。Get-Partition 无法取得 F: 分区对象，未据此推断物理拓扑。

## 代码链路

1. `electron/playback-coordinator.ts` 的 api() 对请求设 5000 ms 期限；原生起播先 POST playback-session，body 为 mode=direct。已安装编译 JS 也确认这两个行为。
2. `backend/internal/app/app.go` 即使 forceDirect=true，也先执行 resolvePlaybackMediaInfo 和 resolvePlaybackDurationSec。持久探测缓存读取前会 os.Stat；缺少媒体时长还会尝试额外时长探测。
3. 文件系统调用不随 HTTP context 立即取消，因此磁盘重试可能使 Server 请求继续等待。
4. `src/services/desktop-playback-service.ts` 原生启动抛错后导航到 engine=web；Web 重新请求 playback，在探测信息缺失或能力不匹配时可能进入 HLS。日志确认 HLS 被取消，但没有原生错误持久日志，无法直接确认每次回退的具体错误码。

## 处理建议与待验证项

优先停止对 F: 的播放、扫描和写入；正常退出占用该盘的应用后，检查 USB 连接、硬盘盒供电与接口，恢复连接后首先验证少量文件读取。若错误持续，先保全重要数据并进一步诊断硬件；不直接对异常盘执行修复写入或大范围扫描。

存储恢复后复测 ROE-392 / JUQ-711，核对原生启动耗时和系统事件是否消失。当前尚未完成恢复后的播放验收。

软件后续可独立优化：原生 direct 描述避免强制同步完整探测；为慢存储提供可见的准备状态；区分存储不可用与引擎故障，避免自动回退造成重复探测；记录原生失败阶段和错误码。此处为建议，本次未实施，也未声称能够中断 Windows 底层 I/O。
