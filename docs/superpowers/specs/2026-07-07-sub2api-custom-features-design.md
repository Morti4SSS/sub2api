# 2026-07-07 sub2api 自定义能力重建设计

> 状态：已废弃。2026-07-22 合并官方 `v0.1.163` 时已移除本文设计的 Claude Code 私有模型目录、官方模型外壳和思考强度映射；本文仅作为版本演进记录，不代表当前运行行为。

## 背景

当前 VPS 已部署 `MortiSSS-v0.1.146-r1`，运行代码主体收敛到官方 `Wei-Shaw/sub2api v0.1.146`。旧 `MortiSSS-v0.1.138-r2` 中的自定义测试词、Claude Code 模型映射、思考强度映射等没有作为当前运行能力继续携带。

本设计用于在 `v0.1.146` 基线上重新实现必要自定义能力，并保留账号数据安全、代码可合并和 VPS 可回滚。

## 目标

- 恢复 Codex / Claude Code 账号测试词自定义，测试时使用前端输入内容，空值才回退默认测试词。
- 恢复 Codex / Claude Code 客户端限制兼容能力，保留真实客户端标识透传，同时隔离会话 ID。
- 为 Claude Code CLI 提供通用模型映射配置，让 `/model` 能看到映射后的显示名和请求模型。
- 为 Claude Code CLI 提供通用思考强度映射配置，将 `low/medium/high/xhigh/max` 映射为上游实际字段和值。
- 在账号列表提供紧凑的 AT/RT 状态提示，便于快速判断账号刷新是否正常。
- 提高多中转站接入时的上游错误可见性，避免最终只显示 sub2api 的泛化 `no available accounts`。
- 保留已有轮换相关标识，不因重建这些能力而丢失账号池调度可读信息。
- 建立自定义能力维护台账，让后续跟进官方更新时能快速确认哪些改动必须保留、如何验证、如何回滚。

## 非目标

- 不新增任何非 Codex/OpenAI、Claude/Claude Code 的自定义能力或具体厂商预设。
- 不把账号凭证、token、cookie、私钥、Admin API Key 或数据库 dump 写入 GitHub。
- 不重做完整调度系统，不改变账号池选择策略本身。
- 不把上游错误正文无条件原样暴露给普通调用端。
- 不在本设计中修改 VPS 生产配置；部署阶段另行备份并记录。

## 总体方案

采用“通用配置 + 安全摘要 + 管理端可见详情”的方案。

- 模型和强度映射继续放在账号 `extra` 中，由账号配置维护，避免引入新表和迁移。
- Claude Code 的可见模型目录由账号 `extra.claude_code_model_catalog` 提供，既用于 `/v1/models` 展示，也用于请求模型到上游模型的解析。
- 思考强度映射由账号 `extra.claude_code_effort_mapping` 提供，只在识别为 Claude Code CLI 的 Anthropic 路径上生效。
- AT/RT 状态由后端账号列表响应提供脱敏摘要，前端以紧凑小字展示。
- 上游错误在服务层统一提取脱敏摘要，调用端返回短摘要，管理端错误详情和账号列表可查看更具体的脱敏信息。
- 自定义维护说明放在仓库根目录 `CUSTOMIZATIONS.md`，记录远端关系、分支/tag 规则、配置契约、验证命令和升级流程；账号数据和 VPS 私密备份仍只保存在 VPS 或本地安全位置。

## 模型映射设计

配置 owner 为账号 `extra.claude_code_model_catalog`。

每个条目描述一个 Claude Code 可见模型：

```json
{
  "claude_code_model_catalog": [
    {
      "role": "sonnet",
      "request_model": "cc-relay-sonnet",
      "display_name": "Relay Sonnet",
      "upstream_model": "provider-sonnet",
      "supports_1m": false,
      "capabilities": ["thinking"]
    }
  ]
}
```

运行规则：

- `/v1/models` 仅在 Claude Code CLI 请求上下文中优先返回 catalog。
- 请求中的 `model` 命中 catalog `request_model` 时，转发给上游使用 `upstream_model`。
- 普通 Anthropic 请求不使用 Claude Code catalog，避免影响非 Claude Code 客户端。
- catalog 为空时保持官方 `v0.1.146` 现有行为。
- 前端账号编辑页提供 JSON/结构化编辑入口，并展示 `/model` 预览。

## 思考强度映射设计

配置 owner 为账号 `extra.claude_code_effort_mapping`。

配置以 Claude Code 可见模型 `request_model` 为 key：

```json
{
  "claude_code_effort_mapping": {
    "cc-relay-sonnet": {
      "target_field": "reasoning_effort",
      "values": {
        "low": "low",
        "medium": "medium",
        "high": "high",
        "xhigh": "high",
        "max": "high"
      }
    }
  }
}
```

支持的目标字段：

- `reasoning_effort`
- `reasoning.effort`
- `output_config.effort`
- `thinking.budget_tokens`

运行规则：

- Claude Code 请求存在强度档位时，按映射写入上游请求体。
- 上游档位少于 Claude Code 档位时，允许多个 Claude Code 档位折叠到同一个上游值。
- 未配置某档位或请求未显式携带档位时，使用映射中最高可用档位，优先级为 `max > xhigh > high > medium > low`。
- 完全未配置强度映射时，默认启用“最强可表达思考模式”：优先 `max`，无法表达时不强行注入未知字段。
- 前端在编辑页展示“档位折叠提示”，例如 `xhigh/max -> high`。

## 测试词自定义设计

前端账号测试弹窗保存本地上次输入的测试词，并在发起测试时传给后端。

后端测试链路要求：

- Codex / OpenAI Responses 测试读取 `prompt`。
- Claude / Anthropic 测试读取 `prompt`。
- 空白或全空格才回退默认值。
- 默认测试词保持短且低风险，例如 `Respond with OK.`。

