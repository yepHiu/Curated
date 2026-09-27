# GitHub Latest 与组件更新渠道解耦

## 问题与已核实影响

2026-09-27，Full 1.7.3 首次 CD 因 Latest 指向 Full 1.7.2 而在 `legacy_latest()` 失败。临时将 Latest 指回 v1.5.8 后，第 2 次运行完成全部质量、Windows/Mac 构建和升级验收，并公开 Full 1.7.3。发布配置将公共下载入口绑在旧一体版上，是这次阻塞的根因。

旧 v1.5.8 的更新器请求写死的 `/releases/latest`，只接受 `vX.Y.Z`，并按首个含 setup/installer 的 EXE 选安装包。面对 `full-v1.7.3`，版本解析返回错误，`HasUpdate=false`，正常下载入口被禁止。不能通过修改仓库源码远程修补这些已安装二进制；也不能通过给 Full 伪装普通 v 标签绕开检查，否则多 EXE 资产可能选中 Desktop。其他历史版本不作未经验证的兼容性承诺。

## 实施规则

1. GitHub Latest 选择最高数字 SemVer 的公开正式 Full，不依赖发布时间、当前 Latest 或单组件版本。没有 Full 时才选择最新 legacy v 版本；忽略草稿、预发布及单独 Server/Desktop。目标缺少正确非空 Full 安装器时拒绝推广，不静默回退。
2. Server/Desktop 保持独立 `release-channels/server.json`、`desktop.json` 与组件/平台/摘要校验。Latest 只负责用户下载导航。
3. 去掉组件 check、publish、channels 对 legacy Latest 的强制要求。上传和摘要校验成功后公开，再更新组件渠道并协调 Latest。协调失败允许恢复，不把已公开发行版退回草稿，也不覆盖产物。
4. legacy 发布同样不能抢占已存在 Full 的 Latest。恢复渠道后重试协调。新增手动 `Release - Reconcile Latest` 工作流，与所有发布任务共用并发锁；只修改 Latest 标记。
5. 文档和当前 Full 发行说明显式说明：旧一体版用户从 Latest 下载 Full，手动运行一次完成迁移，之后独立组件更新不依赖 Latest。不承诺旧客户端的应用内更新检查能被远端修复。

## 验证与落地

回归覆盖：当前 Latest 是任意版本也不阻止发布；Full 优先与数值版本排序；草稿/预发布排除；单组件和旧版本补发不抢占；分页；缺失安装器拒绝；重复协调幂等；协调失败不隐藏公开发行；发布和恢复路径调用一致策略。

发布脚本修改无需重新构建 Full 1.7.3，也不移动其标签。推送维护修复后，使用新工作流将已通过 CD 的 Full 1.7.3 设为 Latest，并核对公开资产、组件渠道与最终 API 状态。历史标签中的旧脚本保持不变；后续恢复使用主分支维护工具，不重跑旧标签发布逻辑来恢复新策略。
