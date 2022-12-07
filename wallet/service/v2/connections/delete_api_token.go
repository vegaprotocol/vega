package connections

import (
	"fmt"
)

func DeleteAPIToken(tokenStore TokenStore, rawToken string) error {
	token, err := AsToken(rawToken)
	if err != nil {
		return fmt.Errorf("the token is not valid: %w", err)
	}

	if exist, err := tokenStore.TokenExists(token); err != nil {
		return fmt.Errorf("could not verify the token existence: %w", err)
	} else if !exist {
		return ErrTokenDoesNotExist
	}

	if err := tokenStore.DeleteToken(token); err != nil {
		return fmt.Errorf("could not delete the token: %w", err)
	}

	return nil
}
