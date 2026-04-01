# 中国大陆网络环境下的刮削稳定性优化实施记录

## 本轮已落地

- 同源演员头像链路
  - 新增演员头像本地缓存字段与迁移：`avatar_local_path`、`avatar_last_http_status`、`avatar_last_error`、`avatar_last_fetched_at`
  - 演员资料刮削成功后，后端异步下载头像到缓存目录
  - 新增同源接口：`GET /api/library/actors/{name}/asset/avatar`
  - 演员资料和演员列表 DTO 增加本地头像状态字段，后端优先把 `avatarUrl` 改写为同源地址
- 预览图后端兜底
  - 预览图本地存在时继续优先走 `/api/library/movies/{movieId}/asset/preview/{index}`
  - 本地缺失但存在 `source_url` 时，后端会带 `Referer` 兜底拉取并转发
- 资源元数据增强
  - `media_assets` 新增 `source_provider`、`referer_url`、`last_http_status`、`last_error`、`last_fetched_at`
  - 刮削写库时把 provider / referer 一并保存，便于后续诊断热链、反爬与重试
- Provider 调度与健康缓存基础版
  - 新增设置字段：`metadataMovieStrategy`
  - 当前支持：`auto-global`、`auto-cn-friendly`、`custom-chain`、`specified`
  - `auto-cn-friendly` 会优先使用较适合大陆环境的一组 provider 顺序
  - Metatube 运行时维护 provider 健康状态：连续失败次数、平均延迟、冷却截止时间、错误分类
- 错误分类补强
  - 任务 DTO 新增：`errorCategory`、`provider`
  - Provider 健康 DTO 新增：`errorCategory`、`cooldownUntil`、`consecutiveFailures`、`avgLatencyMs`
  - 当前已覆盖的机器可读分类：
    - `dns_failure`
    - `connect_timeout`
    - `tls_failure`
    - `region_restricted`
    - `hotlink_denied`
    - `provider_empty_result`
    - `provider_invalid_content`
    - `parser_failed`
- 资源真实性检查
  - 头像/图片下载要求 `Content-Type` 为图片
  - 超小响应会被拒绝写入，避免把验证页、空壳页、脏数据当成正常图片缓存

## 本轮验证

- 前端类型检查通过：`pnpm typecheck`
- 后端核心包测试通过：
  - `go test ./internal/assets ./internal/storage ./internal/server ./internal/app ./internal/scraper/metatube`

## 本轮未完成

- 设置页还没有完整展示新的 provider 策略文案、冷却状态和大陆友好说明
- 任务详情页还没有把 providerAttempts / skippedProviders / assetFailures 系统化展示出来
- 预览图兜底当前是后端转发，不会自动回填落盘
- 反爬真实性判断目前先做了通用网络/图片层拦截，尚未对 FANZA / JavBus / FC2 / Gfriends 写完整站点级结构探针

## 下一步建议

- 在设置页加入“大陆友好模式”显式开关与推荐说明
- 增加 provider 诊断 API，把搜索可达、详情可达、图片可达拆开
- 为关键 provider 增加页面关键词/DOM 探针，识别登录页、验证页、区域限制页
- 把预览图兜底成功后的资源回填到本地缓存，减少重复跨境请求
