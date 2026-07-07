package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type claudeCodeCatalogRepoStub struct {
	AccountRepository

	all     []Account
	byGroup map[int64][]Account
}

func (s *claudeCodeCatalogRepoStub) ListSchedulable(ctx context.Context) ([]Account, error) {
	out := make([]Account, len(s.all))
	copy(out, s.all)
	return out, nil
}

func (s *claudeCodeCatalogRepoStub) ListSchedulableByGroupID(ctx context.Context, groupID int64) ([]Account, error) {
	accounts := s.byGroup[groupID]
	out := make([]Account, len(accounts))
	copy(out, accounts)
	return out, nil
}

func TestAccountClaudeCodeModelCatalogResolvesUpstreamModel(t *testing.T) {
	account := &Account{
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

	catalog := account.GetClaudeCodeModelCatalog()
	require.Len(t, catalog, 1)
	require.Equal(t, "Relay Sonnet", catalog[0].DisplayName)
	require.Equal(t, []string{"thinking", "effort"}, catalog[0].Capabilities)
	require.True(t, catalog[0].Supports1M)

	upstream, matched := account.ResolveClaudeCodeCatalogModel("relay-sonnet")
	require.True(t, matched)
	require.Equal(t, "upstream-sonnet", upstream)
}

func TestGatewayServiceGetClaudeCodeModelCatalogDeduplicatesByRequestModel(t *testing.T) {
	groupID := int64(7)
	repo := &claudeCodeCatalogRepoStub{
		byGroup: map[int64][]Account{
			groupID: {
				{
					ID:       1,
					Platform: PlatformAnthropic,
					Extra: map[string]any{
						"claude_code_model_catalog": []any{
							map[string]any{
								"role":          "sonnet",
								"display_name":  "Relay Sonnet A",
								"request_model": "relay-sonnet",
							},
						},
					},
				},
				{
					ID:       2,
					Platform: PlatformAnthropic,
					Extra: map[string]any{
						"claude_code_model_catalog": []any{
							map[string]any{
								"role":          "sonnet",
								"display_name":  "Relay Sonnet B",
								"request_model": "relay-sonnet",
							},
						},
					},
				},
				{
					ID:       3,
					Platform: PlatformOpenAI,
					Extra: map[string]any{
						"claude_code_model_catalog": []any{
							map[string]any{
								"role":          "openai",
								"display_name":  "Wrong Platform",
								"request_model": "wrong-platform",
							},
						},
					},
				},
			},
		},
	}
	svc := &GatewayService{accountRepo: repo}

	catalog := svc.GetClaudeCodeModelCatalog(context.Background(), &groupID, PlatformAnthropic)
	require.Len(t, catalog, 1)
	require.Equal(t, "relay-sonnet", catalog[0].RequestModel)
	require.Equal(t, "Relay Sonnet A", catalog[0].DisplayName)
}

func TestApplyClaudeCodeEffortMappingMapsExplicitEffort(t *testing.T) {
	account := &Account{
		Extra: map[string]any{
			"claude_code_effort_mapping": map[string]any{
				"relay-sonnet": map[string]any{
					"target_field": "reasoning_effort",
					"values": map[string]any{
						"low":    "low",
						"medium": "high",
						"high":   "max",
						"xhigh":  "max",
						"max":    "max",
					},
				},
			},
		},
	}
	body := []byte(`{"model":"relay-sonnet","output_config":{"effort":"xhigh"}}`)

	rewritten, changed := ApplyClaudeCodeEffortMapping(body, account, "relay-sonnet")
	require.True(t, changed)
	require.Equal(t, "max", gjson.GetBytes(rewritten, "reasoning_effort").String())
}

func TestApplyClaudeCodeEffortMappingDefaultsToStrongestConfiguredLevel(t *testing.T) {
	account := &Account{
		Extra: map[string]any{
			"claude_code_effort_mapping": map[string]any{
				"relay-sonnet": map[string]any{
					"target_field": "thinking.budget_tokens",
					"values": map[string]any{
						"max": "32000",
					},
				},
			},
		},
	}
	body := []byte(`{"model":"relay-sonnet","messages":[]}`)

	rewritten, changed := ApplyClaudeCodeEffortMapping(body, account, "relay-sonnet")
	require.True(t, changed)
	require.Equal(t, int64(32000), gjson.GetBytes(rewritten, "thinking.budget_tokens").Int())
	require.Equal(t, "enabled", gjson.GetBytes(rewritten, "thinking.type").String())
}
