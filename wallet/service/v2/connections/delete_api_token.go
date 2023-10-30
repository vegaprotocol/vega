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
