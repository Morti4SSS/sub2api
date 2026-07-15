# Sub2API Claude Code Routing r3

本文件是 r3 的唯一实施契约。基线提交为 `5e4870c902383302e61fd971eaa147b793d9831e`，开发分支为 `vps/custom-cc-routing-r3`。

## 目标与边界

只维护 Claude Code CLI 和 Codex CLI。每个第三方中转站 API Key 作为独立账号录入；用户手工启停账号并设置优先级。Claude Code 在同一 Sub2API 渠道中切换标准外壳时不需要重启。

本轮完成：

1. 稳定的 Claude Code 模型目录和严格账号调度。
2. 账号级上游模型、上下文窗口和思考强度映射。
3. 模型同步失败诊断与手工填写兜底。
4. Claude/Codex 真实网关连接测试、自定义完整问题和 CLI 身份。
5. 可区分的路由/上游错误。
6. OAuth/Setup Token 的真实刷新审计。

本轮不做：未知模型能力探测、厂商预设、动态修改 `/model` 名称、读取本机 Claude Code 配置、账号-分组绑定级映射、失败后的跨协议重试、不同真实模型自动切换、原子账号切换系统。

## 唯一数据 owner

### Claude Code 标准外壳

- 外壳 ID 和默认显示名：`backend/internal/pkg/claude/constants.go` 中的 `claude.DefaultModels`。
- 分组实际暴露哪些外壳：现有 `groups.models_list_config`。
- `/v1/models` 只从上述稳定目录读取，不从当前启用账号拼接目录。
- 关闭账号不得让外壳消失。

### 账号级路由

唯一 owner 为 `accounts.extra.claude_code_routes`：

```json
[
  {
    "shell_model": "claude-opus-4-8",
    "upstream_model": "glm-5.2",
    "display_name": "强模型路由",
    "context_window": 200000,
    "thinking": {
      "mode": "levels",
      "target_field": "reasoning_effort",
      "levels": ["high", "max"],
      "overrides": {}
    }
  }
]
```

运行时规则：

1. 只有显式配置了请求外壳的启用账号才有资格参与调度。
2. 空路由不再等于支持全部 Claude Code 模型。
3. 同一真实模型在不同中转站可配置同一外壳并通过优先级主备。
4. 不同真实模型必须使用不同外壳，由用户在 Claude Code `/model` 主动切换。
5. 当前外壳无账号时不得自动改用其他外壳或真实模型。

Codex 继续使用 `credentials.model_mapping`，不迁移到此字段。

### 思考映射

Claude Code 源档位固定为 `low/medium/high/xhigh/max`，唯一常量 owner 与 `DefaultModels` 同在 `claude/constants.go`。

默认使用档位型配置。用户按从弱到强填写 N 个上游档位，系统按下式生成五档映射：

```text
round(sourceIndex * (N - 1) / 4)
```

- 允许通过 `overrides` 逐档覆盖。
- 客户端未传合法档位时不注入。
- 账号未配置思考策略时不猜测、不注入。
- `thinking.budget_tokens` 作为折叠高级设置，只接受正整数。
- 不按模型名推断厂商或思考协议。

### 模型列表地址

唯一 owner 为 `accounts.extra.upstream_models_url`。它保存用户填写的完整模型列表 URL；为空时沿用现有标准 URL 推导。URL 继续经过现有安全 allowlist 校验。

### Token 刷新审计

唯一 owner 为 `accounts.extra.token_refresh_status`：

```json
{
  "last_attempt_at": "RFC3339",
  "last_result": "success|failed",
  "trigger": "manual|background",
  "error": "redacted",
  "next_window_start": "RFC3339",
  "next_window_end": "RFC3339"
}
```

API Key 账号不显示 AT/RT。OAuth/Setup Token 只展示真实记录，不根据 Refresh Token 是否存在推断成功。

## 兼容、迁移与移除

旧字段仅限：

- `extra.claude_code_model_catalog`
- `extra.claude_code_effort_mapping`

兼容规则：

1. 有 `claude_code_routes` 时，运行时只读新字段。
2. 没有新字段时，编辑表单可读取旧字段并生成待保存的新结构。
3. r3 保存账号时写新字段，同时生成上述两个旧字段的狭窄兼容投影，保证回滚 r2 后配置仍可读。
4. 不做数据库 schema 迁移，继续使用现有 JSONB。
5. r3 不移除旧字段写入。只有在 r3 生产稳定、r2 回滚窗口结束、数据库与加密账号快照均验证可恢复后，才可另开任务停止兼容投影。

## 错误契约

- `route_not_found`：稳定目录不存在请求外壳。
- `no_eligible_account`：外壳存在，但没有已启用且显式配置该外壳的账号。
- `upstream_model_not_found`：已命中账号，上游返回真实模型不存在。
- `upstream_error`：其他上游错误。

模型同步错误只返回脱敏 URL（scheme/host/port/path）、HTTP 状态、Content-Type 和受控错误类别；不得返回 query、认证头、Key、Token 或原始响应正文。

## 实施阶段与保存点

所有阶段遵循 RED -> GREEN -> 小范围回归。每个阶段绿色后单独 commit 并推送；高风险阶段再建立远端 checkpoint tag。

### 阶段 0：基线与版本保护

- 创建远端备份 tag `vps-backup/sub2api-before-cc-routing-r3-20260715`。
- 创建并跟踪分支 `vps/custom-cc-routing-r3`。
- 复跑 r2 前端定向测试，记录本机缺少 Go `1.26.4` 的限制。
- 保存点：`docs: lock Claude Code routing r3 execution contract`。

基线结果（2026-07-15）：4 个定向 Vitest 文件、12 项测试全部通过；本机 PATH 未安装 Go，后端基线由后续 GitHub Actions 验证。

