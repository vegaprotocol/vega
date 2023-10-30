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
	"context"
	"time"

	"code.vegaprotocol.io/vega/wallet/wallet"
)

// Generates mocks
//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/wallet/service/v2/connections WalletStore,TimeService,TokenStore,SessionStore

type TimeService interface {
	Now() time.Time
}

type WalletStore interface {
	WalletExists(ctx context.Context, name string) (bool, error)
	UnlockWallet(ctx context.Context, name, passphrase string) error
	IsWalletAlreadyUnlocked(ctx context.Context, name string) (bool, error)
	GetWallet(ctx context.Context, name string) (wallet.Wallet, error)
	OnUpdate(callbackFn func(context.Context, wallet.Event))
}

// TokenStore is the component used to retrieve and update the API tokens from the
// computer.
type TokenStore interface {
	TokenExists(Token) (bool, error)
	ListTokens() ([]TokenSummary, error)
	DescribeToken(Token) (TokenDescription, error)
	SaveToken(TokenDescription) error
	DeleteToken(Token) error
	OnUpdate(callbackFn func(ctx context.Context, tokens ...TokenDescription))
}

type SessionStore interface {
	ListSessions(context.Context) ([]Session, error)
	DeleteSession(context.Context, Token) error
	TrackSession(session Session) error
}
