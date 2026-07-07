package service

import (
	"fmt"
	"regexp"
	"strings"
)

const upstreamErrorClientSummaryLimit = 220

var (
	clientSummarySensitiveQueryParamRegex = regexp.MustCompile(`(?i)([?&](?:key|api_key|client_secret|access_token|refresh_token|id_token|token)=)[^&"\s]+`)
	clientSummarySensitivePairRegex       = regexp.MustCompile(`(?i)\b(authorization|api[_-]?key|access[_-]?token|refresh[_-]?token|id[_-]?token|client[_-]?secret|cookie)\s*[:=]\s*["']?[^"',\s}]+`)
	clientSummaryOpenAISecretRegex        = regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{8,}`)
)

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

func sanitizeClientUpstreamErrorSummary(msg string) string {
	if msg == "" {
		return msg
	}
	msg = clientSummarySensitiveQueryParamRegex.ReplaceAllString(msg, `$1***`)
	msg = clientSummarySensitivePairRegex.ReplaceAllString(msg, `$1=***`)
	msg = clientSummaryOpenAISecretRegex.ReplaceAllString(msg, `sk-***`)
	return msg
}