### 阶段 1：稳定目录、严格调度与错误分类

主要文件：

- `backend/internal/pkg/claude/constants.go`
- `backend/internal/service/gateway_service.go`
- `backend/internal/service/account.go`
- 相关 service/handler 测试

先补失败测试：账号关闭后目录仍存在；未配置外壳的账号被排除；未知外壳、无合格账号和上游模型不存在返回不同错误；不得回退其他模型。

保存点：阶段 commit + `checkpoint/cc-routing-r3-catalog`。

### 阶段 2：账号路由、上下文、思考与模型同步

主要文件：

- `backend/internal/service/account.go`
- `backend/internal/service/gateway_request.go`
- `backend/internal/service/upstream_models.go`
- `backend/internal/handler/admin/account_handler.go`
- `frontend/src/components/account/ClaudeCodeConfigEditor.vue`
- `frontend/src/components/account/claudeCodeConfig.ts`
- `frontend/src/components/account/ModelWhitelistSelector.vue`
- `frontend/src/components/account/CreateAccountModal.vue`
- `frontend/src/components/account/EditAccountModal.vue`
- `frontend/src/api/admin/accounts.ts`
- `frontend/src/types/index.ts`
- 对应 Go/Vitest 文件

先补失败测试：新旧字段读取优先级、保存兼容投影、严格路由、五档生成/覆盖、不配置不注入、任意正整数上下文和 1M、共同安全最小上下文、同步失败保留手工输入、自定义 URL 脱敏诊断。

保存点：后端契约、前端编辑器、模型同步分别做可回退 commit；阶段完成后建立 `checkpoint/cc-routing-r3-account-routes`。

### 阶段 3：真实网关连接测试与 CLI 身份

主要文件：

- `backend/internal/service/account_test_service.go`
- `backend/internal/service/gateway_service.go`
- `backend/internal/service/openai_gateway_service.go`
- `frontend/src/components/account/AccountTestModal.vue`
- 对应 Go/Vitest 文件

普通测试固定当前账号，并复用生产 `GatewayService.Forward` 或 `OpenAIGatewayService.Forward`。Claude 模拟 Claude Code CLI，OpenAI 模拟 Codex CLI。禁止空值回退到 `hi/hello/ping/test` 等探活词；默认值和自定义值都必须是完整问题。Compact 只保留为 Codex 折叠高级测试。

测试结果至少显示：请求外壳、命中账号、映射后模型、CLI 身份、透传状态、思考映射、上游 HTTP 状态。

保存点：Claude 与 Codex 网关测试各自可验证后 commit；阶段 tag `checkpoint/cc-routing-r3-gateway-test`。

### 阶段 4：上游错误与 Token 刷新审计

主要文件：

- `backend/internal/service/oauth_refresh_api.go`
- `backend/internal/service/token_refresh_service.go`
- `backend/internal/handler/dto/types.go`
- `backend/internal/handler/dto/mappers.go`
- `backend/internal/repository/account_repo.go`
- `frontend/src/components/account/AccountTokenStatusCell.vue`
- `frontend/src/types/index.ts`
- 对应 Go/Vitest 文件

先补失败测试：API Key 无 AT/RT；无记录不伪造成功；手动/后台成功失败均落审计；错误脱敏；下次刷新窗口正确；上游 404 与其他错误分类正确。

保存点：错误契约和刷新审计分开 commit；阶段 tag `checkpoint/cc-routing-r3-observability`。

### 阶段 5：全量验证、构建与部署

```powershell
pnpm --dir frontend exec vitest run
pnpm --dir frontend typecheck
pnpm --dir frontend build
go -C backend test -tags unit ./internal/service ./internal/handler/admin ./internal/handler/dto -count=1
```

本机没有准确 Go 版本时，不用错误版本代替；由 GitHub Actions 使用仓库声明的 Go `1.26.4` 完成后端测试和构建。

生产部署前必须全部满足：

1. Git 工作树干净，所有 commit 已推远端，CI 全绿。
2. 生成 PostgreSQL dump 并校验非空及 SHA256。
3. 生成新的账号加密 `.json.enc`，拉回本机并用本机私钥解密验证。
4. 备份 `.env`、`config.yaml`、Compose 覆盖文件和当前容器/镜像状态；敏感备份只在 VPS/本机离线目录，不进 Git。
5. 验证当前 r2 镜像 `sha256:6ccd3881bc2a7a4d2bbbcae9c505576d6e81d646a0c9596137d76a9aeab59688` 仍可作为快速回滚入口。
6. 构建并部署不可变 `vps-custom-<git-sha>` 镜像，不使用浮动 tag 作为最终部署依据。
7. 只重建 `sub2api` 应用容器，不改 PostgreSQL、Redis、Nginx 或数据目录。
8. 验证健康检查、账号/用户/API Key 计数、真实 Claude Code CLI 和 Codex CLI 烟测。
9. 更新 `D:\workSpace\codex\VPS\modules\sub2api\README.md` 和 `stages\2026-07-15-sub2api-cc-routing-r3.md`。

若新的加密账号快照无法生成、拉回或解密，停止部署，先修复备份机制。

## Git 执行纪律

- 单分支串行实施；核心 service 文件共享度高，不拆并行 worktree。
- 只 stage 本阶段明确文件，不使用 `git add .`。
- 每个 commit 必须带对应测试证据并立即推送到 `morti/vps/custom-cc-routing-r3`。
- 不 amend 已推 commit，不 force-push，不改写 r2 历史。
- checkpoint tag 只指向已推送、测试绿色的 commit。
- 账号数据、数据库 dump、Token、API Key、`.env`、私钥和解密快照永不进入 Git/GitHub。
