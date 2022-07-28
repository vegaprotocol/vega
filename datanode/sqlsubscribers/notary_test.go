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

package sqlsubscribers_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/data-node/datanode/sqlsubscribers"
	"code.vegaprotocol.io/data-node/datanode/sqlsubscribers/mocks"
	"code.vegaprotocol.io/data-node/logging"
	v1 "code.vegaprotocol.io/protos/vega/commands/v1"
	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/events"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestNotary_Push(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockNotaryStore(ctrl)

	store.EXPECT().Add(context.Background(), gomock.Any()).Times(1)
	subscriber := sqlsubscribers.NewNotary(store, logging.NewTestLogger())
	err := subscriber.Push(context.Background(),
		events.NewNodeSignatureEvent(context.Background(),
			v1.NodeSignature{
				Id:   "someid",
				Sig:  []byte("somesig"),
				Kind: v1.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_WITHDRAWAL,
			},
		),
	)
	require.NoError(t, err)
}

func TestNotary_PushWrongEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	store := mocks.NewMockNotaryStore(ctrl)
	subscriber := sqlsubscribers.NewNotary(store, logging.NewTestLogger())
	subscriber.Push(context.Background(), events.NewOracleDataEvent(context.Background(), oraclespb.OracleData{}))
}
