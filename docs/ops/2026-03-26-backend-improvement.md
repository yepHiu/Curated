# 后端代码优化建议与质量问题分析

## 一、严重质量问题 🔴

### 1. SQLite 单连接限制
**位置**: `internal/storage/sqlite.go:32`

```go
db.SetMaxOpenConns(1)  // 严重限制并发性能
```

**问题**:
- 强制单连接导致所有数据库操作串行化
- 高并发场景下请求排队，延迟飙升
- 读写无法分离

**建议**:
```go
// 使用连接池，允许适量并发
db.SetMaxOpenConns(10)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(time.Hour)
```

### 2. 任务仅内存存储，重启丢失
**位置**: `internal/tasks/manager.go`

**问题**:
- 所有任务状态保存在内存 map 中
- 服务重启后任务历史完全丢失
- 无法实现分布式或高可用部署

**建议**:
- 将任务状态持久化到数据库（tasks 表已存在，但似乎只用于保存，不用于恢复）
- 启动时从数据库恢复未完成任务
- 或者使用嵌入式任务队列（如 SQLite 队列表）

### 3. 竞态条件风险
**位置**: `internal/app/app.go:enqueueScrape`

```go
func (a *App) enqueueScrape(...) {
    go func() {
        a.scrapeSem <- struct{}{}  // 可能阻塞很久
        defer func() { <-a.scrapeSem }()
        a.runScrape(...)
    }()
}
```

**问题**:
- 启动大量 goroutine 等待 semaphore，内存泄漏风险
- 没有优雅关闭机制，可能丢失正在执行的任务

**建议**:
```go
func (a *App) enqueueScrape(ctx context.Context, ...) error {
    select {
    case a.scrapeSem <- struct{}{}:
        go func() {
            defer func() { <-a.scrapeSem }()
            a.runScrape(ctx, ...)
        }()
    case <-ctx.Done():
        return ctx.Err()
    default:
        return errors.New("scrape queue full")
    }
}
```

---

## 二、性能问题 🟡

### 1. 目录扫描无并发
**位置**: `internal/scanner/service.go:119-155`

**问题**:
- 使用 `filepath.Walk` 单线程遍历
- 大目录（数万文件）扫描缓慢
- 没有利用 IO 并行性

**建议**:
```go
// 使用并发 walker
func (s *Service) findVideoFilesConcurrent(paths []string) ([]string, error) {
    var wg sync.WaitGroup
    fileChan := make(chan string, 100)
    errChan := make(chan error, 1)

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    for _, root := range paths {
        wg.Add(1)
        go func(r string) {
            defer wg.Done()
            // 并发遍历每个根目录
        }(root)
    }

    // 收集结果...
}
```

### 2. 数据库查询 N+1 问题
**位置**: `internal/storage/library_repository.go`

**问题**:
- `ListMovies` 查询主表后，演员、标签需要额外查询
- 每页 50 条可能产生 1 + 50 + 50 = 101 次查询

**当前代码模式**:
```go
// 先查 movies
rows, _ := s.db.QueryContext(ctx, movieQuery)
for rows.Next() {
    // 每部电影再查演员、标签
    actors := s.getMovieActors(movieID)
    tags := s.getMovieTags(movieID)
}
```

**建议**:
```go
// 使用 JOIN 一次性查询
SELECT m.*,
       GROUP_CONCAT(DISTINCT a.name) as actors,
       GROUP_CONCAT(DISTINCT t.name) as tags
FROM movies m
LEFT JOIN movie_actors ma ON m.id = ma.movie_id
LEFT JOIN actors a ON ma.actor_id = a.id
LEFT JOIN movie_tags mt ON m.id = mt.movie_id
LEFT JOIN tags t ON mt.tag_id = t.id
GROUP BY m.id
```

### 3. 刮削并发度固定，无优先级
**位置**: `internal/app/app.go:scrapeSem`

**问题**:
- 使用固定大小的 channel 做信号量
- 没有区分用户触发的刮削和后台自动刮削的优先级
- FC2 和普通番号混用同一队列

**建议**:
- 实现优先级队列（用户手动触发 > 新入库 > 批量刷新）
- 按 provider 分组限制并发（避免单个 provider 被封）

### 4. 大文件视频流无优化
**位置**: `internal/server/server.go:349-391`

