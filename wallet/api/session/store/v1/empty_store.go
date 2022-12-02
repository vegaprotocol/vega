package v1

import (
	"errors"

	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/session"
)

var ErrSavingTokenIsDisabled = errors.New("saving tokens is disabled")

// EmptyStore can be used to disable the support for long-living API tokens.
type EmptyStore struct{}

func (e EmptyStore) TokenExists(_ string) (bool, error) {
	return false, nil
}

func (e EmptyStore) ListTokens() ([]session.TokenSummary, error) {
	return []session.TokenSummary{}, nil
}

func (e EmptyStore) GetToken(_ string) (session.Token, error) {
	return session.Token{}, api.ErrTokenDoesNotExist
}

func (e EmptyStore) SaveToken(_ session.Token) error {
	return ErrSavingTokenIsDisabled
}

func (e EmptyStore) DeleteToken(_ string) error {
	return api.ErrTokenDoesNotExist
}

func NewEmptyStore() *EmptyStore {
	return &EmptyStore{}
}
