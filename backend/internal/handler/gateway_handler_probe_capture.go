package handler

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const (
	anthropicDesktopProbeCaptureModel           = "claude-opus-4-8"
	anthropicDesktopProbeExactText              = "."
	anthropicDesktopProbeCaptureObservedMaxToks = 64
	anthropicDesktopProbeCapturePreviewMaxRunes = 120
)

type anthropicDesktopProbeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicDesktopProbeMessage struct {
	Role    string                          `json:"role"`
	Content anthropicDesktopProbeTextSource `json:"content"`
}

type anthropicDesktopProbeTextSource struct {
	Text string
}

func (s *anthropicDesktopProbeTextSource) UnmarshalJSON(data []byte) error {
	if s == nil {
		return nil
	}

	var directText string
	if err := json.Unmarshal(data, &directText); err == nil {
		s.Text = strings.TrimSpace(directText)
		return nil
	}

	var blocks []anthropicDesktopProbeContentBlock
	if err := json.Unmarshal(data, &blocks); err != nil {
		return err
	}
	s.Text = extractAnthropicDesktopProbeFirstText(blocks)
	return nil
}

type anthropicDesktopProbeCapture struct {
	Observed      bool
	ExactMatch    bool
	Model         string
	MaxTokens     int
	MessagesCount int
	FirstRole     string
	FirstText     string
	FirstTextLen  int
}

func inspectAnthropicDesktopProbeCapture(parsed *service.ParsedRequest) anthropicDesktopProbeCapture {
	info := anthropicDesktopProbeCapture{}
	if parsed == nil {
		return info
	}

	info.Model = strings.TrimSpace(parsed.Model)
	if !strings.EqualFold(info.Model, anthropicDesktopProbeCaptureModel) {
		return info
	}

	var messages []anthropicDesktopProbeMessage
	if err := parsed.DecodeMessages(&messages); err != nil {
		return info
	}
	info.MaxTokens = parsed.MaxTokens
	info.MessagesCount = len(messages)
	if len(messages) == 0 {
		return info
	}

	info.FirstRole = strings.TrimSpace(messages[0].Role)
	info.FirstText = strings.TrimSpace(messages[0].Content.Text)
	info.FirstTextLen = utf8.RuneCountInString(info.FirstText)
	if len(messages) == 1 && info.FirstTextLen > 0 && info.MaxTokens > 0 && info.MaxTokens <= anthropicDesktopProbeCaptureObservedMaxToks {
		info.Observed = true
	}
	if len(messages) == 1 &&
		strings.EqualFold(info.FirstRole, "user") &&
		info.MaxTokens == 1 &&
		info.FirstText == anthropicDesktopProbeExactText {
		info.ExactMatch = true
	}
	return info
}

func extractAnthropicDesktopProbeFirstText(content []anthropicDesktopProbeContentBlock) string {
	for _, block := range content {
		if strings.EqualFold(strings.TrimSpace(block.Type), "text") {
			return strings.TrimSpace(block.Text)
		}
	}
	return ""
}

func anthropicDesktopProbeCapturePreview(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= anthropicDesktopProbeCapturePreviewMaxRunes {
		return text
	}
	return string(runes[:anthropicDesktopProbeCapturePreviewMaxRunes]) + "..."
}

func shouldShortCircuitAnthropicDesktopProbe(info anthropicDesktopProbeCapture, isStream bool) bool {
	return info.ExactMatch && !isStream
}