**问题**:
- 使用标准 `http.ServeContent`，虽支持 Range 但无缓冲优化
- 没有 CDN/缓存层支持
- 高码率视频可能产生大量小 IO

**建议**:
- 添加缓冲读取器
- 实现自适应码率（如果将来支持转码）
- 添加 ETag 和 Last-Modified 优化缓存

---

## 三、代码质量问题 🟠

### 1. 错误处理不一致
**位置**: 多处

**问题示例**:
```go
// 方式1: 直接忽略
_ = a.emitEvent(output, contracts.EventTaskCompleted, ...)

// 方式2: 只记录日志
if err := a.emitEvent(...); err != nil {
    a.logger.Error("...", zap.Error(err))
}

// 方式3: 返回错误
if err := a.writeResponse(...); err != nil {
    return err
}
```

**建议**: 统一错误处理策略，关键路径（如任务完成事件）不应静默失败

### 2. 函数过长，职责过多
**位置**: `internal/app/app.go`

| 函数 | 行数 | 问题 |
|------|------|------|
| `handleCommand` | 111 | 处理所有 stdio 命令，switch 语句过长 |
| `runScan` | 208 | 包含扫描、入库、事件发射、刮削触发 |
| `runMovieScrapeBody` | 79 | 包含刮削、保存、NFO、事件、资源下载 |

**建议**:
- 按命令类型拆分为独立方法
- 提取小函数：saveMetadata, emitProgress, enqueueAssets 等

### 3. 魔法字符串
**位置**: 多处

```go
// 硬编码的状态值
result.Status = "skipped"
result.Status = "imported"
result.Status = "updated"
taskType = "scan.library"
taskType = "scrape.movie"
```

**建议**:
```go
const (
    StatusSkipped  = "skipped"
    StatusImported = "imported"
    StatusUpdated  = "updated"

    TaskTypeScanLibrary  = "scan.library"
    TaskTypeScrapeMovie  = "scrape.movie"
)
```

### 4. 上下文传递问题
**位置**: `internal/app/app.go:enqueueScrape`

```go
// 问题：使用 parentCtx 但 goroutine 可能跑很久
func (a *App) enqueueScrape(parentCtx context.Context, ...) {
    go func() {
        a.scrapeSem <- struct{}{}
        a.runScrape(parentCtx, ...)  // 如果 parentCtx 是请求上下文，可能很快超时
    }()
}
```

**建议**:
- 后台任务使用 `a.appCtx`（应用生命周期）
- 传递带超时的子上下文

### 5. 重复代码
**位置**: `internal/app/app.go`

多处重复:
```go
if emitErr := a.emitEvent(...); emitErr != nil {
    a.logger.Error("failed to emit...", ...)
}
```

**建议**: 提取辅助函数
```go
func (a *App) emitSafe(eventType string, payload any, fallbackLog string) {
    if err := a.emitEvent(a.output, eventType, payload); err != nil {
        a.logger.Warn(fallbackLog, zap.Error(err))
    }
}
```

---

## 四、架构设计问题 🔵

### 1. 循环依赖风险
**位置**: app.go 依赖众多包

**问题**:
- App 结构体直接依赖 scanner、scraper、assets、storage 等
- 所有功能耦合在 App 中，难以单元测试

**建议**: 使用依赖注入框架或简化依赖关系

```go
// 使用接口解耦
type MovieImporter interface {
    Import(ctx context.Context, file ScanFileResult) error
}

type MetadataFetcher interface {
    Fetch(ctx context.Context, number string) (Metadata, error)
}
```

### 2. 存储层接口缺失
**位置**: `internal/storage/*.go`

**问题**:
- 直接暴露 `*SQLiteStore` 结构体
- 调用方依赖具体实现，无法 Mock 测试

**建议**:
```go
type Store interface {
    ListMovies(ctx context.Context, req ListMoviesRequest) (MoviesPageDTO, error)
    GetMovieDetail(ctx context.Context, id string) (MovieDetailDTO, error)
    // ...
}

var _ Store = (*SQLiteStore)(nil)
```

### 3. 事件系统过于简单
**位置**: `internal/app/app.go:emitEvent`

**问题**:
- 仅支持 stdio 输出事件
- HTTP 模式下事件丢失（output=io.Discard）
- 无事件订阅/广播机制

**建议**:
- 实现 Pub/Sub 模式
- HTTP 模式使用 WebSocket 或 Server-Sent Events
- 支持多监听器

---

