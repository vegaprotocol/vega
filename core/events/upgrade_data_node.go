// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package events

import (
	"context"

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type UpgradeDataNode struct {
	*Base
}

func NewUpgradeDataNode(ctx context.Context) *UpgradeDataNode {
	return &UpgradeDataNode{
		Base: newBase(ctx, UpgradeDataNodeEvent),
	}
}

func (a UpgradeDataNode) StreamMessage() *eventspb.BusEvent {
	return newBusEventFromBase(a.Base)
}

func UpgradeDataNodeEventFromStream(ctx context.Context, be *eventspb.BusEvent) *UpgradeDataNode {
	return &UpgradeDataNode{
		Base: newBaseFromBusEvent(ctx, UpgradeDataNodeEvent, be),
	}
}
