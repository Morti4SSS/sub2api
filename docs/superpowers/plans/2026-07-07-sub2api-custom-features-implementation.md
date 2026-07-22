# Sub2API Custom Features Implementation Plan

> **Deprecated:** The private Claude Code model catalog, official-model shell, and effort-mapping work in this plan was removed when upstream `v0.1.163` was merged on 2026-07-22. This file is retained only as implementation history and does not describe current runtime behavior.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the required Codex / Claude Code custom features on top of `MortiSSS-v0.1.146-r1` without losing upstream mergeability or VPS rollback safety.

**Architecture:** Keep all user-facing custom behavior behind existing account `extra`, DTO, and gateway boundaries. Add narrow helpers for Claude Code catalog / effort mapping, token-status summaries, and upstream-error summaries instead of restructuring the gateway or scheduler.

**Tech Stack:** Go backend, Gin handlers, Ent-backed service models, Vue 3 frontend, TypeScript, Vitest, GitHub Actions, Docker Compose on VPS.

---

## Scope Check

The spec spans four visible areas, but they share the same deployment target and account/gateway contracts. Implement them as sequential, independently testable tasks:

- Account test prompt fix.
- Claude Code model and effort mapping.
- Account-list AT/RT status.
- Upstream error visibility.
- Deployment and rollback documentation.
- Customization ledger and upstream-sync workflow.

Do not introduce custom features outside Codex/OpenAI and Claude/Claude Code. Do not add provider-specific presets; configuration must stay generic.

## File Structure

Backend:

- Create `CUSTOMIZATIONS.md`: record the private customization ledger, config owners, upstream-sync workflow, verification commands, and rollback rules.

- Modify `backend/internal/service/account_test_service.go`: pass custom prompt into Claude and OpenAI test payloads.
- Modify `backend/internal/service/account_test_service_openai_test.go`: add prompt regression tests for Claude and OpenAI paths.
- Modify `backend/internal/service/account.go`: add Claude Code catalog parsing and effort mapping helpers backed by `accounts.extra`.
- Modify `backend/internal/service/gateway_request.go`: add or restore generic effort mapping function if it is missing in this baseline.
- Modify `backend/internal/service/gateway_service.go`: apply Claude Code catalog model resolution and effort mapping on Anthropic request forwarding paths.
- Modify `backend/internal/handler/gateway_handler.go`: expose Claude Code catalog entries from `/v1/models` for Claude Code contexts and improve failover-exhausted messages.
- Modify `backend/internal/handler/openai_gateway_handler.go`, `backend/internal/handler/gateway_handler_responses.go`, `backend/internal/handler/gateway_handler_chat_completions.go`: use shared upstream-error summary helper on OpenAI/Codex and Claude-compatible exits.
- Create `backend/internal/service/upstream_error_summary.go`: build safe client-facing upstream error summaries.
- Create `backend/internal/service/upstream_error_summary_test.go`: test extraction and redaction.
- Modify `backend/internal/handler/dto/types.go`: add `TokenStatus` DTO.
- Modify `backend/internal/handler/dto/mappers.go`: derive account `token_status` without exposing token values.
- Modify `backend/internal/handler/dto/account_mapper_redact_test.go`: test AT/RT summary and redaction.

Frontend:

- Modify `frontend/src/components/account/AccountTestModal.vue` and `frontend/src/components/admin/account/AccountTestModal.vue`: send trimmed prompt for OpenAI/Anthropic text tests and existing image tests.
- Modify `frontend/src/components/account/__tests__/AccountTestModal.spec.ts` and `frontend/src/components/admin/account/__tests__/AccountTestModal.spec.ts`: assert prompt is sent for text tests.
- Create `frontend/src/components/account/claudeCodeConfig.ts`: generic catalog / effort config reader and writer.
- Create `frontend/src/components/account/ClaudeCodeConfigEditor.vue`: compact editor for Claude Code catalog and effort mappings.
- Modify `frontend/src/components/account/CreateAccountModal.vue` and `frontend/src/components/account/EditAccountModal.vue`: mount the config editor for Anthropic API Key / compatible relay accounts.
- Create `frontend/src/components/account/__tests__/claudeCodeConfig.spec.ts`: test config reader/writer and effort fold summaries.
- Create `frontend/src/components/account/AccountTokenStatusCell.vue`: compact AT/RT list display.
- Create `frontend/src/components/account/__tests__/AccountTokenStatusCell.spec.ts`: test normal, failed, missing RT, and unknown states.
- Modify `frontend/src/views/admin/AccountsView.vue`: add `AT/RT` column and cell.
- Modify `frontend/src/types/index.ts`: add `AccountTokenStatus` and account field.
- Modify `frontend/src/i18n/locales/zh.ts` and `frontend/src/i18n/locales/en.ts`: add labels.

Docs / deploy:

- Modify `docs/superpowers/specs/2026-07-07-sub2api-custom-features-design.md` only if implementation discovers a contract mismatch.
- Modify `D:\workSpace\codex\VPS\modules\sub2api\README.md` after deployment.
- Create `D:\workSpace\codex\VPS\stages\YYYY-MM-DD-sub2api-custom-features-r2.md` after deployment.

---

### Task 0: Add Customization Maintenance Ledger

**Files:**
- Create: `CUSTOMIZATIONS.md`
- Modify: `docs/superpowers/specs/2026-07-07-sub2api-custom-features-design.md`
- Modify: `docs/superpowers/plans/2026-07-07-sub2api-custom-features-implementation.md`

- [x] **Step 1: Record the long-term customization ledger**

Create `CUSTOMIZATIONS.md` at the repository root. It must document:

- remote ownership: `upstream` for official `Wei-Shaw/sub2api`, `morti` for custom deploy branches and tags.
- branch and tag rules: custom work on `vps/*` or `vps/custom-*`; deploy tags under `vps-deploy/*`; never push `main` / `master` for this work.
- customization ledger: custom prompt tests, Codex / Claude Code passthrough, Claude Code catalog, Claude Code effort mapping, AT/RT status, upstream error summaries, and rotation visibility.
- config owners: `extra.claude_code_model_catalog` and `extra.claude_code_effort_mapping`.
- upstream update workflow: fetch official, sync branch, merge official tag, reapply/verify custom ledger, push new branch, deploy with backup.
- sensitive-data rule: account data, `.env`, `config.yaml`, dumps, token, cookie, and keys never go to GitHub.

- [x] **Step 2: Link the ledger from the design and plan**

Update the design and this plan so future workers know that `CUSTOMIZATIONS.md` is part of the required maintenance surface, not a temporary note.

### Task 1: Fix Account Test Prompt Propagation

**Files:**
- Modify: `frontend/src/components/account/AccountTestModal.vue`
- Modify: `frontend/src/components/admin/account/AccountTestModal.vue`
- Modify: `frontend/src/components/account/__tests__/AccountTestModal.spec.ts`
- Modify: `frontend/src/components/admin/account/__tests__/AccountTestModal.spec.ts`
- Modify: `backend/internal/service/account_test_service.go`
- Modify: `backend/internal/service/account_test_service_openai_test.go`

- [ ] **Step 1: Write frontend failing tests for text prompt submission**

In `frontend/src/components/account/__tests__/AccountTestModal.spec.ts`, add this test under the existing `describe('AccountTestModal')` block:

