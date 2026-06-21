package handler

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNoAvailableOpenAIAccountsMessage(t *testing.T) {
	groupID := int64(19)

	msg := noAvailableOpenAIAccountsMessage(&groupID, "gpt-5.5", errors.New("no available accounts"))

	require.Equal(t, "No available OpenAI accounts for group 19 and model gpt-5.5: no available accounts", msg)
}

func TestNoAvailableOpenAIAccountsMessageDefaults(t *testing.T) {
	msg := noAvailableOpenAIAccountsMessage(nil, "", nil)

	require.Equal(t, "No available OpenAI accounts for group ungrouped: no available accounts", msg)
}
