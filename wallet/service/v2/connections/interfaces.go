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
