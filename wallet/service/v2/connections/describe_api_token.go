package connections

import (
	"fmt"
	"time"
)

type TokenDescription struct {
	Description    string            `json:"description"`
	CreationDate   time.Time         `json:"creationDate"`
	ExpirationDate *time.Time        `json:"expirationDate"`
	Token          Token             `json:"token"`
	Wallet         WalletCredentials `json:"wallet"`
}

func DescribeAPIToken(tokenStore TokenStore, rawToken string) (TokenDescription, error) {
	token, err := AsToken(rawToken)
	if err != nil {
		return TokenDescription{}, fmt.Errorf("the token is not valid: %w", err)
	}

	tokenDescription, err := tokenStore.DescribeToken(token)
	if err != nil {
		return TokenDescription{}, fmt.Errorf("could not retrieve the token information: %w", err)
	}

	return tokenDescription, nil
}
