// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
