package service

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	upstreamErrorClientSummaryLimit = 220
	UpstreamErrorCode               = "upstream_error"
	UpstreamModelNotFoundErrorCode  = "upstream_model_not_found"
)

var (
	clientSummarySensitiveQueryParamRegex = regexp.MustCompile(`(?i)([?&](?:key|api_key|client_secret|access_token|refresh_token|id_token|token)=)[^&"\s]+`)
	clientSummaryAuthorizationSchemeRegex = regexp.MustCompile(`(?i)\b(authorization)\s*[:=]\s*["']?(?:bearer|basic|token)\s+[^"',\s}]+`)
	clientSummarySensitivePairRegex       = regexp.MustCompile(`(?i)\b(authorization|api[_-]?key|access[_-]?token|refresh[_-]?token|id[_-]?token|client[_-]?secret|cookie|token)\s*[:=]\s*["']?[^"',\s}]+`)
	clientSummaryOpenAISecretRegex        = regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{8,}`)
)

// UpstreamFailureDetails is the safe client-facing description of one
// upstream failure. Reason never contains known credential forms.
type UpstreamFailureDetails struct {
	Code       string
	StatusCode int
	Reason     string
}

// DescribeUpstreamFailoverError is the client-facing owner for upstream error
// classification and redaction used by gateway responses and account tests.
func DescribeUpstreamFailoverError(failoverErr *UpstreamFailoverError) UpstreamFailureDetails {
	if failoverErr == nil {
		return UpstreamFailureDetails{Code: UpstreamErrorCode}
	}
	return DescribeUpstreamError(failoverErr.StatusCode, failoverErr.ResponseBody)
}

// DescribeUpstreamError classifies an upstream status/body pair and returns a
// redacted reason suitable for CLI responses and admin diagnostics.
func DescribeUpstreamError(statusCode int, responseBody []byte) UpstreamFailureDetails {
	details := UpstreamFailureDetails{Code: UpstreamErrorCode, StatusCode: statusCode}
	if isUpstreamModelNotFoundError(statusCode, responseBody) {
		details.Code = UpstreamModelNotFoundErrorCode
	}
	details.Reason = strings.TrimSpace(ExtractUpstreamErrorMessage(responseBody))
	details.Reason = sanitizeClientUpstreamErrorSummary(details.Reason)
	details.Reason = truncateString(details.Reason, upstreamErrorClientSummaryLimit)
	return details
}

func BuildFailoverExhaustedClientMessage(base string, failoverErr *UpstreamFailoverError) string {
	if failoverErr == nil {
		return buildUpstreamFailureClientMessage(base, UpstreamFailureDetails{Code: UpstreamErrorCode})
	}
	return BuildUpstreamErrorClientMessage(base, failoverErr.StatusCode, failoverErr.ResponseBody)
}

// BuildUpstreamErrorClientMessage combines a stable client message with the
// real upstream status and redacted reason.
func BuildUpstreamErrorClientMessage(base string, statusCode int, responseBody []byte) string {
	return buildUpstreamFailureClientMessage(base, DescribeUpstreamError(statusCode, responseBody))
}

func buildUpstreamFailureClientMessage(base string, details UpstreamFailureDetails) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "Service temporarily unavailable"
	}

	parts := make([]string, 0, 2)
	if details.StatusCode > 0 {
		parts = append(parts, fmt.Sprintf("%d", details.StatusCode))
	}
	if details.Reason != "" {
		parts = append(parts, details.Reason)
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
	msg = clientSummaryAuthorizationSchemeRegex.ReplaceAllString(msg, `$1=***`)
	msg = clientSummarySensitivePairRegex.ReplaceAllString(msg, `$1=***`)
	msg = clientSummaryOpenAISecretRegex.ReplaceAllString(msg, `sk-***`)
	return msg
}