```ts
it('sends custom prompt for text account tests', async () => {
  const wrapper = mount(AccountTestModal, {
    props: {
      show: true,
      account: {
        id: 9,
        name: 'claude-relay',
        platform: 'anthropic',
        type: 'apikey'
      } as any
    },
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /></div>' },
        Select: { template: '<select />' }
      },
      mocks: {
        $t: (key: string) => key
      }
    }
  })

  await flushPromises()
  const textarea = wrapper.find('textarea')
  await textarea.setValue('  explain status  ')
  await wrapper.find('button[type="submit"]').trigger('submit')
  await flushPromises()

  const call = (global.fetch as any).mock.calls[0]
  expect(JSON.parse(call[1].body)).toMatchObject({
    model_id: expect.any(String),
    prompt: 'explain status'
  })
})
```

Add the same behavioral test to `frontend/src/components/admin/account/__tests__/AccountTestModal.spec.ts`, adapting the mount helper already present in that file.

- [ ] **Step 2: Run frontend tests and confirm failure**

Run:

```powershell
cd frontend
pnpm vitest run src/components/account/__tests__/AccountTestModal.spec.ts src/components/admin/account/__tests__/AccountTestModal.spec.ts
```

Expected: at least one test fails because text tests currently send `prompt: ''`.

- [ ] **Step 3: Fix both frontend request bodies**

In `frontend/src/components/account/AccountTestModal.vue`, replace:

```ts
prompt: supportsImageTest.value ? testPrompt.value.trim() : '',
```

with:

```ts
prompt: testPrompt.value.trim(),
```

In `frontend/src/components/admin/account/AccountTestModal.vue`, replace:

```ts
prompt: supportsImageTest.value ? testPrompt.value.trim() : ''
```

with:

```ts
prompt: testPrompt.value.trim()
```

- [ ] **Step 4: Add backend Claude prompt regression test**

In `backend/internal/service/account_test_service_openai_test.go`, add a test that calls the Claude payload builder directly:

```go
func TestCreateClaudeTestPayloadUsesCustomPrompt(t *testing.T) {
	payload, err := createTestPayloadWithPrompt("claude-sonnet-4-5", "say compact ok")
	require.NoError(t, err)

	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	require.Contains(t, string(raw), "say compact ok")
	require.NotContains(t, string(raw), "hi")
}

func TestCreateClaudeTestPayloadDefaultsBlankPrompt(t *testing.T) {
	payload, err := createTestPayloadWithPrompt("claude-sonnet-4-5", "   ")
	require.NoError(t, err)

	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	require.Contains(t, string(raw), "Respond with OK.")
}
```

Add imports if missing:

```go
import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)
```

- [ ] **Step 5: Implement prompt-aware Claude payload**

In `backend/internal/service/account_test_service.go`, replace `createTestPayload(modelID string)` with this wrapper and helper:

```go
const defaultAccountTestPrompt = "Respond with OK."

func createTestPayload(modelID string) (map[string]any, error) {
	return createTestPayloadWithPrompt(modelID, defaultAccountTestPrompt)
}

func createTestPayloadWithPrompt(modelID string, prompt string) (map[string]any, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		prompt = defaultAccountTestPrompt
	}
	if strings.TrimSpace(modelID) == "" {
		modelID = claude.DefaultTestModel
	}
	return map[string]any{
		"model":      modelID,
		"max_tokens": 32,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "text",
						"text": prompt,
					},
				},
			},
		},
	}, nil
}
```

Change `TestAccountConnection` to pass the prompt to Claude:

```go
return s.testClaudeAccountConnection(c, account, modelID, prompt)
```

Change the method signature:

```go
func (s *AccountTestService) testClaudeAccountConnection(c *gin.Context, account *Account, modelID string, prompt string) error {
```

Change the payload creation call:

```go
payload, err := createTestPayloadWithPrompt(testModelID, prompt)
```

Leave `routeAntigravityTest` outside this custom prompt change; the scoped prompt fix only covers Codex/OpenAI and Claude/Anthropic accounts.

- [ ] **Step 6: Run backend tests**

Run:

```powershell
cd backend
go test ./internal/service -run "TestCreateClaudeTestPayloadUsesCustomPrompt|TestCreateClaudeTestPayloadDefaultsBlankPrompt|TestAccountTestService_TestAccountConnection_OpenAI" -count=1
```

Expected: PASS.

- [ ] **Step 7: Run frontend tests again**

Run:

```powershell
cd frontend
pnpm vitest run src/components/account/__tests__/AccountTestModal.spec.ts src/components/admin/account/__tests__/AccountTestModal.spec.ts
```

Expected: PASS.

- [ ] **Step 8: Commit checkpoint only after user approval**

If the user has explicitly approved commits, run:

```bash
git add frontend/src/components/account/AccountTestModal.vue frontend/src/components/admin/account/AccountTestModal.vue frontend/src/components/account/__tests__/AccountTestModal.spec.ts frontend/src/components/admin/account/__tests__/AccountTestModal.spec.ts backend/internal/service/account_test_service.go backend/internal/service/account_test_service_openai_test.go
git commit -m "fix: honor custom account test prompts"
```

If commit approval has not been given, leave the working tree uncommitted and continue.

---

### Task 2: Restore Generic Claude Code Config UI

**Files:**
- Create: `frontend/src/components/account/claudeCodeConfig.ts`
- Create: `frontend/src/components/account/ClaudeCodeConfigEditor.vue`
- Create: `frontend/src/components/account/__tests__/claudeCodeConfig.spec.ts`
- Modify: `frontend/src/components/account/CreateAccountModal.vue`
- Modify: `frontend/src/components/account/EditAccountModal.vue`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`

- [ ] **Step 1: Write config helper tests**

Create `frontend/src/components/account/__tests__/claudeCodeConfig.spec.ts`:

```ts
import { describe, expect, it } from 'vitest'
import {
  buildClaudeCodeEffortSummary,
  createEmptyClaudeCodeCatalogEntry,
  readClaudeCodeCatalog,
  readClaudeCodeEffortMappings,
  writeClaudeCodeConfigToExtra
} from '../claudeCodeConfig'

describe('claudeCodeConfig', () => {
  it('reads and writes catalog entries', () => {
    const extra: Record<string, unknown> = {}
    writeClaudeCodeConfigToExtra(extra, [
      {
        role: 'sonnet',
        displayName: 'Relay Sonnet',
        requestModel: 'relay-sonnet',
        upstreamModel: 'upstream-sonnet',
        supports1m: true,
        capabilities: 'thinking, effort'
      }
    ], [])

    expect(extra.claude_code_model_catalog).toEqual([
      {
        role: 'sonnet',
        display_name: 'Relay Sonnet',
        request_model: 'relay-sonnet',
        upstream_model: 'upstream-sonnet',
        supports_1m: true,
        capabilities: ['thinking', 'effort']
      }
    ])
    expect(readClaudeCodeCatalog(extra)[0].requestModel).toBe('relay-sonnet')
  })

  it('summarizes folded effort mappings', () => {
    expect(buildClaudeCodeEffortSummary({
      model: 'relay-sonnet',
      targetField: 'reasoning_effort',
      low: 'low',
      medium: 'medium',
      high: 'high',
      xhigh: 'high',
      max: 'high'
    })).toBe('low -> low; medium -> medium; high/xhigh/max -> high')
  })

  it('returns one empty row for missing config', () => {
    expect(readClaudeCodeCatalog({})).toEqual([createEmptyClaudeCodeCatalogEntry()])
    expect(readClaudeCodeEffortMappings({})[0].targetField).toBe('output_config.effort')
  })
})
```

- [ ] **Step 2: Run helper tests and confirm failure**

Run:

```powershell
cd frontend
pnpm vitest run src/components/account/__tests__/claudeCodeConfig.spec.ts
```

Expected: FAIL because `claudeCodeConfig.ts` does not exist.

- [ ] **Step 3: Add `claudeCodeConfig.ts`**

Create `frontend/src/components/account/claudeCodeConfig.ts`:

```ts
export interface ClaudeCodeCatalogForm {
  role: string
  displayName: string
  requestModel: string
  upstreamModel: string
  supports1m: boolean
  capabilities: string
}

