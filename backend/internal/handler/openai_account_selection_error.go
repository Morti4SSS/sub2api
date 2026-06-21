package handler

import (
	"fmt"
	"strings"
)

func noAvailableOpenAIAccountsMessage(groupID *int64, model string, err error) string {
	model = strings.TrimSpace(model)
	errText := ""
	if err != nil {
		errText = strings.TrimSpace(err.Error())
	}
	if errText == "" {
		errText = "no available accounts"
	}
	groupText := "ungrouped"
	if groupID != nil {
		groupText = fmt.Sprintf("%d", *groupID)
	}
	if model == "" {
		return fmt.Sprintf("No available OpenAI accounts for group %s: %s", groupText, errText)
	}
	return fmt.Sprintf("No available OpenAI accounts for group %s and model %s: %s", groupText, model, errText)
}
