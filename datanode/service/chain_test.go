// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/service"
	"code.vegaprotocol.io/data-node/datanode/service/mocks"
	"code.vegaprotocol.io/data-node/logging"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestChainService(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()
	defer ctrl.Finish()
	store := mocks.NewMockChainStore(ctrl)
	svc := service.NewChain(store, logging.NewTestLogger())

	t.Run("fetching unset chain", func(t *testing.T) {
		// Should not be cached so expect another call to the store
		store.EXPECT().Get(ctx).Return(entities.Chain{}, entities.ErrChainNotFound).Times(2)
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