export interface ClaudeCodeEffortForm {
  model: string
  targetField: string
  low: string
  medium: string
  high: string
  xhigh: string
  max: string
}

export const createEmptyClaudeCodeCatalogEntry = (): ClaudeCodeCatalogForm => ({
  role: '',
  displayName: '',
  requestModel: '',
  upstreamModel: '',
  supports1m: false,
  capabilities: ''
})

export const createEmptyClaudeCodeEffortEntry = (): ClaudeCodeEffortForm => ({
  model: '',
  targetField: 'output_config.effort',
  low: '',
  medium: '',
  high: '',
  xhigh: '',
  max: ''
})

export const buildClaudeCodeEffortSummary = (entry: ClaudeCodeEffortForm) => {
  const groups = new Map<string, string[]>()
  for (const level of ['low', 'medium', 'high', 'xhigh', 'max'] as const) {
    const mapped = entry[level].trim()
    if (!mapped) continue
    const existing = groups.get(mapped) ?? []
    groups.set(mapped, existing.concat(level))
  }
  return Array.from(groups.entries())
    .map(([mapped, levels]) => `${levels.join('/')} -> ${mapped}`)
    .join('; ')
}

const stringValue = (value: unknown) => (typeof value === 'string' ? value : '').trim()

const stringListValue = (value: unknown): string[] => {
  if (!Array.isArray(value)) return []
  return value.map((item) => stringValue(item)).filter(Boolean)
}

export const readClaudeCodeCatalog = (extra?: Record<string, unknown>): ClaudeCodeCatalogForm[] => {
  const raw = extra?.claude_code_model_catalog
  if (!Array.isArray(raw)) return [createEmptyClaudeCodeCatalogEntry()]
  const entries = raw
    .map((item) => {
      if (!item || typeof item !== 'object') return null
      const data = item as Record<string, unknown>
      return {
        role: stringValue(data.role),
        displayName: stringValue(data.display_name),
        requestModel: stringValue(data.request_model),
        upstreamModel: stringValue(data.upstream_model),
        supports1m: data.supports_1m === true,
        capabilities: stringListValue(data.capabilities).join(', ')
      }
    })
    .filter((item): item is ClaudeCodeCatalogForm => item !== null)
  return entries.length > 0 ? entries : [createEmptyClaudeCodeCatalogEntry()]
}

export const readClaudeCodeEffortMappings = (extra?: Record<string, unknown>): ClaudeCodeEffortForm[] => {
  const raw = extra?.claude_code_effort_mapping
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return [createEmptyClaudeCodeEffortEntry()]
  const entries = Object.entries(raw as Record<string, unknown>)
    .map(([model, item]) => {
      if (!item || typeof item !== 'object' || Array.isArray(item)) return null
      const data = item as Record<string, unknown>
      const values = data.values && typeof data.values === 'object' && !Array.isArray(data.values)
        ? data.values as Record<string, unknown>
        : {}
      return {
        model: model.trim(),
        targetField: stringValue(data.target_field) || 'output_config.effort',
        low: stringValue(values.low),
        medium: stringValue(values.medium),
        high: stringValue(values.high),
        xhigh: stringValue(values.xhigh),
        max: stringValue(values.max)
      }
    })
    .filter((item): item is ClaudeCodeEffortForm => item !== null)
  return entries.length > 0 ? entries : [createEmptyClaudeCodeEffortEntry()]
}

export const writeClaudeCodeConfigToExtra = (
  extra: Record<string, unknown>,
  catalog: ClaudeCodeCatalogForm[],
  effortMappings: ClaudeCodeEffortForm[]
) => {
  const catalogPayload = catalog
    .map((entry) => {
      const role = entry.role.trim()
      const displayName = entry.displayName.trim()
      const requestModel = entry.requestModel.trim()
      if (!role || !displayName || !requestModel) return null
      const payload: Record<string, unknown> = { role, display_name: displayName, request_model: requestModel }
      const upstreamModel = entry.upstreamModel.trim()
      if (upstreamModel) payload.upstream_model = upstreamModel
      if (entry.supports1m) payload.supports_1m = true
      const capabilities = entry.capabilities.split(',').map((item) => item.trim()).filter(Boolean)
      if (capabilities.length > 0) payload.capabilities = capabilities
      return payload
    })
    .filter((item): item is Record<string, unknown> => item !== null)

  if (catalogPayload.length > 0) extra.claude_code_model_catalog = catalogPayload
  else delete extra.claude_code_model_catalog

  const effortPayload: Record<string, unknown> = {}
  for (const [index, entry] of effortMappings.entries()) {
    const model = entry.model.trim() || catalog[index]?.requestModel.trim() || ''
    if (!model) continue
    const values: Record<string, string> = {}
    for (const key of ['low', 'medium', 'high', 'xhigh', 'max'] as const) {
      const value = entry[key].trim()
      if (value) values[key] = value
    }
    if (Object.keys(values).length === 0) continue
    effortPayload[model] = {
      target_field: entry.targetField.trim() || 'output_config.effort',
      values
    }
  }

  if (Object.keys(effortPayload).length > 0) extra.claude_code_effort_mapping = effortPayload
  else delete extra.claude_code_effort_mapping
}
```

- [ ] **Step 4: Add config editor component**

Create `frontend/src/components/account/ClaudeCodeConfigEditor.vue` with a compact form that imports the helpers above. The component must expose `v-model:catalog` and `v-model:effortMappings`, render rows with `data-testid` prefixes `claude-code-catalog-*` and `claude-code-effort-*`, and display `buildClaudeCodeEffortSummary(entry)` below each effort row.

Use the old r2 component as reference:

```powershell
git show vps-backup/sub2api-before-v0.1.146-20260707:frontend/src/components/account/ClaudeCodeConfigEditor.vue
```

When copying, keep only generic catalog and effort controls; do not add provider presets.

- [ ] **Step 5: Wire editor into create/edit modals**

In `CreateAccountModal.vue` and `EditAccountModal.vue`:

```ts
import ClaudeCodeConfigEditor from './ClaudeCodeConfigEditor.vue'
import {
  createEmptyClaudeCodeCatalogEntry,
  createEmptyClaudeCodeEffortEntry,
  readClaudeCodeCatalog,
  readClaudeCodeEffortMappings,
  writeClaudeCodeConfigToExtra,
  type ClaudeCodeCatalogForm,
  type ClaudeCodeEffortForm
} from './claudeCodeConfig'
```

Add state:

```ts
const claudeCodeCatalog = ref<ClaudeCodeCatalogForm[]>([createEmptyClaudeCodeCatalogEntry()])
const claudeCodeEffortMappings = ref<ClaudeCodeEffortForm[]>([createEmptyClaudeCodeEffortEntry()])
```

When loading existing account `extra`, call:

```ts
claudeCodeCatalog.value = readClaudeCodeCatalog(account.extra as Record<string, unknown> | undefined)
claudeCodeEffortMappings.value = readClaudeCodeEffortMappings(account.extra as Record<string, unknown> | undefined)
```

Before submit, after building `extra`, call:

```ts
writeClaudeCodeConfigToExtra(extra, claudeCodeCatalog.value, claudeCodeEffortMappings.value)
```

Render the editor only for Anthropic accounts that can represent third-party relay models:

```vue
<ClaudeCodeConfigEditor
  v-if="platform === 'anthropic' && accountType === 'apikey'"
  v-model:catalog="claudeCodeCatalog"
  v-model:effort-mappings="claudeCodeEffortMappings"
