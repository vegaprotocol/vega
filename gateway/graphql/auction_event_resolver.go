package gql

import (
	"context"

	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"
)

type auctionEventResolver VegaResolverRoot

func (r *auctionEventResolver) AuctionStart(ctx context.Context, obj *proto.AuctionEvent) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Start)), nil

}
func (r *auctionEventResolver) AuctionEnd(ctx context.Context, obj *proto.AuctionEvent) (string, error) {
	if obj.End > 0 {
		return vegatime.Format(vegatime.UnixNano(obj.End)), nil
	}
	return "", nil
}

func (r *auctionEventResolver) Trigger(ctx context.Context, obj *proto.AuctionEvent) (AuctionTrigger, error) {
	return convertAuctionTriggerFromProto(obj.Trigger)
}
