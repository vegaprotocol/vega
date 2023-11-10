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

package sqlsubscribers_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/sqlsubscribers"
	"code.vegaprotocol.io/vega/datanode/sqlsubscribers/mocks"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	v1 "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestNotary_Push(t *testing.T) {
	ctrl := gomock.NewController(t)

	store := mocks.NewMockNotaryStore(ctrl)

	store.EXPECT().Add(context.Background(), gomock.Any()).Times(1)
	subscriber := sqlsubscribers.NewNotary(store)
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

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	store := mocks.NewMockNotaryStore(ctrl)
	subscriber := sqlsubscribers.NewNotary(store)
	subscriber.Push(context.Background(), events.NewOracleDataEvent(context.Background(), vegapb.OracleData{}))
}
