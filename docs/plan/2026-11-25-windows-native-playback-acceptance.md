# Windows 原生播放人工验收

日期：2026-11-25。状态：步骤已写成清单，**尚未在本环境执行**。关联 REQ-0058、REQ-0059。

Linux 上的 CI 只运行假引擎单元测试，并明确把实机项标成 skipped 或 manual-windows。不能把 CI 通过写成 Windows 已验收。清单实现在 `electron/native-playback-acceptance.ts`，检查命令：

```bash
pnpm test:electron -- electron/native-playback-acceptance.test.ts electron/playback-engine-missing.test.ts electron/playback-resources.test.ts
```

## 在 Windows 上才做的步骤

准备一台已安装或可安装 Curated Desktop 的 Windows 机器，并准备至少三份片源：已记录的 CD1、另一份普通 MP4、一份 HEVC 或此前 Chromium 直放不友好的文件。记录机器、Windows 版本、主屏和副屏 DPI。

1. **缺引擎**：删掉或改名随包 `mpv.exe` 后打开播放。设置里应说明没有找到引擎，播放回到 Web，且不应先弹出原生窗口。
2. **更多片源**：三份片源分别用原生播放器播放至少一分钟，记录暂停、seek、倍速、音量、停止，以及分部续播是否仍落在原来的文件和时间。
3. **整片同步**：选一份完整成片播到结尾，或连续播放不少于片长的 80%。记录音画是否逐渐拉开、结束后进度是否停在结尾附近。五分钟片段不能代替这一项。
4. **混合 DPI**：把原生窗口从高 DPI 屏拖到另一块不同 DPI 的屏幕，再拖回来。记录画面是否拉伸、控件是否错位、最小化后能否恢复。
5. **安装包**：用正式安装包做一次干净安装、一次覆盖升级和一次卸载。确认原生引擎仍只来自安装目录，卸载后不留下播放进程。

每一项单独记录结果。任一项没做，就不能把 REQ-0058 或 REQ-0059 标成 verified。
