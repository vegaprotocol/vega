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
	"time"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
)

type Token string

func (t Token) String() string {
	return string(t)
}

func (t Token) Short() string {
	if len(t) > 0 {
		return string([]byte(t)[:4]) + ".." + string([]byte(t)[len(t)-5:])
	}
	return ""
}

func GenerateToken() Token {
	return Token(vgrand.RandomStr(64))
}

func AsToken(token string) (Token, error) {
	if len(token) == 0 {
		return "", ErrTokenIsRequired
	}
	if len(token) != 64 {
		return "", ErrInvalidTokenFormat
	}
	return Token(token), nil
}

type TokenSummary struct {
	Description    string     `json:"description"`
	Token          Token      `json:"token"`
	CreationDate   time.Time  `json:"creationDate"`
	ExpirationDate *time.Time `json:"expirationDate"`
}

type WalletCredentials struct {
	Name       string `json:"name"`
	Passphrase string `json:"passphrase"`
}

type Session struct {
	Token    Token  `json:"token"`
	Hostname string `json:"hostname"`
	Wallet   string `json:"wallet"`
}
