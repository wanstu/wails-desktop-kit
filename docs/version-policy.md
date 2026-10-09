# Kit 单一版本与发布规范

Kit 是单个 Go Module、单个发布主线，不允许“每个 Agent 发布自己的 Kit”。本文目标版本为 `v0.11.3`；发布成功前，以 GitHub 最新正式 Release 与 tag 为准。

## 唯一事实来源

1. **已发布**：GitHub 上的 immutable 正式 tag、GitHub Release、对应的 `go.mod` / workflow / 源码。
2. **已合并待发布**：远程 `origin/master` 的提交；不等同于已发布。
3. **开发中**：单独分支 / ADM 管理的 worktree；不得让消费者默认引用此分支。
4. **已使用**：消费者 `go.mod` 和 workflow 中的明确版本，以及实际源码调用；不要根据 Kit 当前版本猜测消费者版本。

本地 `master` 可能滞后于 `origin/master`，必须先核对二者祖先和工作区未提交修改；不允许以陈旧本地分支强推远程主线。

## 版本及合并要求

- 常规功能通过 ADM 管理 worktree 上的功能分支完成，禁止 Agent 擅自直接使用 `git worktree`。
- 功能必须包含代码、测试、中文/英文必要文档以及版本首次引入信息；保持 Go API 向后兼容，破坏性改变须单列迁移指南。
- 同一发布只允许 **一个正式 tag**（RC 是验收标签，不是消费者稳定目标）；禁止每个 Agent 自行提升 `master` 和打 tag。
- 发布前基于最新 `origin/master` 合并功能分支，检查冲突，跑 `go test ./...`、`go vet ./...`、CI 的跨平台构建 / 安装验证。
- 发正式版本时，`README`、`docs/status.md`、`docs/capabilities.md`、`docs/build-release.md`、reusable workflow 中默认 helper 的版本必须一致。
- GitHub Release 成功以及 CI 核对后才能对消费者宣告“可用”；仅 push tag 不等于验证 Release 产物。
- 发布后消费者 **显式升级** `go.mod` 中的 Kit、`uses: ...@tag`、`desktopkit-cli-version`；禁止长期引用本地 replace、worktree 或分支浮动引用。
- 变更时保留业务工作树未提交代码，不擅自覆盖/清理；拒绝以 force push 或重新指向旧 tag 解决版本冲突。

## 一次发布的核对顺序

```text
检查工作树与远程 refs
  → 选择当前正式 tag 为基线
  → 合并功能与兼容修复
  → 更新代码/测试/文档/默认 workflow helper
  → go test / go vet / 差异审查
  → 远程 master 合入并确认 CI
  → 创建唯一正式 tag 和 GitHub Release
  → 核对 Release/CI/产物
  → 通知消费者并逐个升级
```

## 已知版本混乱的历史修复

2026-10-09 核对到本地 `master=4390bb2`（v0.10.3）而远程 `origin/master=62236c0`（v0.11.0）。远程 v0.11.0 已合并 Linux systemd 打包与中英文 NSIS / Updater，不允许从旧 master 倒退发布。v0.11.1 补入 `servicecontrol` 并让默认 workflow helper 与新发布一致。历史 v0.10.x tag 保留供现有消费者锁定，不重写 tag。2026-10-09 复核发现 v0.11.1 遗漏了 v0.10.2/0.10.3 维护线提交，v0.11.2 已将两次维护变更逐项移植并验证，并通过合并维护线提交关闭 Git 祖先关系分叉；额外修复 Debian data.tar.gz 父目录条目。禁止通过改写已发布 v0.11.1 的 tag 解决此问题。
