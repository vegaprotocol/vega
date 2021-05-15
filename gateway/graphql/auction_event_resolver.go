package gql

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
	"code.vegaprotocol.io/vega/vegatime"
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
