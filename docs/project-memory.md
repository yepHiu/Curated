# 项目记忆：`jav-shadcn`

## 1. 当前仓库事实

### 项目定位

- 当前仓库本质上仍是一个基于 `Vue 3 + TypeScript + Vite 8` 的前端单页应用脚手架。
- `docs/jav-libary.md` 描述的是目标产品蓝图，不代表当前仓库已经落地为完整桌面应用。
- 当前阶段应将仓库视为“JAV-Library 的前端起步壳”，而不是已经具备 `Electron + Go + SQLite + mpv` 能力的成品。

### 当前技术栈

- 框架：`vue@3`
- 语言：`TypeScript`
- 构建：`vite`
- 样式：`tailwindcss@4` + `@tailwindcss/vite`
- 动画：`tw-animate-css`
- UI 体系：`shadcn-vue`
- 基础依赖：
  - `reka-ui`
  - `class-variance-authority`
  - `clsx`
  - `tailwind-merge`
  - `lucide-vue-next`
- 包管理器：`pnpm`

### 已落地目录与入口

- 应用入口：`src/main.ts`
- 根组件：`src/App.vue`
- 全局样式与主题变量：`src/style.css`
- UI 组件目录：`src/components/ui`
- 工具函数：`src/lib/utils.ts`
- 静态资源：`public`、`src/assets`
- 文档目录：`docs`

### 当前运行状态

- `src/main.ts` 仅挂载 `App.vue`。
- `src/App.vue` 仍是按钮级演示页面，不代表真实业务结构。
- 当前已确认存在的基础 UI 组件主要是 `src/components/ui/button`。
- 仓库尚未看到 `vue-router`、`pinia`、API 请求层、业务模块目录、测试体系。

## 2. 前端与 UI 约定

### `shadcn-vue` 事实

- 配置文件：`components.json`
- 配置 schema：`https://shadcn-vue.com/schema.json`
- 风格：`new-york`
- `baseColor`：`neutral`
- 图标库：`lucide`
- Tailwind 全局样式入口：`src/style.css`
- 已启用 CSS Variables 主题方案

### 别名约定

- `@/*` -> `src/*`
- `@/components` -> `src/components`
- `@/components/ui` -> `src/components/ui`
- `@/lib` -> `src/lib`
- `@/lib/utils` -> `src/lib/utils`
- `@/composables` -> `src/composables`

### 协作注意事项

- 这是 `shadcn-vue` 项目，不要默认套用 React 版 `shadcn/ui` CLI 和目录假设。
- 未来新增 UI 时，应优先复用 `src/components/ui` 与 `src/style.css` 中已有语义化主题变量。
- 当前项目仍处于骨架阶段，优先补齐结构化能力，不适合直接零散堆砌业务页面。

## 3. 目标产品架构愿景

根据 `docs/jav-libary.md`，目标产品愿景是一个桌面端 JAV 媒体库应用，目标架构包括：

- 宿主层：`Electron`
- 前端渲染层：`Vue 3 + shadcn-vue`
- 本地服务层：`Go Backend`
- 数据持久化：`SQLite`
- 播放能力：`mpv + FFmpeg`
- 元数据能力：`metatube-sdk-go`
- 日志能力：`zap`

目标产品侧的核心模块方向是合理的：

- `Library Manager` 负责影片库查询与用户操作
- `Scanner Service` 负责目录扫描与入库
- `Metadata Scraper` 负责元数据搜刮与图片缓存
- `Player Controller` 负责控制 `mpv`
- `Database Layer` 负责本地持久化

## 4. 当前差距与关键风险

### 当前差距

- 仓库中尚未看到 `Electron` 运行时、主进程、`preload` 桥接层。
- 仓库中尚未看到 `Go` 后端工程、进程管理、数据库实现或 `SQLite schema`。
- 仓库中尚未看到播放器集成层、命名管道通信层、任务状态系统。
- 现有前端也尚未形成页面骨架、服务层、领域模型、路由和状态管理。

### 关键风险判断

- 最大风险不是技术选型，而是“当前仓库事实”和“目标最终架构”容易被混淆。
- 若直接按最终蓝图组织前端代码，容易在没有桥接协议和服务边界的前提下把 UI 与未来桌面能力耦合在一起。
- 文档中存在 `HTTP / IPC`、`Electron IPC`、`mpv JSON IPC` 多种链路描述，但当前没有统一定义前端应依赖哪一层抽象。
- 扫描、搜刮、缩略图生成、缓存刷新本质上都是后台任务，目前方案缺少任务状态、事件推送、失败重试和日志关联设计。
- 数据结构偏向 MVP，尚未覆盖去重、扫描状态、资源缓存、来源追踪、播放历史等后续关键能力。

## 5. 推荐的演进原则

- 先建设前端应用骨架，再接入桌面能力。
- UI 应只依赖统一的前端服务层，不直接感知 `Go`、`SQLite`、`mpv` 的具体通信细节。
- Electron `preload` 层未来应作为 Renderer 的唯一桥接入口，避免前端混用多套调用方式。
- 在真正接入后端前，应先定义好 `domain models`、`DTOs`、`event types`、`error codes`。
- 设计文档必须区分“当前已实现”“未来目标”“待决策项”，避免把规划写成事实。

## 6. 建议的阶段化落地顺序

### 第一阶段：前端骨架

优先补齐以下能力：

1. `router`
2. `views`
3. `layouts`
4. `services`
5. `types`
6. `stores`

### 第二阶段：桥接协议设计

- 定义前端服务接口
- 定义事件订阅模型
- 定义扫描、搜刮、播放、设置相关 DTO
- 定义错误码与任务状态枚举

### 第三阶段：桌面运行时接入

- 接入 `Electron`
- 增加主进程与 `preload`
- 设计安全边界与调用白名单
- 管理 `Go` 子进程生命周期

### 第四阶段：后台任务系统

- 扫描任务
- 搜刮任务
- 图片缓存任务
- 日志与失败重试

### 第五阶段：播放器与体验完善

- 接入 `mpv`
- 同步播放状态
- 处理进度、暂停、结束事件
- 完善详情页、播放器页和设置页联动

## 7. 当前明确不应假设的能力

- 不要假设仓库已经是 Electron 项目。
- 不要假设 Renderer 可以直接调用本地数据库或本地文件系统能力。
- 不要假设现有前端已经具备扫描、搜刮、播放、设置持久化能力。
- 不要假设 React 版 `shadcn/ui` CLI 能直接驱动当前 `shadcn-vue` 配置。

## 8. 下一轮开发的优先事项

推荐优先顺序：

1. 清理模板残留，修正 `README.md` 与应用标题。
2. 建立前端应用目录骨架与页面路由。
3. 定义面向未来桌面层的前端服务接口与类型。
4. 先做可替换的 mock 数据流，再考虑真实桌面桥接。
5. 在协议稳定后，再接入 `Electron`、`Go`、任务系统与播放器。

## 9. 维护说明

- 本文用于沉淀“当前仓库事实”和“长期演进判断”。
- 当仓库真正引入 `Electron`、`Go`、`SQLite`、播放器桥接或新的目录骨架时，应优先更新本文。
- 若设计文档与实际代码不一致，应以代码现状为事实，并在文档中明确标注差距。
