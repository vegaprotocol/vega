package session

import (
	"time"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
)

type TokenSummary struct {
	Description string    `json:"description"`
	Token       string    `json:"token"`
	CreateAt    time.Time `json:"createAt"`
	Expiry      *int64    `json:"expiry"`
}

type Token struct {
	Description string            `json:"description"`
	Expiry      *int64            `json:"expiry"`
	Token       string            `json:"token"`
	Wallet      WalletCredentials `json:"wallet"`
}

type WalletCredentials struct {
	Name       string `json:"name"`
	Passphrase string `json:"passphrase"`
}

func GenerateToken() string {
	return vgrand.RandomStr(64)
}