## 五、安全与可靠性 🛡️

### 1. SQL 注入风险（低）
**位置**: 部分动态 SQL

虽然使用 `?` 占位符，但 `buildMovieFilters` 等函数拼接字符串需谨慎审查。

**建议**: 所有动态查询使用参数化查询，避免字符串拼接

### 2. 路径遍历风险
**位置**: `internal/library/organize.go`

```go
func OrganizeVideoFile(path string, number string) (string, error) {
    // 需要验证 path 是否在允许的库目录内
}
```

**建议**: 添加路径验证，确保目标路径在配置的库路径内

### 3. 资源泄露
**位置**: `internal/assets/service.go`

**问题**:
- HTTP 客户端没有超时设置
- 下载大文件可能占用连接池过久

**建议**:
```go
client := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        ResponseHeaderTimeout: 10 * time.Second,
    },
}
```

### 4. 无限递归风险
**位置**: `internal/scanner/service.go:filepath.Walk`

**问题**:
- 如果遇到循环链接（symlink），可能无限递归
- 无最大深度限制

**建议**:
```go
const maxWalkDepth = 10

func (s *Service) findVideoFilesWithDepth(paths []string, maxDepth int) ([]string, error) {
    // 跟踪深度，防止过深遍历
}
```

---

## 六、可观测性 📊

### 1. 缺少关键指标
- 扫描速率（文件/秒）
- 刮削成功率
- 数据库查询延迟
- 队列长度

**建议**: 集成 Prometheus 指标

```go
var scanDuration = prometheus.NewHistogram(...)
var scrapeSuccess = prometheus.NewCounterVec(...)
```

### 2. 日志格式不一致
**问题**:
- 有些使用 `zap.Error`，有些使用字符串拼接
- 缺少 Trace ID 进行请求追踪

**建议**: 统一日志格式，添加请求上下文

---

## 七、具体优化建议清单

### 高优先级（立即修复）

1. **修复 SQLite 连接池限制** - 性能提升 10x+
2. **添加任务持久化** - 避免重启丢失
3. **修复 enqueueScrape goroutine 泄露** - 稳定性
4. **优化 ListMovies N+1 查询** - 响应速度提升

### 中优先级（近期修复）

5. **拆分长函数** - 代码可维护性
6. **统一错误处理** - 健壮性
7. **添加存储层接口** - 可测试性
8. **实现优先级队列** - 用户体验

### 低优先级（长期规划）

9. **并发目录扫描** - 大规模库性能
10. **事件系统重构** - WebSocket 支持
11. **Prometheus 指标** - 可观测性
12. **依赖注入框架** - 架构整洁

---

## 八、重构示例

### 原始代码（app.go:runScan 简化）
```go
func (a *App) runScan(ctx context.Context, output io.Writer, taskID string, paths []string) {
    // 200+ 行，包含：
    // - 进度回调
    // - 文件整理
    // - 入库逻辑
    // - 刮削触发
    // - 事件发射
}
```

### 重构后
```go
type ScanOrchestrator struct {
    scanner  *scanner.Service
    store    *storage.SQLiteStore
    importer MovieImporter
    notifier EventNotifier
}

func (o *ScanOrchestrator) Run(ctx context.Context, taskID string, paths []string) (*ScanResult, error) {
    files, err := o.scanner.Scan(ctx, paths)
    if err != nil {
        return nil, err
    }

    result := &ScanResult{}
    for _, file := range files {
        imported, err := o.importer.Import(ctx, file)
        if err != nil {
            result.AddError(file.Path, err)
            continue
        }
        if imported.IsNew {
            result.Imported++
        } else {
            result.Updated++
        }
    }

    o.notifier.NotifyScanCompleted(ctx, result)
    return result, nil
}
```

---

## 九、性能基准测试建议

```go
// 需要添加的 benchmark

func BenchmarkListMovies(b *testing.B) {
    // 测试 10k 影片的列表性能
}

func BenchmarkScanLargeDirectory(b *testing.B) {
    // 测试 100k 文件的扫描性能
}

func BenchmarkConcurrentScrape(b *testing.B) {
    // 测试刮削并发度
}
```

---

**总结**: 后端代码整体结构清晰，但存在连接池配置不当、任务不持久化、部分性能瓶颈等问题。建议优先修复 SQLite 连接数和任务持久化问题，再逐步优化查询性能和代码质量。
