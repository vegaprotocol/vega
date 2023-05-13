package v1

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/wallet/service/v2/connections"
)

var ErrSavingTokenIsDisabled = errors.New("saving tokens is disabled")

// EmptyStore can be used to disable the support for long-living API tokens.
type EmptyStore struct{}

func (e EmptyStore) TokenExists(_ connections.Token) (bool, error) {
	return false, nil
}

func (e EmptyStore) ListTokens() ([]connections.TokenSummary, error) {
	return []connections.TokenSummary{}, nil
}

func (e EmptyStore) DescribeToken(_ connections.Token) (connections.TokenDescription, error) {
	return connections.TokenDescription{}, ErrTokenDoesNotExist
}

func (e EmptyStore) SaveToken(_ connections.TokenDescription) error {
	return ErrSavingTokenIsDisabled
}

func (e EmptyStore) DeleteToken(_ connections.Token) error {
	return ErrTokenDoesNotExist
}

func (e EmptyStore) OnUpdate(_ func(context.Context, ...connections.TokenDescription)) {}

func (e EmptyStore) Close() {}

func NewEmptyStore() *EmptyStore {
	return &EmptyStore{}
}
