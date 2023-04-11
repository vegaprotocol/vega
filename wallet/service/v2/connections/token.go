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