/>
```

- [ ] **Step 6: Add locale labels**

In `frontend/src/i18n/locales/zh.ts`, add labels under `admin.accounts.anthropic`:

```ts
claudeCodeMapping: 'Claude Code 模型映射',
claudeCodeMappingDesc: '配置 Claude Code /model 可见模型、显示名、上游模型和思考强度映射。',
claudeCodeRole: '角色',
claudeCodeDisplayName: '显示名',
claudeCodeRequestModel: '请求模型',
claudeCodeUpstreamModel: '上游模型',
claudeCodeSupports1m: '支持 1M 上下文',
claudeCodeCapabilities: '能力',
claudeCodeAddModel: '添加模型',
claudeCodeEffortMapping: '思考强度映射',
claudeCodeEffortModel: '模型',
claudeCodeEffortTarget: '目标字段',
claudeCodeEffortSummary: '折叠关系'
```

Add English equivalents in `frontend/src/i18n/locales/en.ts`.

- [ ] **Step 7: Run frontend checks**

Run:

```powershell
cd frontend
pnpm vitest run src/components/account/__tests__/claudeCodeConfig.spec.ts
pnpm typecheck
```

Expected: PASS.

- [ ] **Step 8: Commit checkpoint only after user approval**

If commit approval has been given:

```bash
git add frontend/src/components/account/claudeCodeConfig.ts frontend/src/components/account/ClaudeCodeConfigEditor.vue frontend/src/components/account/__tests__/claudeCodeConfig.spec.ts frontend/src/components/account/CreateAccountModal.vue frontend/src/components/account/EditAccountModal.vue frontend/src/i18n/locales/zh.ts frontend/src/i18n/locales/en.ts
git commit -m "feat: add claude code mapping editor"
```

---

### Task 3: Restore Claude Code Runtime Mapping

**Files:**
- Modify: `backend/internal/service/account.go`
- Modify: `backend/internal/service/gateway_request.go`
- Modify: `backend/internal/service/gateway_request_test.go`
- Modify: `backend/internal/service/gateway_service.go`
- Modify: `backend/internal/service/gateway_anthropic_apikey_passthrough_test.go`
- Modify: `backend/internal/handler/gateway_handler.go`
- Modify: `backend/internal/handler/gateway_models_test.go`

- [ ] **Step 1: Add backend tests for account config parsing**

In `backend/internal/service/account_wildcard_test.go`, add:

```go
func TestAccountClaudeCodeModelCatalogConfig(t *testing.T) {
	acc := &Account{
		Extra: map[string]any{
			"claude_code_model_catalog": []any{
				map[string]any{
					"role":           "sonnet",
					"display_name":   "Relay Sonnet",
					"request_model":  "relay-sonnet",
					"upstream_model": "upstream-sonnet",
					"supports_1m":    true,
					"capabilities":   []any{"thinking", "effort"},
				},
			},
		},
	}

	catalog := acc.GetClaudeCodeModelCatalog()
	require.Len(t, catalog, 1)
	require.Equal(t, "relay-sonnet", catalog[0].RequestModel)
	require.Equal(t, "upstream-sonnet", catalog[0].UpstreamModel)
	require.True(t, catalog[0].Supports1M)

	upstream, ok := acc.ResolveClaudeCodeCatalogModel("relay-sonnet")
	require.True(t, ok)
	require.Equal(t, "upstream-sonnet", upstream)
}
```

- [ ] **Step 2: Add backend tests for effort mapping**

In `backend/internal/service/gateway_request_test.go`, add:

```go
func TestClaudeCodeEffortMappingMapsXHighToHigh(t *testing.T) {
	body := []byte(`{"model":"relay-sonnet","output_config":{"effort":"xhigh"},"messages":[]}`)
	acc := &Account{
		Extra: map[string]any{
			"claude_code_effort_mapping": map[string]any{
				"relay-sonnet": map[string]any{
					"target_field": "reasoning_effort",
					"values": map[string]any{
						"xhigh": "high",
						"max":   "high",
					},
				},
			},
		},
	}

	got, changed := ApplyClaudeCodeEffortMapping(body, acc, "relay-sonnet")
	require.True(t, changed)
	require.Equal(t, "high", gjson.GetBytes(got, "reasoning_effort").String())
}
```

Ensure imports include:

```go
import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)
```

- [ ] **Step 3: Run tests and confirm failure**

Run:

```powershell
cd backend
go test ./internal/service -run "TestAccountClaudeCodeModelCatalogConfig|TestClaudeCodeEffortMappingMapsXHighToHigh" -count=1
```

Expected: FAIL because helper methods are missing or incomplete.

- [ ] **Step 4: Add account catalog helpers**

In `backend/internal/service/account.go`, add:

```go
type ClaudeCodeModelCatalogEntry struct {
	Role          string
	DisplayName   string
	RequestModel  string
	UpstreamModel string
	Supports1M    bool
	Capabilities  []string
}

