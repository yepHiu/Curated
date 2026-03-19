# JAV-Library 项目设计文档

---

## 目录

1. [系统架构](#1-系统架构)
2. [技术栈](#2-技术栈)
3. [核心模块](#3-核心模块)
   - 3.1 影片库管理
   - 3.2 扫描服务
   - 3.3 元数据搜刮
   - 3.4 播放器
4. [数据结构](#4-数据结构)
5. [文件存储结构](#5-文件存储结构)
6. [播放器设计](#6-播放器设计)
7. [UI 页面设计](#7-ui-页面设计)
8. [设置系统](#8-设置系统)
9. [性能优化](#9-性能优化)
10. [日志系统](#10-日志系统)

---

## 1. 系统架构

### 整体架构图

```
┌───────────────────────────────┐
│           Electron             │
│                                │
│  ┌───────────────┐             │
│  │  Vue UI       │             │
│  │  (Renderer)   │             │
│  └───────┬───────┘             │
│          │ IPC                  │
│  ┌───────▼────────┐            │
│  │ Electron Main  │            │
│  └───────┬────────┘            │
└──────────│─────────────────────┘
           │ HTTP / IPC
           ▼
┌───────────────────────────────┐
│          Go Backend            │
│                                │
│  ├─ Library Manager            │
│  ├─ Scanner Service            │
│  ├─ Metadata Scraper           │
│  ├─ Database Layer (SQLite)    │
│  └─ Player Controller          │
└──────────┬─────────────────────┘
           │ JSON IPC
           ▼
      mpv Player
```

### 核心模块职责


| 模块              | 职责                                  |
| ------------------- | --------------------------------------- |
| Library Manager   | 影片库的增删改查管理                  |
| Scanner Service   | 扫描目录、识别影片、触发入库          |
| Metadata Scraper  | 调用 metatube-sdk-go 搜刮元数据和封面 |
| Player Controller | 启动/控制 mpv 播放器                  |
| Database Layer    | SQLite 数据持久化                     |

---

## 2. 技术栈


| 层级       | 技术选型        |
| ------------ | ----------------- |
| 程序框架   | Electron        |
| 后端语言   | Go              |
| 前端框架   | Vue 3           |
| UI 组件库  | shadcn-vue      |
| 播放器     | mpv + FFmpeg    |
| 数据库     | SQLite          |
| 元数据搜刮 | metatube-sdk-go |
| 日志库     | zap（Go）       |

---

## 3. 核心模块

### 3.1 影片库管理（Library Manager）

负责影片的增删改查，向 UI 提供以下能力：

- 获取全部影片列表（支持分页、筛选、排序）
- 获取单部影片详情
- 更新影片信息（收藏、评分、用户标签等）
- 删除影片记录

### 3.2 扫描服务（Scanner Service）

#### 扫描流程

```
扫描配置的影片目录
        │
        ▼
  识别视频文件
  （mp4/mkv/avi/mov/ts）
        │
        ▼
    提取番号
        │
        ▼
  查询数据库是否已存在
        │
   ┌────┴────┐
  存在      不存在
   │          │
   ▼          ▼
更新路径    创建新影片记录
             │
             ▼
         搜刮元数据
             │
             ▼
      下载封面 / 预览图
             │
             ▼
          写入数据库
```

#### 支持的视频格式

```
mp4、mkv、avi、mov
```

#### 番号识别规则

番号是影片入库和元数据搜刮的核心标识，需要从文件名中统一解析。

**识别规则**

以下文件名均应解析为 `ABC-123`：

```
ABC-123.mp4
abc123.mp4
ABC_123.mp4
ABC123.mp4
```

**正则匹配规则**

```
([A-Za-z]{2,5})[-_ ]?(\d{2,5})
```

提取后统一转换为大写并加连字符格式：`ABC-123`

**解析流程**

```
原始文件名
    │
    ▼
正则提取字母部分 + 数字部分
    │
    ▼
统一格式化为 ABC-123
    │
    ▼
传入 metatube-sdk-go 查询
```

> 若番号无法识别，则跳过该文件，记录到日志中待人工处理。

### 3.3 元数据搜刮（Metadata Scraper）

核心依赖 **metatube-sdk-go**，该 SDK 提供了几乎全部 AV 元数据、封面、预览图的搜刮能力。

#### 搜刮流程

```
传入番号
    │
    ▼
调用 metatube-sdk-go
    │
    ▼
获取 metadata（标题、女优、时长、标签等）
    │
    ▼
下载 poster（封面）
下载 thumb（预览图）
    │
    ▼
写入数据库 + 保存图片到本地缓存
```

#### 未来扩展：搜刮器抽象接口

为支持未来接入多数据源（如 JAVBus、JAVDB、Fanza），搜刮模块应抽象为接口：

```
scrapers/
   ├─ metatube/
   ├─ javbus/
   └─ javdb/
```

```go
type Scraper interface {
    Search(num string) (*Metadata, error)
    DownloadCover(url string) ([]byte, error)
}
```

### 3.4 播放器（Player Controller）

由 Go 后端负责管理 mpv 进程，暴露控制接口给 Electron 主进程调用。

提供的控制方法：

```
Play(file string)
Pause()
Seek(seconds int)
SetVolume(v int)
ToggleFullscreen()
Stop()
```

---

## 4. 数据结构

### Movies 表

```sql
movies
----------------------------
id              INTEGER PRIMARY KEY
title           TEXT
num             TEXT        -- 番号，如 ABC-123
path            TEXT        -- 影片文件路径
runtime         INTEGER     -- 时长（秒）
rating          REAL        -- 评分
favorite        BOOLEAN
first_add_time  DATETIME
```

### Actors 表

```sql
actors
----------------------------
id              INTEGER PRIMARY KEY
name            TEXT
avatar          TEXT        -- 本地缓存头像路径
```

### MovieActors 关联表（多对多）

```sql
movie_actors
----------------------------
movie_id        INTEGER
actor_id        INTEGER
```

### Tags 表

```sql
tags
----------------------------
id              INTEGER PRIMARY KEY
name            TEXT
type            TEXT        -- 'nfo'（来自NFO解析） 或 'user'（用户自定义）
```

### MovieTags 关联表（多对多）

```sql
movie_tags
----------------------------
movie_id        INTEGER
tag_id          INTEGER
```

---

## 5. 文件存储结构

### 影片目录结构

以番号 `ABC-123` 的影片为例：

```
ABC-123/
 ├─ ABC-123.mp4       -- 影片视频
 ├─ ABC-123.nfo       -- 元数据文件（NFO格式）
 ├─ poster.jpg        -- 封面
 ├─ thumb.jpg         -- 预览图主图
 └─ preview/          -- 预览截图目录
     ├─ 01.jpg
     ├─ 02.jpg
     └─ ...
```

> 封面和预览图文件名不再包含番号，保持目录结构清晰。

### 应用缓存目录

```
/Library/cache/
 └─ poster/           -- 封面缩略图缓存
     ├─ ABC-123_small.jpg
     └─ ABC-123_large.jpg
```

---

## 6. 播放器设计

### 技术方案

采用 **mpv + FFmpeg** 实现播放，mpv 直接渲染到 Electron 窗口，HTML 层叠加控制界面。

### 视频渲染方式

让 mpv 渲染到 Electron 原生窗口句柄：

```bash
mpv --wid=<window_id> --input-ipc-server=\\.\pipe\mpv-pipe movie.mkv
```

Electron 获取窗口句柄：

```js
win.getNativeWindowHandle()
```

窗口结构：

```
Electron Window
│
├─ mpv 视频层（原生渲染）
└─ HTML overlay（控制界面）
```

### UI 播放器层次结构

```
Player
 ├─ Video Surface     -- mpv 视频画面
 ├─ Subtitle Layer    -- 字幕层
 ├─ Control Overlay   -- 播放控制栏
 └─ Gesture Layer     -- 手势/鼠标事件捕获
```

### UI 控制 mpv 的流程

mpv 提供 **JSON IPC 接口**，Windows 下通过命名管道通信：

```
\\.\pipe\mpv-pipe
```

常用控制命令：

```json
// 播放 / 暂停
{ "command": ["cycle", "pause"] }

// 设置暂停状态
{ "command": ["set_property", "pause", true] }

// 快进 10 秒
{ "command": ["seek", 10] }

// 设置音量
{ "command": ["set_property", "volume", 80] }
```

完整控制链路：

```
UI 按钮点击
     │
     ▼
Vue Renderer
     │ Electron IPC
     ▼
Electron Main / Go Backend
     │
     ▼
mpv JSON IPC（命名管道）
```

调用示例：

```js
window.electronAPI.playPause()
```

### mpv 事件监听

mpv 会主动推送以下事件，Go 后端需监听并转发给 UI：


| 事件       | 用途                           |
| ------------ | -------------------------------- |
| `time-pos` | 当前播放进度（用于进度条更新） |
| `duration` | 影片总时长                     |
| `pause`    | 暂停状态变化                   |
| `end-file` | 播放结束                       |

---

## 7. UI 页面设计

### 侧边栏（Sider）

- 全部影片
- 喜爱的影片
- 最近添加
- 标签

### 影片库页面（MoviesLibraryPage）

- 影片卡片：展示封面（poster）、影片标题、女优信息
- 支持筛选（按标签、女优、评分等）
- 支持排序（按添加时间、评分、时长等）
- 瀑布流布局，使用虚拟滚动（见性能优化章节）

### 影片详情页（MoviesDetailPage）

- 展示标题、女优、封面、预览图
- 展示来自 NFO 的标签
- 支持用户自定义添加/删除标签
- 收藏和评分功能

### 播放器页面（PlayerPage）

- 基本播放控制（播放、暂停、快进、快退、音量）
- 进度条
- 全屏切换

### 设置页面（SettingsPage）

- 添加 / 删除影片存储目录（支持多目录）
- 设置自动扫描时间间隔
- 手动触发扫描按钮
- 硬件解码开关

---

## 8. 设置系统

应用配置持久化到本地 `config.json` 文件：

```json
{
  "library_paths": [
    "D:/Movies",
    "E:/AV"
  ],
  "scan_interval": 3600,
  "player": {
    "hardware_decode": true
  }
}
```


| 字段                     | 说明                       |
| -------------------------- | ---------------------------- |
| `library_paths`          | 影片存储目录列表，支持多个 |
| `scan_interval`          | 自动扫描间隔，单位秒       |
| `player.hardware_decode` | 是否启用 mpv 硬件解码      |

---

## 9. 性能优化

### 封面缩略图

影片库页面不直接读取原始封面文件，而是使用预生成的缩略图：

```
poster_small.jpg   -- 用于影片库瀑布流卡片
poster_large.jpg   -- 用于影片详情页
```

缩略图统一缓存至：

```
/Library/cache/poster/
```

### 虚拟滚动

影片库页面使用虚拟滚动（Virtual Scroll）技术，避免大量 DOM 节点导致卡顿。

推荐使用：

```
vue-virtual-scroller
```

在影片数量达到数百乃至数千部时，保持流畅的滚动体验。

---

## 10. 日志系统

Go 后端使用 **zap** 记录运行日志，日志文件路径：

```
logs/
 └─ app.log
```

需要覆盖以下事件的日志记录：


| 模块       | 记录内容                                |
| ------------ | ----------------------------------------- |
| 扫描服务   | 扫描开始/结束、识别到的文件、跳过的文件 |
| 元数据搜刮 | 搜刮成功/失败、番号、来源               |
| 播放器     | 启动、停止、错误                        |
| 错误       | 所有异常堆栈                            |
