package connections

import (
	"fmt"
)

type ListAPITokensResult struct {
	Tokens []TokenSummary `json:"tokens"`
}

func ListAPITokens(tokenStore TokenStore) (ListAPITokensResult, error) {
	tokens, err := tokenStore.ListTokens()
	if err != nil {
		return ListAPITokensResult{}, fmt.Errorf("could not list the tokens: %w", err)
	}

	return ListAPITokensResult{
		Tokens: tokens,
	}, nil
}
