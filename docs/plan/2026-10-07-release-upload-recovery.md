# release-20261007-2 上传恢复

## 原因与边界

Actions run `37505997682` 的 attempt 1 在上传前失败；Windows 与 macOS 构建及全部质量门禁成功。attempt 2 仍使用标签提交 `c5f40bf7aa2d357c67fc82b5eb497afaa92599ff`，不会读取后来合入 master 的修复 `e4dcd138`，因此 Re-run 会重复原错误。

修复统一新引擎锁文件记录的 LF 哈希，兼容旧 CRLF 哈希；源码资产自身校验和、内嵌锁内容、组件来源提交与冻结批次验证保持生效。不移动原标签，不重新分配版本，不替换已有安装包。

## 恢复步骤

在包含 `e4dcd138` 的干净工具工作区执行以下命令。原标签单独检出，仅作为 `--source-root`；运行的 Python 脚本必须来自修复工作区。

```powershell
git worktree add --detach .workspace/release-recovery/37505997682/source c5f40bf7aa2d357c67fc82b5eb497afaa92599ff
gh run download 37505997682 --repo yepHiu/Curated --name windows-release-release-20261007-2-1 --dir .workspace/release-recovery/37505997682/windows
gh run download 37505997682 --repo yepHiu/Curated --name macos-desktop-release-20261007-2-1 --dir .workspace/release-recovery/37505997682/macos
python scripts/release/component_cd.py stage --tag release-20261007-2 --source-root .workspace/release-recovery/37505997682/source --windows .workspace/release-recovery/37505997682/windows --macos .workspace/release-recovery/37505997682/macos --output .workspace/release-recovery/37505997682/staged
```

已存在的 source checkout、下载目录和暂存资产应复用；`stage` 拒绝覆盖已有 assets 目录。若需再次校验，用 `verify_distribution`，不要删除原始资产重新打包。

准备上传时设置仓库与凭据；凭据仅存于当前进程环境，不写入文件或日志：

```powershell
$env:GITHUB_REPOSITORY = 'yepHiu/Curated'
$env:GH_TOKEN = gh auth token
python scripts/release/component_cd.py publish --tag release-20261007-2 --source-root .workspace/release-recovery/37505997682/source --output .workspace/release-recovery/37505997682/staged --mode draft
```

`draft` 上传并校验全部资产，但不推进更新通道。正式公开时通过同一工具的 `--mode publish` 完成公开、通道推进与 Latest 校验；只在网页上点 Publish 不会执行通道推进。上传恢复不需要 git push，也不需要改旧标签。远程旧 Actions 记录会继续显示失败，不代表恢复后的资产验证失败。

## 验证

修复发布脚本测试共 107 项通过，覆盖 CRLF 生成/LF 发布、真实锁内容变化拒绝、来源提交校验和源码资产独立发布。实际资产恢复结果以本次操作记录为准。
