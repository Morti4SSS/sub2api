# MortiSSS sub2api Customizations

本文档记录本仓库相对官方 `Wei-Shaw/sub2api` 的私有改动边界。目标是以后跟进官方更新时，不需要重新回忆历史对话，也不会把账号数据、密钥或部署数据误传到 GitHub。

## 当前维护模型

- `upstream`：官方仓库 `Wei-Shaw/sub2api`，只用于拉取官方更新。
- `morti`：自定义仓库 `Morti4SSS/sub2api`，保存可公开的代码改动、计划文档和部署 tag。
- 部署分支使用 `vps/*` 或 `vps/custom-*` 命名，不直接推 `main` 或 `master`。
- 每次部署镜像都创建 `vps-deploy/*` tag，tag 指向通过验证的源码 commit。
- VPS 上的 `.env`、`config.yaml`、数据库 dump、账号 token、cookie、密钥不进入 GitHub。

## 自定义能力台账

| 能力 | 目标 | 主要 owner | 典型验证 |
| --- | --- | --- | --- |
| 测试词自定义 | OpenAI/Codex 与 Claude/Anthropic 账号测试时使用前端输入的 prompt，空值才回退默认测试词 | 前端测试弹窗、`backend/internal/service/account_test_service.go` | AccountTestModal Vitest、account test Go 单测 |
| Codex / Claude Code 客户端兼容 | 保留允许客户端的真实 `User-Agent` / `Originator`，隔离会话 ID | OpenAI / OAuth / WS passthrough 服务 | `Test.*Codex.*`、`Test.*ClaudeCode.*`、`Test.*Passthrough.*` |
| 账号 AT/RT 状态 | 管理端账号列表紧凑显示 OpenAI/Codex 与 Claude 账号 access / refresh token 状态，不显示原文 | account DTO、账号列表前端列 | DTO redaction 单测、AccountTokenStatusCell Vitest |
| 上游错误摘要 | OpenAI/Codex 与 Claude failover 耗尽时返回脱敏的上游错误摘要，避免只看到 sub2api 泛化错误 | `UpstreamFailoverError`、错误摘要 helper、gateway handlers | upstream_error_summary Go 单测、handler 单测 |
| 轮换可读标识 | 保留官方已有账号池、调度、临时不可调度等状态，不因自定义改动覆盖 | account DTO、admin account list | 账号列表人工检查、已有 handler/dto 单测 |

## Claude Code 模型行为

- 不维护私有 Claude Code 模型目录、官方模型外壳或账号级思考强度映射。
- Claude Code 通过本地配置直接发送中转站支持的真实模型 ID；Sub2API 不尝试改变 Claude Code `/model` 的显示内容。
- 官方已有的通用模型映射、协议转换和 OpenAI reasoning 策略保持原样，不属于本仓库私有契约。

## 跟进官方更新流程

1. 确认当前工作区干净，未提交改动先保存到自定义分支。
2. 拉取官方更新：`git fetch upstream --tags`。
3. 从当前部署基线开同步分支，例如 `vps/sync-upstream-YYYYMMDD`。
4. 合并官方 tag 或官方分支到同步分支，先解决官方代码冲突。
5. 对照本文件“自定义能力台账”和当前实现 diff，确认每个自定义能力仍在。
6. 跑最小验证：

```powershell
cd backend
go test ./internal/service ./internal/handler ./internal/handler/dto -count=1

cd ..\frontend
pnpm vitest run src/components/account/__tests__/AccountTestModal.spec.ts src/components/admin/account/__tests__/AccountTestModal.spec.ts src/components/account/__tests__/AccountTokenStatusCell.spec.ts
pnpm typecheck
pnpm build

git diff --check
```

7. 验证通过后推新 `vps/*` 分支，不直接推默认分支。
8. 构建新 GHCR 镜像，部署前在 VPS 备份 `.env`、Compose 文件、`config.yaml`、PostgreSQL dump 和当前镜像信息。
9. 部署后在 VPS 文档记录镜像 tag、commit、备份目录、验证结果和回滚命令。

## 合并冲突优先级

- 账号数据安全优先：任何 token、cookie、密钥、数据库 dump 都不能进入 Git。
- 官方 bugfix 优先保留，私有逻辑尽量重接到官方新结构上。
- 自定义配置 key 不随意改名；确需改名必须同时更新前端、后端、测试、文档和迁移/兼容说明。
- 如果官方已经实现同类能力，优先使用官方实现，只补缺失的 Codex / Claude Code 私有需求。
- 高冲突文件先加回归测试，再重接实现，避免靠人工判断。

## 回滚原则

- 代码回滚：优先切回上一个 `vps-deploy/*` tag 对应镜像。
- 镜像回滚：修改 VPS `SUB2API_IMAGE` 回上一版并重建 app 容器。
- 数据库回滚：只能在明确确认后恢复 dump，因为会覆盖备份时间点之后的新账号和新配置。