## 客户端限制兼容设计

保留旧版本已验证的方向：

- 对允许 Codex CLI / Claude Code CLI 的 OpenAI OAuth 账号，转发真实 `User-Agent` 和 `Originator`。
- 对会话相关头继续重写或隔离，避免跨用户会话碰撞。
- 不扩大到未配置允许客户端的账号。
- Codex 和 Claude Code 两条路径分别加测试，避免只修一边。

## 账号列表 AT/RT 状态设计

账号列表新增紧凑状态展示，建议列名为 `AT/RT`。

后端响应只提供脱敏摘要：

```json
{
  "token_status": {
    "access_token": "refreshing",
    "refresh_token": "present",
    "refresh_window_hours": 12,
    "last_refresh_at": "2026-07-07T20:30:00+08:00",
    "last_refresh_result": "success",
    "last_refresh_error": ""
  }
}
```

前端展示规则：

- 小字号、紧凑行高，适合快速扫列表。
- 正常：`AT 自动 / RT 有 / 12h`
- 异常：`RT 缺失`、`刷新失败`、`AT 将过期`
- 鼠标悬停或展开时显示最近刷新时间和脱敏失败原因。
- 不显示 AT、RT 原文，不显示 credentials 内部字段值。

## 上游错误可见性设计

问题场景：

多中转站接入时，上游可能返回余额不足、模型不可用、账号限制、鉴权失败、限流、服务故障等错误。当前最终响应常塌缩为 sub2api 的 `Service temporarily unavailable` 或 `no available accounts`，使用者无法判断是哪一个中转站出了什么问题。

设计分层：

- 调用端响应：返回安全摘要，说明最后一次或主要上游失败类型。
- 管理端详情：展示脱敏后的上游状态码、错误摘要、账号名/分组/模型/中转站标识。
- 运维日志：保留已脱敏的上游响应片段，便于排查。

安全摘要示例：

```json
{
  "error": {
    "type": "api_error",
    "message": "No available accounts. Last upstream error: 403 provider rejected request: model unavailable"
  }
}
```

管理端摘要示例：

```text
上游 403 / relay-a / model glm-5.1 / 模型不可用 / 3 分钟前
```

运行规则：

- 上游 HTTP 错误进入 `UpstreamFailoverError` 时提取状态码和错误 message。
- 最后一轮账号 failover 耗尽时，优先使用最后一个脱敏上游错误摘要补充响应。
- 如果没有上游错误，只保留现有 `no available accounts` 逻辑。
- 错误提取必须过滤 token、authorization、cookie、key、secret、完整 URL query 等敏感片段。
- 管理端可查看更多脱敏上下文，普通用户和调用端只看短摘要。

## 错误处理

- 配置 JSON 解析失败时，账号编辑页阻止保存并显示具体字段。
- catalog 中 `id` 重复时保存失败。
- effort `target_field` 不在允许列表时保存失败。
- 上游错误正文过大时只截断摘要，不保存全文。
- 账号列表 token 状态读取失败时显示 `状态未知`，不影响账号列表加载。

## 测试计划

后端最小测试：

- Codex / Claude Code 账号测试使用自定义 prompt，空值回退默认 prompt。
- Claude Code `/v1/models` 返回 catalog 显示名和模型 ID。
- Claude Code 请求模型按 catalog 映射到 `upstream_model`。
- Claude Code `low/medium/high/xhigh/max` 按 effort mapping 写入目标字段。
- 未配置 effort mapping 时不破坏现有请求。
- 上游 403/429/5xx failover 耗尽后，最终错误包含脱敏上游摘要。
- 脱敏函数过滤 token、authorization、cookie、api key、query secret。
- 账号列表响应包含 token 状态摘要且不包含 AT/RT 原文。

前端最小测试：

- 账号测试弹窗提交自定义 prompt。
- Claude Code 配置编辑器保存 catalog 和 effort mapping。
- `/model` 预览和强度折叠提示正确显示。
- 账号列表 `AT/RT` 状态列正常、异常、未知三类展示。
- 错误详情展示上游摘要，不出现敏感字段。

部署验证：

- 本地或 CI 通过后端相关 Go 测试。
- 前端 `pnpm typecheck` 和相关 Vitest 通过。
- 构建自有 GHCR 镜像。
- VPS 部署前备份 PostgreSQL dump、`.env`、Compose 文件、`config.yaml` 和当前镜像状态。
- 部署后验证容器健康、`/health`、版本号、测试词、Claude Code `/model`、强度映射、账号列表 AT/RT 状态和上游错误摘要。

## 回滚

- 源码使用新分支开发，推送新分支，不直接推 main/master。
- 部署前创建源码 tag 和 VPS 数据备份。
- 镜像异常时先回滚 `SUB2API_IMAGE` 到 `MortiSSS-v0.1.146-r1` 或部署前镜像。
- 数据库恢复必须单独确认，因为会覆盖备份时间点之后的数据。
- 后续跟进官方更新前，先对照 `CUSTOMIZATIONS.md` 的能力台账确认当前自定义差异，更新后再逐项跑回归验证。

## 待实施计划覆盖

- 对比 `vps-backup/sub2api-before-v0.1.146-20260707` 中旧 r2 实现，只借鉴通过验证的测试和配置契约，不直接整段搬代码。
- 在 `v0.1.146` 基线上按测试词、Claude Code 映射、AT/RT 状态、上游错误透明化四个小阶段实现。
- 每个阶段先补最小测试，再实现，再运行局部验证。
- 同步维护 `CUSTOMIZATIONS.md`，把自定义能力固定成可审计的台账和上游更新流程。
