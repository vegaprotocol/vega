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
