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

// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/datanode/service/mocks"
)

func TestChainService(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	store := mocks.NewMockChainStore(ctrl)
	svc := service.NewChain(store)

	t.Run("fetching unset chain", func(t *testing.T) {
		// Should not be cached so expect another call to the store
		store.EXPECT().Get(ctx).Return(entities.Chain{}, entities.ErrNotFound).Times(2)
		for i := 0; i < 2; i++ {
			chainID, err := svc.GetChainID()
			assert.NoError(t, err)
			assert.Equal(t, "", chainID)
		}
	})

	t.Run("error when fetching chain", func(t *testing.T) {
		ourError := errors.New("oops")
		// should not be cached so expect another call to the store
		store.EXPECT().Get(ctx).Return(entities.Chain{}, ourError).Times(2)
		for i := 0; i < 2; i++ {
			chainID, err := svc.GetChainID()
			assert.ErrorIs(t, err, ourError)
			assert.Equal(t, "", chainID)
		}
	})

	t.Run("fetching already set chain", func(t *testing.T) {
		// *should* be cached so do not expect another call to the store
		store.EXPECT().Get(ctx).Return(entities.Chain{ID: "my-test-chain"}, nil)
		for i := 0; i < 2; i++ {
			chainID, err := svc.GetChainID()
			assert.NoError(t, err)
			assert.Equal(t, "my-test-chain", chainID)
		}
	})

	t.Run("setting chain", func(t *testing.T) {
		store.EXPECT().Set(ctx, entities.Chain{ID: "my-test-chain"}).Return(nil)
		err := svc.SetChainID("my-test-chain")
		assert.NoError(t, err)
	})
}
