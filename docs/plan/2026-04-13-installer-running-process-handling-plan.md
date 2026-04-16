# 安装器处理运行中 Curated 进程方案

## 背景

当前 Windows 安装器由 Inno Setup 模板 `scripts/release/windows/Curated.iss.tpl` 生成。

现状：

- 安装器模板只定义了基础 `[Setup]`、`[Files]`、`[Icons]`、`[Run]`
- 没有启用 Inno Setup 的关闭运行中程序能力
- Curated release 在 Windows 默认以托盘模式运行，主进程为 `curated.exe`
- 程序已有单实例互斥名：`Local\Curated.Tray.Singleton`

结果：

- 当旧版本 `curated.exe` 正在运行时，安装器不会主动关闭它
- 升级安装时容易出现文件占用、无法覆盖、或需要用户手动退出程序

## 根因

问题不是打包脚本没找到进程，而是安装器模板本身没有实现“检测并关闭运行中 Curated”的逻辑。

当前缺少的能力主要有两类：

1. 安装阶段识别 Curated 正在运行
2. 安装前请求或强制退出 Curated，再继续复制文件

## 可选方案

### 方案 A：启用 Inno Setup 的关闭运行中应用能力

做法：

- 在 `[Setup]` 中启用 `CloseApplications=yes`
- 配置 `AppMutex=Local\Curated.Tray.Singleton`
- 结合 `CloseApplicationsFilter=curated.exe`
- 保持 `RestartApplications=no`

优点：

- 改动最小
- 保持安装逻辑集中在 Inno Setup
- 对当前单实例托盘模式最匹配
- 后续维护成本最低

风险：

- 依赖 Inno Setup 对运行中进程/互斥体的识别
- 如果未来运行方式变化，需要同步更新互斥名或过滤条件

### 方案 B：增加安装前自定义脚本，先优雅退出，再回退到强制结束

做法：

- 在 Inno `[Code]` 中加入自定义逻辑
- 安装开始前检测 `curated.exe`
- 先尝试按互斥/窗口/进程名触发优雅退出
- 超时后再调用强制终止

优点：

- 可控性最高
- 可以区分“请求退出”和“强杀”
- 能加入更明确的中文提示

风险：

- 实现复杂度更高
- 需要自己维护超时、失败分支和用户提示
- 若直接强杀，可能打断正在进行的后台任务

### 方案 C：应用内增加显式“优雅退出”入口，再让安装器调用它

做法：

- 给 `curated.exe` 增加命令行参数或本地控制接口，例如 `--shutdown`
- 安装器升级前调用这个入口
- 应用收到后自行收尾并退出

优点：

- 用户态退出路径最干净
- 适合未来做自动升级器
- 后续可复用到重启、更新、诊断工具

风险：

- 不是纯安装器改动
- 需要同时改 Go 后端和安装器
- 交付周期最长

## 推荐

推荐先做 **方案 A**，必要时为方案 B 预留扩展点。

原因：

- 当前根因就在安装器模板层，先在 Inno Setup 把“识别并关闭运行中 Curated”补齐，收益最大
- 现有程序已经有稳定的单实例互斥名 `Local\Curated.Tray.Singleton`，可以直接作为安装器识别信号
- 不需要先改应用行为，就能解决大部分升级安装失败场景

建议的首轮实现范围：

- Inno Setup 模板增加 `CloseApplications=yes`
- Inno Setup 模板增加 `AppMutex=Local\Curated.Tray.Singleton`
- 视兼容性再增加 `CloseApplicationsFilter=curated.exe`
- 安装文案增加“将关闭正在运行的 Curated 后继续安装”
- 重新打一个 installer 包验证升级流程

## 验证标准

- Curated 正在托盘运行时，启动安装器不再要求用户手动先退出
- 安装器能成功关闭旧进程并覆盖安装目录
- 安装完成后可正常启动新版本
- 升级后不会出现两个 Curated 实例并存

## 后续扩展

如果方案 A 在部分机器上仍不稳定，再进入方案 B：

- 加自定义安装脚本
- 增加更明确的超时和错误提示
- 必要时再补应用内 `--shutdown` 能力

## 实施步骤

1. 修改 `scripts/release/windows/Curated.iss.tpl`
   - 在 `[Setup]` 中加入 `CloseApplications=yes`
   - 在 `[Setup]` 中加入 `RestartApplications=no`
   - 在 `[Setup]` 中加入 `AppMutex=Local\Curated.Tray.Singleton`
   - 在 `[Setup]` 中加入 `CloseApplicationsFilter=curated.exe`

2. 重新生成安装器脚本与安装包
   - 执行 `pnpm release:installer`
   - 确认生成的 `release/installer/Curated.iss` 包含上述配置
   - 确认新的 `Curated-Setup-<version>.exe` 产物生成成功

3. 升级流程验证
   - 手动启动一个正在运行的 `curated.exe`
   - 运行新安装器
   - 确认安装器会处理运行中的 Curated，而不是直接卡在文件占用

## 2026-04-16 补充：为什么现在还是只会提示手动关闭

经过复核当前模板 `scripts/release/windows/Curated.iss.tpl`，现状已经是：

- `CloseApplications=yes`
- `RestartApplications=no`
- `AppMutex=Local\Curated.Tray.Singleton`
- `CloseApplicationsFilter=curated.exe`

但用户实际看到的行为仍然是：

- 安装器先提示“程序正在运行，请先关闭后再继续”
- 没有进入“由安装器自动关闭应用”的确认交互

根因不是 `CloseApplications` 失效，而是 **`AppMutex` 抢先触发了安装器启动阶段的阻塞检查**。

根据 Inno Setup 官方文档：

- `AppMutex` 会在安装器启动时直接弹出：
  - “检测到应用正在运行，请先关闭，再点 OK 继续 / Cancel 退出”
- `CloseApplications=yes` 的自动关闭交互则发生在：
  - `Preparing to Install` 页面
  - 安装器开始实际复制文件之前

这两个机制叠加后，`AppMutex` 会更早拦住流程，导致用户根本走不到 `CloseApplications` 那一页。

## 2026-04-16 调整建议

为实现“检测到 Curated 正在运行时，给用户一个确认选项，然后由安装器自己关闭程序并继续安装”，推荐把方案 A 收敛为：

1. **移除安装器模板里的 `AppMutex`**
   - 不再使用安装器启动时的“手动先关闭程序”阻塞提示
2. **保留 `CloseApplications=yes`**
   - 让 Inno Setup 在安装前阶段使用 Restart Manager 检测并关闭占用文件的进程
3. **保留 `CloseApplicationsFilter=curated.exe`**
   - 将自动关闭范围限定为 Curated 主程序
4. **保留 `RestartApplications=no`**
   - 避免安装完成后自动重启旧实例

这样用户预期会变成：

- 启动安装器
- 安装器检测到正在运行的 `curated.exe`
- 在安装前页面列出它，并询问是否由安装器自动关闭
- 用户确认后，安装器自行关闭程序并继续安装

## 2026-04-16 风险与边界

- 该方案依赖 Inno Setup 的 `CloseApplications` + Windows Restart Manager。
- 如果个别机器上 Restart Manager 无法关闭目标进程，再考虑补 `[Code]` 自定义退出逻辑或强制终止逻辑。
- 应用自己的单实例互斥逻辑不需要删除；本次只调整 **安装器脚本中的 `AppMutex` 检查策略**。
