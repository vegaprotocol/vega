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

package sqlsubscribers

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/protos/vega"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestEventDeduplicator_Flush(t *testing.T) {
	edd := NewEventDeduplicator[string, *vega.LiquidityProvision](
		func(ctx context.Context, lp *vega.LiquidityProvision) string { return lp.Id },
		func(lp1 *vega.LiquidityProvision, lp2 *vega.LiquidityProvision) bool { return proto.Equal(lp1, lp2) },
	)

	lp1 := &vega.LiquidityProvision{
		Id: "1",
	}

	edd.AddEvent(context.Background(), lp1)
	events := edd.Flush()
	assert.Equal(t, lp1, events["1"])

	lp2 := &vega.LiquidityProvision{
		Id:     "1",
		Status: vega.LiquidityProvision_STATUS_PENDING,
	}

	edd.AddEvent(context.Background(), lp2)
	events = edd.Flush()
	assert.Equal(t, lp2, events["1"])

	edd.AddEvent(context.Background(), lp2)
	events = edd.Flush()
	assert.Equal(t, 0, len(events))

	edd.AddEvent(context.Background(), lp2)
	edd.AddEvent(context.Background(), lp1)
	edd.AddEvent(context.Background(), lp2)
	events = edd.Flush()
	assert.Equal(t, 0, len(events))

	edd.AddEvent(context.Background(), lp1)
	edd.AddEvent(context.Background(), lp2)
	edd.AddEvent(context.Background(), lp1)
	events = edd.Flush()
	assert.Equal(t, lp1, events["1"])
}