func (a *Account) GetClaudeCodeModelCatalog() []ClaudeCodeModelCatalogEntry {
	if a == nil || a.Extra == nil {
		return nil
	}
	raw, ok := a.Extra["claude_code_model_catalog"].([]any)
	if !ok {
		return nil
	}

	out := make([]ClaudeCodeModelCatalogEntry, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry := ClaudeCodeModelCatalogEntry{
			Role:          extraStringValue(m["role"]),
			DisplayName:   extraStringValue(m["display_name"]),
			RequestModel:  extraStringValue(m["request_model"]),
			UpstreamModel: extraStringValue(m["upstream_model"]),
			Supports1M:    extraBoolValue(m["supports_1m"]),
			Capabilities:  extraStringSliceValue(m["capabilities"]),
		}
		if entry.Role == "" || entry.DisplayName == "" || entry.RequestModel == "" {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func (a *Account) ResolveClaudeCodeCatalogModel(requestModel string) (string, bool) {
	requestModel = strings.TrimSpace(requestModel)
	if requestModel == "" {
		return "", false
	}
	for _, entry := range a.GetClaudeCodeModelCatalog() {
		if entry.RequestModel != requestModel {
			continue
		}
		if entry.UpstreamModel != "" {
			return entry.UpstreamModel, true
		}
		return entry.RequestModel, true
	}
	return "", false
}
```

If `extraStringValue`, `extraBoolValue`, or `extraStringSliceValue` already exist, reuse them. If not, add small helpers in the same file near other `extra` helpers.

- [ ] **Step 5: Add effort mapping helper**

In `backend/internal/service/gateway_request.go`, add or restore:

```go
func ApplyClaudeCodeEffortMapping(body []byte, account *Account, requestModel string) ([]byte, bool) {
	if account == nil || account.Extra == nil || strings.TrimSpace(requestModel) == "" {
		return body, false
	}
	rawConfig, ok := account.Extra["claude_code_effort_mapping"].(map[string]any)
	if !ok {
		return body, false
	}
	modelConfig, ok := rawConfig[strings.TrimSpace(requestModel)].(map[string]any)
	if !ok {
		return body, false
	}
	targetField := claudeCodeEffortStringValue(modelConfig["target_field"])
	if !allowedClaudeCodeEffortTargetField(targetField) {
		return body, false
	}
	rawEffort := gjson.GetBytes(body, "output_config.effort").String()
	effort := NormalizeClaudeOutputEffort(rawEffort)
	if effort == nil {
		return body, false
	}
	values, ok := modelConfig["values"].(map[string]any)
	if !ok {
		return body, false
	}
	mapped := claudeCodeEffortStringValue(values[*effort])
	if mapped == "" {
		mapped = claudeCodeEffortStringValue(values["max"])
	}
	if mapped == "" {
		return body, false
	}
	if targetField == "thinking.budget_tokens" {
		budget, err := strconv.Atoi(mapped)
		if err != nil {
			return body, false
		}
		modified, err := sjson.SetBytes(body, targetField, budget)
		return modified, err == nil
	}
	modified, err := sjson.SetBytes(body, targetField, mapped)
	if err != nil {
		return body, false
	}
	return modified, true
}
```

Add imports if missing:

```go
import (
	"math"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)
```

- [ ] **Step 6: Apply catalog and effort mapping in Anthropic forwarding**

In the Anthropic request path in `backend/internal/service/gateway_service.go`, after the account has been selected and before the upstream HTTP request is created:

```go
requestModel := parsed.Model
upstreamModel := account.GetMappedModel(requestModel)
if isClaudeCodeClient(c) {
	if catalogModel, ok := account.ResolveClaudeCodeCatalogModel(requestModel); ok {
		upstreamModel = catalogModel
	}
	if mappedBody, changed := ApplyClaudeCodeEffortMapping(body, account, requestModel); changed {
		body = mappedBody
	}
}
```

Use the existing Claude Code detection helper in the file. If the current helper is named differently, use that exact helper and do not add a second detector.

- [ ] **Step 7: Expose catalog in `/v1/models` for Claude Code**

In `backend/internal/handler/gateway_handler.go`, update the models handler so that when the request is identified as Claude Code and the selected Anthropic account has catalog entries, it returns catalog `request_model` and `display_name`.

The response item shape must match existing model list shape. For each entry:

```go
map[string]any{
	"id":           entry.RequestModel,
	"display_name": entry.DisplayName,
	"type":         "model",
	"metadata": map[string]any{
		"supports_1m": entry.Supports1M,
	},
}
```

- [ ] **Step 8: Run backend mapping tests**

Run:

```powershell
cd backend
go test ./internal/service ./internal/handler -run "TestAccountClaudeCodeModelCatalogConfig|TestClaudeCodeEffortMappingMapsXHighToHigh|TestGatewayModelsUseClaudeCodeCatalogDisplayNames|TestGatewayService_AnthropicAPIKeyPassthrough_ClaudeCodeEffortMapping" -count=1
```

Expected: PASS.

- [ ] **Step 9: Commit checkpoint only after user approval**

If commit approval has been given:

```bash
git add backend/internal/service/account.go backend/internal/service/gateway_request.go backend/internal/service/gateway_request_test.go backend/internal/service/gateway_service.go backend/internal/service/gateway_anthropic_apikey_passthrough_test.go backend/internal/handler/gateway_handler.go backend/internal/handler/gateway_models_test.go
git commit -m "feat: restore claude code model mapping"
```

---

### Task 4: Add Compact Account List AT/RT Status

**Files:**
- Modify: `backend/internal/handler/dto/types.go`
- Modify: `backend/internal/handler/dto/mappers.go`
- Modify: `backend/internal/handler/dto/account_mapper_redact_test.go`
- Modify: `frontend/src/types/index.ts`
- Create: `frontend/src/components/account/AccountTokenStatusCell.vue`
- Create: `frontend/src/components/account/__tests__/AccountTokenStatusCell.spec.ts`
- Modify: `frontend/src/views/admin/AccountsView.vue`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`

- [ ] **Step 1: Write DTO test**

In `backend/internal/handler/dto/account_mapper_redact_test.go`, add:

```go
func TestAccountFromServiceShallow_BuildsTokenStatusWithoutSecrets(t *testing.T) {
	reasonUntil := time.Now().Add(10 * time.Minute)
	src := &service.Account{
		ID:       42,
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":  "at-secret",
			"refresh_token": "rt-secret",
			"expires_at":    time.Now().Add(30 * time.Minute).Unix(),
		},
		TempUnschedulableUntil:  &reasonUntil,
		TempUnschedulableReason: "token refresh retry exhausted: upstream timeout",
	}

	got := AccountFromServiceShallow(src)
	require.NotNil(t, got.TokenStatus)
	require.Equal(t, "present", got.TokenStatus.AccessToken)
	require.Equal(t, "present", got.TokenStatus.RefreshToken)
	require.Equal(t, "failed", got.TokenStatus.RefreshState)
	require.Contains(t, got.TokenStatus.Message, "upstream timeout")

	raw, err := json.Marshal(got)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "at-secret")
	require.NotContains(t, string(raw), "rt-secret")
}
```

- [ ] **Step 2: Run DTO test and confirm failure**

Run:

```powershell
cd backend
go test ./internal/handler/dto -run TestAccountFromServiceShallow_BuildsTokenStatusWithoutSecrets -count=1
```

Expected: FAIL because `TokenStatus` is missing.

- [ ] **Step 3: Add DTO type**

In `backend/internal/handler/dto/types.go`, add:

```go
type AccountTokenStatus struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	RefreshState string `json:"refresh_state"`
	Message      string `json:"message,omitempty"`
}
```

Add to `type Account`:

```go
TokenStatus *AccountTokenStatus `json:"token_status,omitempty"`
```

- [ ] **Step 4: Derive token status in mapper**

In `backend/internal/handler/dto/mappers.go`, add:

```go
func accountTokenStatusFromService(a *service.Account, credsStatus map[string]bool) *AccountTokenStatus {
	if a == nil || a.Type != service.AccountTypeOAuth {
		return nil
	}
	out := &AccountTokenStatus{
		AccessToken:  "missing",
		RefreshToken: "missing",
		RefreshState: "unknown",
	}
	if credsStatus["has_access_token"] {
		out.AccessToken = "present"
	}
	if credsStatus["has_refresh_token"] {
		out.RefreshToken = "present"
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(a.TempUnschedulableReason)), "token refresh retry exhausted") {
		out.RefreshState = "failed"
		out.Message = strings.TrimSpace(a.TempUnschedulableReason)
		return out
	}
	if strings.Contains(strings.ToLower(a.ErrorMessage), "token refresh failed") {
		out.RefreshState = "failed"
		out.Message = strings.TrimSpace(a.ErrorMessage)
		return out
	}
	if out.RefreshToken == "present" {
		out.RefreshState = "auto"
	} else {
		out.RefreshState = "manual"
	}
	return out
}
```

Add `strings` to imports.

In `AccountFromServiceShallow`, after `CredentialsStatus: credsStatus,` set:

```go
TokenStatus: accountTokenStatusFromService(a, credsStatus),
```

- [ ] **Step 5: Add frontend type**

In `frontend/src/types/index.ts`, add:

```ts
export interface AccountTokenStatus {
  access_token: 'present' | 'missing'
  refresh_token: 'present' | 'missing'
  refresh_state: 'auto' | 'manual' | 'failed' | 'unknown'
  message?: string
}
```

Add to `Account`:

```ts
token_status?: AccountTokenStatus | null
```

- [ ] **Step 6: Create compact cell component**

Create `frontend/src/components/account/AccountTokenStatusCell.vue`:

```vue
<template>
  <div class="flex min-w-[5.5rem] flex-col gap-0.5 text-[11px] leading-4" :title="title">
    <div class="flex items-center gap-1">
      <span :class="dotClass(accessOk)" />
      <span class="font-mono text-gray-700 dark:text-gray-200">AT {{ accessText }}</span>
    </div>
    <div class="flex items-center gap-1">
      <span :class="dotClass(refreshOk)" />
      <span class="font-mono text-gray-700 dark:text-gray-200">RT {{ refreshText }}</span>
    </div>
    <div :class="stateClass">{{ stateText }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Account } from '@/types'

const props = defineProps<{ account: Account }>()

const status = computed(() => props.account.token_status ?? null)
const accessOk = computed(() => status.value?.access_token === 'present')
const refreshOk = computed(() => status.value?.refresh_token === 'present')
const accessText = computed(() => accessOk.value ? '有' : '缺')
const refreshText = computed(() => refreshOk.value ? '有' : '缺')

const stateText = computed(() => {
  switch (status.value?.refresh_state) {
    case 'auto':
      return '自动'
    case 'manual':
      return '手动'
    case 'failed':
      return '失败'
    default:
      return '未知'
  }
})

const stateClass = computed(() => {
  if (status.value?.refresh_state === 'failed') return 'font-mono text-red-600 dark:text-red-300'
  if (status.value?.refresh_state === 'auto') return 'font-mono text-emerald-600 dark:text-emerald-300'
  return 'font-mono text-gray-500 dark:text-dark-400'
})

const title = computed(() => status.value?.message || stateText.value)

function dotClass(ok: boolean) {
  return [
    'h-1.5 w-1.5 rounded-full',
    ok ? 'bg-emerald-500' : 'bg-amber-500'
  ]
}
</script>
```

- [ ] **Step 7: Add cell tests**

Create `frontend/src/components/account/__tests__/AccountTokenStatusCell.spec.ts`:

```ts
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import AccountTokenStatusCell from '../AccountTokenStatusCell.vue'

const mountCell = (tokenStatus: any) => mount(AccountTokenStatusCell, {
  props: {
    account: {
      id: 1,
      name: 'acct',
      platform: 'openai',
      type: 'oauth',
      token_status: tokenStatus
    } as any
  }
})

describe('AccountTokenStatusCell', () => {
  it('renders automatic token refresh state compactly', () => {
    const wrapper = mountCell({ access_token: 'present', refresh_token: 'present', refresh_state: 'auto' })
    expect(wrapper.text()).toContain('AT 有')
    expect(wrapper.text()).toContain('RT 有')
    expect(wrapper.text()).toContain('自动')
  })

  it('renders failed refresh message in title', () => {
    const wrapper = mountCell({ access_token: 'present', refresh_token: 'present', refresh_state: 'failed', message: 'token refresh retry exhausted: timeout' })
    expect(wrapper.text()).toContain('失败')
    expect(wrapper.attributes('title')).toContain('timeout')
  })
})
```

- [ ] **Step 8: Wire column into AccountsView**

In `frontend/src/views/admin/AccountsView.vue`, import:

```ts
import AccountTokenStatusCell from '@/components/account/AccountTokenStatusCell.vue'
```

Add cell template near status / schedulable:

```vue
<template #cell-token_status="{ row }">
  <AccountTokenStatusCell :account="row" />
</template>
```

Add column after `schedulable`:

```ts
{ key: 'token_status', label: t('admin.accounts.columns.tokenStatus'), sortable: false },
```

- [ ] **Step 9: Add i18n labels**

In `zh.ts` under `admin.accounts.columns`:

```ts
tokenStatus: 'AT/RT',
```

In `en.ts` under `admin.accounts.columns`:

```ts
tokenStatus: 'AT/RT',
```

- [ ] **Step 10: Run tests**

Run:

```powershell
cd backend
go test ./internal/handler/dto -run "TestAccountFromServiceShallow_BuildsTokenStatusWithoutSecrets|TestAccountFromServiceShallow_RedactsSensitiveCredentials" -count=1

cd ..\frontend
pnpm vitest run src/components/account/__tests__/AccountTokenStatusCell.spec.ts
pnpm typecheck
```

Expected: PASS.

- [ ] **Step 11: Commit checkpoint only after user approval**

If commit approval has been given:

```bash
git add backend/internal/handler/dto/types.go backend/internal/handler/dto/mappers.go backend/internal/handler/dto/account_mapper_redact_test.go frontend/src/types/index.ts frontend/src/components/account/AccountTokenStatusCell.vue frontend/src/components/account/__tests__/AccountTokenStatusCell.spec.ts frontend/src/views/admin/AccountsView.vue frontend/src/i18n/locales/zh.ts frontend/src/i18n/locales/en.ts
git commit -m "feat: show account token refresh status"
```

---

### Task 5: Make Upstream Errors Visible Without Leaking Secrets

**Files:**
- Create: `backend/internal/service/upstream_error_summary.go`
- Create: `backend/internal/service/upstream_error_summary_test.go`
- Modify: `backend/internal/handler/gateway_handler.go`
- Modify: `backend/internal/handler/openai_gateway_handler.go`
- Modify: `backend/internal/handler/gateway_handler_responses.go`
- Modify: `backend/internal/handler/gateway_handler_chat_completions.go`

- [ ] **Step 1: Write summary tests**

Create `backend/internal/service/upstream_error_summary_test.go`:

```go
package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildFailoverExhaustedClientMessageIncludesUpstreamSummary(t *testing.T) {
	err := &UpstreamFailoverError{
		StatusCode: http.StatusForbidden,
		ResponseBody: []byte(`{"error":{"message":"model unavailable for relay account"}}`),
	}

	got := BuildFailoverExhaustedClientMessage("Service temporarily unavailable", err)
	require.Equal(t, "Service temporarily unavailable. Last upstream error: 403 model unavailable for relay account", got)
}

func TestBuildFailoverExhaustedClientMessageRedactsSecrets(t *testing.T) {
	err := &UpstreamFailoverError{
		StatusCode: http.StatusUnauthorized,
		ResponseBody: []byte(`{"error":{"message":"bad key sk-test-secret and refresh_token=rt-secret"}}`),
	}

	got := BuildFailoverExhaustedClientMessage("Service temporarily unavailable", err)
	require.Contains(t, got, "401")
	require.NotContains(t, got, "sk-test-secret")
	require.NotContains(t, got, "rt-secret")
}
```

- [ ] **Step 2: Run tests and confirm failure**

Run:

```powershell
cd backend
go test ./internal/service -run "TestBuildFailoverExhaustedClientMessage" -count=1
```

Expected: FAIL because helper is missing.

- [ ] **Step 3: Add a client-summary sanitizer**

In `backend/internal/service/upstream_error_summary.go`, keep the client-facing summary sanitizer local to this helper so the feature remains scoped to OpenAI/Codex and Claude failover exits:

```go
var (
	clientSummarySensitiveQueryParamRegex = regexp.MustCompile(`(?i)([?&](?:key|api_key|client_secret|access_token|refresh_token|id_token|token)=)[^&"\s]+`)
	clientSummarySensitivePairRegex       = regexp.MustCompile(`(?i)\b(authorization|api[_-]?key|access[_-]?token|refresh[_-]?token|id[_-]?token|client[_-]?secret|cookie)\s*[:=]\s*["']?[^"',\s}]+`)
	clientSummaryOpenAISecretRegex        = regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{8,}`)
)

func sanitizeClientUpstreamErrorSummary(msg string) string {
	if msg == "" {
		return msg
	}
	msg = clientSummarySensitiveQueryParamRegex.ReplaceAllString(msg, `$1***`)
	msg = clientSummarySensitivePairRegex.ReplaceAllString(msg, `$1=***`)
	msg = clientSummaryOpenAISecretRegex.ReplaceAllString(msg, `sk-***`)
	return msg
}
```

- [ ] **Step 4: Add summary helper**

Create `backend/internal/service/upstream_error_summary.go`:

```go
package service

import (
	"fmt"
	"regexp"
	"strings"
)

const upstreamErrorClientSummaryLimit = 220

func BuildFailoverExhaustedClientMessage(base string, failoverErr *UpstreamFailoverError) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "Service temporarily unavailable"
	}
	if failoverErr == nil {
		return base
	}

	parts := make([]string, 0, 2)
	if failoverErr.StatusCode > 0 {
		parts = append(parts, fmt.Sprintf("%d", failoverErr.StatusCode))
	}
	msg := strings.TrimSpace(ExtractUpstreamErrorMessage(failoverErr.ResponseBody))
	msg = sanitizeClientUpstreamErrorSummary(msg)
	if msg != "" {
		parts = append(parts, truncateString(msg, upstreamErrorClientSummaryLimit))
	}
	if len(parts) == 0 {
		return base
	}
	return base + ". Last upstream error: " + strings.Join(parts, " ")
}
```

- [ ] **Step 5: Use helper in failover-exhausted handlers**

In each failover-exhausted handler, after existing status mapping and before writing the final error, wrap the message.

For `backend/internal/handler/gateway_handler.go`:

```go
status, errType, errMsg := h.mapUpstreamError(statusCode)
errMsg = service.BuildFailoverExhaustedClientMessage(errMsg, failoverErr)
h.handleStreamingAwareError(c, status, errType, errMsg, streamStarted)
```

For `backend/internal/handler/openai_gateway_handler.go`:

```go
status, errType, errMsg := h.mapUpstreamError(statusCode)
errMsg = service.BuildFailoverExhaustedClientMessage(errMsg, failoverErr)
h.handleStreamingAwareError(c, status, errType, errMsg, streamStarted)
```

For `backend/internal/handler/gateway_handler_responses.go`:

```go
message := service.BuildFailoverExhaustedClientMessage("All available accounts exhausted", lastErr)
h.responsesErrorResponse(c, statusCode, "server_error", message)
```

For `backend/internal/handler/gateway_handler_chat_completions.go`:

```go
message := service.BuildFailoverExhaustedClientMessage("All available accounts exhausted", lastErr)
h.chatCompletionsErrorResponse(c, statusCode, "server_error", message)
```

- [ ] **Step 6: Run service and handler tests**

Run:

```powershell
cd backend
go test ./internal/service -run "TestBuildFailoverExhaustedClientMessage" -count=1
go test ./internal/handler -run "Test.*Failover.*|Test.*NoAccount.*" -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit checkpoint only after user approval**

If commit approval has been given:

```bash
git add backend/internal/service/upstream_error_summary.go backend/internal/service/upstream_error_summary_test.go backend/internal/handler/gateway_handler.go backend/internal/handler/openai_gateway_handler.go backend/internal/handler/gateway_handler_responses.go backend/internal/handler/gateway_handler_chat_completions.go
git commit -m "feat: surface sanitized upstream errors"
```

---

### Task 6: Verify Codex / Claude Code Client Compatibility

**Files:**
- Modify only if tests fail: `backend/internal/service/openai_gateway_service.go`
- Modify only if tests fail: `backend/internal/service/openai_ws_v2_passthrough_adapter.go`
- Modify only if tests fail: `backend/internal/service/openai_ws_forwarder.go`
- Test: existing `backend/internal/service/openai_oauth_passthrough_test.go`
- Test: existing `backend/internal/service/openai_gateway_service_codex_cli_only_test.go`

- [ ] **Step 1: Run existing compatibility tests**

Run:

```powershell
cd backend
go test ./internal/service -run "Test.*Codex.*|Test.*ClaudeCode.*|Test.*Passthrough.*|Test.*CliOnly.*" -count=1
```

Expected: PASS. If this fails, inspect the failing test and only adjust the specific OpenAI / WS passthrough path.

- [ ] **Step 2: Preserve session isolation**

If a path needs changes, keep this invariant:

```go
// Real client User-Agent and Originator may pass through for allowed Codex / Claude Code clients,
// but Session_ID and Conversation_ID must remain generated or isolated by sub2api.
```

Do not pass upstream-provided session identifiers across users.

- [ ] **Step 3: Add focused regression test if a compatibility gap is found**

If a failing route lacks coverage, add a test that asserts:

```go
require.Equal(t, "ClaudeCode/1.0", req.Header.Get("User-Agent"))
require.Equal(t, "claude_code", req.Header.Get("Originator"))
require.NotEqual(t, "client-session", req.Header.Get("Session_ID"))
require.NotEqual(t, "client-conversation", req.Header.Get("Conversation_ID"))
```

- [ ] **Step 4: Commit checkpoint only after user approval**

If code changed and commit approval has been given:

```bash
git add backend/internal/service/openai_gateway_service.go backend/internal/service/openai_ws_v2_passthrough_adapter.go backend/internal/service/openai_ws_forwarder.go backend/internal/service/openai_oauth_passthrough_test.go backend/internal/service/openai_gateway_service_codex_cli_only_test.go
git commit -m "test: preserve codex and claude code passthrough"
```

---

### Task 7: Full Local Verification

**Files:**
- No code files unless failures require focused fixes.

- [ ] **Step 1: Run Go tests for touched packages**

Run:

```powershell
cd backend
go test ./internal/service ./internal/handler ./internal/handler/dto -count=1
```

Expected: PASS.

- [ ] **Step 2: Run frontend tests**

Run:

```powershell
cd frontend
pnpm vitest run src/components/account/__tests__/AccountTestModal.spec.ts src/components/admin/account/__tests__/AccountTestModal.spec.ts src/components/account/__tests__/claudeCodeConfig.spec.ts src/components/account/__tests__/AccountTokenStatusCell.spec.ts
pnpm typecheck
pnpm build
```

Expected: PASS.

- [ ] **Step 3: Static whitespace check**

Run:

```powershell
git diff --check
```

Expected: no output.

- [ ] **Step 4: Record verification output**

Save the command names and pass/fail results into the final deployment stage document after deployment. Do not paste secrets, `.env`, token values, or database dumps.

---

### Task 8: Build, Backup, Deploy, and Roll Back Safely

**Files:**
- Modify after deployment: `D:\workSpace\codex\VPS\modules\sub2api\README.md`
- Create after deployment: `D:\workSpace\codex\VPS\stages\YYYY-MM-DD-sub2api-custom-features-r2.md`

- [ ] **Step 1: Create source branch**

Run from `D:\workSpace\codex\sub2api\frog-worktree`:

```powershell
git switch -c vps/custom-features-v0.1.146-r2
```

Expected: new branch created from current `vps/merge-v0.1.146-cc-mapping-r1`.

- [ ] **Step 2: Push branch after code is verified**

Run:

```powershell
git push -u morti vps/custom-features-v0.1.146-r2
```

Expected: branch is pushed to GitHub. Do not push `main` or `master`.

- [ ] **Step 3: Create a source tag after CI passes**

Use a new tag name:

```powershell
git tag vps-deploy/sub2api-MortiSSS-v0.1.146-r2
git push morti vps-deploy/sub2api-MortiSSS-v0.1.146-r2
```

Expected: tag exists on GitHub and points to the verified commit.

- [ ] **Step 4: Build GHCR image**

Use the existing GitHub Actions workflow for the custom image. Target tag:

```text
ghcr.io/morti4sss/sub2api:MortiSSS-v0.1.146-r2
```

Expected: GitHub Actions succeeds and pushes the image.

- [ ] **Step 5: Create VPS backup before deploy**

Run on VPS:

```bash
cd /home/mortisss/sub2api
stamp=$(date +%Y%m%d-%H%M%S)
backup_dir="/home/mortisss/sub2api/deploy/backup/${stamp}-before-MortiSSS-v0.1.146-r2"
mkdir -p "$backup_dir"
cp deploy/.env "$backup_dir/env.before"
cp deploy/docker-compose.local.yml "$backup_dir/docker-compose.local.yml.before"
cp deploy/docker-compose.vps-image.override.yml "$backup_dir/docker-compose.vps-image.override.yml.before"
cp deploy/data/config.yaml "$backup_dir/config.yaml.before"
docker inspect sub2api --format '{{.Config.Image}} {{.Image}}' > "$backup_dir/image.before.txt"
docker compose -f deploy/docker-compose.local.yml -f deploy/docker-compose.vps-image.override.yml ps > "$backup_dir/compose.ps.before.txt"
docker exec sub2api-postgres pg_dump -U sub2api sub2api -Fc > "$backup_dir/postgres-before.dump"
sha256sum "$backup_dir/postgres-before.dump" > "$backup_dir/postgres-before.dump.sha256"
echo "$backup_dir"
```

Expected: backup directory path printed. Do not upload dump or `.env` to GitHub.

- [ ] **Step 6: Deploy image**

Run on VPS:

```bash
cd /home/mortisss/sub2api
sed -i 's#^SUB2API_IMAGE=.*#SUB2API_IMAGE=ghcr.io/morti4sss/sub2api:MortiSSS-v0.1.146-r2#' deploy/.env
docker compose -f deploy/docker-compose.local.yml -f deploy/docker-compose.vps-image.override.yml pull sub2api
docker compose -f deploy/docker-compose.local.yml -f deploy/docker-compose.vps-image.override.yml up -d --force-recreate sub2api
```

Expected: only `sub2api` app container is recreated; PostgreSQL and Redis keep their data volumes.

- [ ] **Step 7: Verify deployment**

Run on VPS:

```bash
cd /home/mortisss/sub2api
docker inspect sub2api --format '{{.State.Health.Status}} {{.Config.Image}} {{.Image}}'
docker exec sub2api /app/sub2api --version
curl -fsS http://127.0.0.1:18080/health
curl -fsS https://sub2.mortisss.edu.kg/health
```

Expected:

```text
healthy ghcr.io/morti4sss/sub2api:MortiSSS-v0.1.146-r2 <image-digest>
{"status":"ok"}
{"status":"ok"}
```

- [ ] **Step 8: Functional smoke test**

From the admin UI:

- Test an OpenAI / Codex account with custom prompt `Respond only with SUB2API_OK`.
- Test an Anthropic / Claude Code relay account with custom prompt `Respond only with SUB2API_OK`.
- Open account list and confirm `AT/RT` compact status appears.
- Configure one Anthropic API Key relay with a Claude Code catalog entry and verify Claude Code `/model` shows the mapped display name.
- Trigger or inspect one known failing relay request and confirm the client response or ops detail includes a sanitized upstream status/message.

- [ ] **Step 9: Update VPS docs**

In `D:\workSpace\codex\VPS\modules\sub2api\README.md`, update:

- Current image.
- Current source branch and commit.
- GitHub Actions run URL.
- Image digest.
- Verification results.
- Backup directory.
- Rollback command.

Create `D:\workSpace\codex\VPS\stages\YYYY-MM-DD-sub2api-custom-features-r2.md` with:

- Stage goal.
- Actual changes.
- Involved modules.
- Key config changes.
- Verification results.
- Rollback method.
- Notes that account data was not uploaded to GitHub.

- [ ] **Step 10: Rollback command**

If deployment is unhealthy, run:

```bash
cd /home/mortisss/sub2api
sed -i 's#^SUB2API_IMAGE=.*#SUB2API_IMAGE=ghcr.io/morti4sss/sub2api:MortiSSS-v0.1.146-r1#' deploy/.env
docker compose -f deploy/docker-compose.local.yml -f deploy/docker-compose.vps-image.override.yml up -d --force-recreate sub2api
curl -fsS http://127.0.0.1:18080/health
```

Expected: health returns `{"status":"ok"}`. Restore database only after explicit confirmation because it overwrites data created after the backup.

---

## Self-Review

- Spec coverage: test prompt, Claude Code model mapping, effort mapping, AT/RT account-list status, upstream error visibility, compatibility passthrough, backup, deployment, and rollback all have tasks.
- Upstream-sync coverage: `CUSTOMIZATIONS.md` records the custom ledger, config keys, verification commands, branch/tag model, and sensitive-data boundaries for future official updates.
- Open-ended-work scan: no task uses deferred implementation language. Provider presets are explicitly excluded.
- Type consistency: frontend catalog uses `requestModel`, backend and persisted JSON use `request_model`; effort mapping uses `claude_code_effort_mapping`; token status uses `token_status`.
- Safety: all token-facing UI and DTO work uses redaction or presence booleans only. Deployment backup commands keep dumps and `.env` on VPS/local backup, not GitHub.
