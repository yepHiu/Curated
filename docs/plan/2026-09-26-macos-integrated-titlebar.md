# macOS 窗口栏融合

目标：让系统红黄绿按钮与 Curated 顶部导航融合，保留系统关闭、最小化、全屏行为。

- Electron 在 macOS 使用 `hiddenInset`，保留原生按钮，位置为 `(24, 20)`。按用户「离左侧和顶部更远」的反馈增加留白；WorkBuddy 截屏中的按钮被系统截屏指示遮挡，此值为布局调整值，并非对其精确测量。
- preload 仅增加只读 `windowChrome` 能力标识，前端据此启用样式；网页与 Windows / Linux 不启用。
- 按用户后续要求，系统按钮独占侧栏 40px 顶部区域，Outfit 品牌标题在其下方，去掉品牌下方分割线。宽窗口右侧工具栏保持原来顶部位置和高度，不随侧栏增加留白。顶部延续侧栏背景，侧栏宽度与折叠品牌图案保留；窄窗口隐藏侧栏时，内容区（含阅读器）和抽屉预留按钮空间。
- 工具栏与侧栏顶部空白处允许拖动；表单、导航和菜单所在区域明确排除拖动。
- 阅读器、锁屏与故障页保留顶部拖动区域。背景使用既有主题，不增加独立标题栏色块。
- 验证：Electron bridge 测试、前端类型检查与构建、真实 macOS 窗口的深浅主题、收起侧栏、菜单与搜索、拖动及全屏。跨浏览器显示缩放全套不在本次默认执行范围。

## 验证记录

- 2026-09-26，macOS Electron 42 开发窗口（创建尺寸 1280 × 820）：检查深浅主题、展开/收起侧栏、搜索输入与清除、添加媒体弹窗和 Escape 关闭、原生全屏进入及菜单退出。执行了顶部拖动与最小化操作；未对窗口坐标变化做数值测量。
- `pnpm test:electron`：34 项通过；`pnpm build:electron:main` 通过。
- `NODE_OPTIONS=--no-experimental-webstorage pnpm test src/layouts/AppShell.test.ts src/components/jav-library/AppSidebar.test.ts`：31 项通过。默认 Node 实验性 Web Storage 与测试的 jsdom localStorage 冲突；关闭该实验特性后通过，未修改业务存储逻辑。
- Web API 生产构建（含类型检查）通过；前端修改文件 ESLint 无错误；Electron TS 文件未纳入当前 ESLint 配置，由编译与桌面测试检查。
- 本次未执行完整跨浏览器、DPR、90%–150% 缩放或 Windows 实机矩阵，未单独测量当前窗口 DPR。窄窗口/阅读器/锁屏避让已实现，仍需后续人工矩阵验证。
