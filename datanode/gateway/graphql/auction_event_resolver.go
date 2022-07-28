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

package gql

import (
	"context"

	"code.vegaprotocol.io/data-node/datanode/vegatime"
	types "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

type auctionEventResolver VegaResolverRoot

func (r *auctionEventResolver) AuctionStart(ctx context.Context, obj *eventspb.AuctionEvent) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Start)), nil

}
func (r *auctionEventResolver) AuctionEnd(ctx context.Context, obj *eventspb.AuctionEvent) (string, error) {
	if obj.End > 0 {
		return vegatime.Format(vegatime.UnixNano(obj.End)), nil
	}
	return "", nil
}

func (r *auctionEventResolver) Trigger(ctx context.Context, obj *eventspb.AuctionEvent) (AuctionTrigger, error) {
	return convertAuctionTriggerFromProto(obj.Trigger)
}

func (r *auctionEventResolver) ExtensionTrigger(ctx context.Context, obj *eventspb.AuctionEvent) (*AuctionTrigger, error) {
	if obj.ExtensionTrigger == types.AuctionTrigger_AUCTION_TRIGGER_UNSPECIFIED {
		return nil, nil
	}
	t, err := convertAuctionTriggerFromProto(obj.ExtensionTrigger)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
