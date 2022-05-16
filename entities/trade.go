package entities

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/protos/vega"

	"github.com/shopspring/decimal"
)

type TradeID struct{ ID }

func NewTradeID(id string) TradeID {
	return TradeID{ID: ID(id)}
}

type Trade struct {
	SyntheticTime           time.Time
	VegaTime                time.Time
	SeqNum                  uint64
	ID                      TradeID
	MarketID                MarketID
	Price                   decimal.Decimal
	Size                    uint64
	Buyer                   PartyID
	Seller                  PartyID
	Aggressor               Side
	BuyOrder                OrderID
	SellOrder               OrderID
	Type                    TradeType
	BuyerMakerFee           decimal.Decimal
	BuyerInfrastructureFee  decimal.Decimal
	BuyerLiquidityFee       decimal.Decimal
	SellerMakerFee          decimal.Decimal
	SellerInfrastructureFee decimal.Decimal
	SellerLiquidityFee      decimal.Decimal
	BuyerAuctionBatch       uint64
	SellerAuctionBatch      uint64
}

func (t *Trade) ToProto() *vega.Trade {
	return &vega.Trade{
		Id:        t.ID.String(),
		MarketId:  t.MarketID.String(),
		Price:     t.Price.String(),
		Size:      t.Size,
		Buyer:     t.Buyer.String(),
		Seller:    t.Seller.String(),
		Aggressor: t.Aggressor,
		BuyOrder:  t.BuyOrder.String(),
		SellOrder: t.SellOrder.String(),
		Timestamp: t.VegaTime.UnixNano(),
		Type:      t.Type,
		BuyerFee: &vega.Fee{
			MakerFee:          t.BuyerMakerFee.String(),
			InfrastructureFee: t.BuyerInfrastructureFee.String(),
			LiquidityFee:      t.BuyerLiquidityFee.String(),
		},
		SellerFee: &vega.Fee{
			MakerFee:          t.SellerMakerFee.String(),
			InfrastructureFee: t.SellerInfrastructureFee.String(),
			LiquidityFee:      t.SellerLiquidityFee.String(),
		},
		BuyerAuctionBatch:  t.BuyerAuctionBatch,
		SellerAuctionBatch: t.SellerAuctionBatch,
	}
}

func TradeFromProto(t *vega.Trade, vegaTime time.Time, sequenceNumber uint64) (*Trade, error) {
	syntheticTime := vegaTime.Add(time.Duration(sequenceNumber) * time.Microsecond)

	price, err := decimal.NewFromString(t.Price)
	if err != nil {
		return nil, fmt.Errorf("failed to decode price:%w", err)
	}

	buyerMakerFee := decimal.Zero
	buyerInfraFee := decimal.Zero
	buyerLiquidityFee := decimal.Zero
	if t.BuyerFee != nil {
		buyerMakerFee, err = decimal.NewFromString(t.BuyerFee.MakerFee)
		if err != nil {
			return nil, fmt.Errorf("failed to decode buyer maker fee:%w", err)
		}

		buyerInfraFee, err = decimal.NewFromString(t.BuyerFee.InfrastructureFee)
		if err != nil {
			return nil, fmt.Errorf("failed to decode buyer infrastructure fee:%w", err)
		}

		buyerLiquidityFee, err = decimal.NewFromString(t.BuyerFee.LiquidityFee)
		if err != nil {
			return nil, fmt.Errorf("failed to decode buyer liquidity fee:%w", err)
		}
	}

	sellerMakerFee := decimal.Zero
	sellerInfraFee := decimal.Zero
	sellerLiquidityFee := decimal.Zero
	if t.SellerFee != nil {
		sellerMakerFee, err = decimal.NewFromString(t.SellerFee.MakerFee)
		if err != nil {
			return nil, fmt.Errorf("failed to decode seller maker fee:%w", err)
		}

		sellerInfraFee, err = decimal.NewFromString(t.SellerFee.InfrastructureFee)
		if err != nil {
			return nil, fmt.Errorf("failed to decode buyer infrastructure fee:%w", err)
		}

		sellerLiquidityFee, err = decimal.NewFromString(t.SellerFee.LiquidityFee)
		if err != nil {
			return nil, fmt.Errorf("failed to decode seller liquidity fee:%w", err)
		}
	}

	trade := Trade{
		SyntheticTime:           syntheticTime,
		VegaTime:                vegaTime,
		SeqNum:                  sequenceNumber,
		ID:                      NewTradeID(t.Id),
		MarketID:                NewMarketID(t.MarketId),
		Price:                   price,
		Size:                    t.Size,
		Buyer:                   NewPartyID(t.Buyer),
		Seller:                  NewPartyID(t.Seller),
		Aggressor:               t.Aggressor,
		BuyOrder:                NewOrderID(t.BuyOrder),
		SellOrder:               NewOrderID(t.SellOrder),
		Type:                    t.Type,
		BuyerMakerFee:           buyerMakerFee,
		BuyerInfrastructureFee:  buyerInfraFee,
		BuyerLiquidityFee:       buyerLiquidityFee,
		SellerMakerFee:          sellerMakerFee,
		SellerInfrastructureFee: sellerInfraFee,
		SellerLiquidityFee:      sellerLiquidityFee,
		BuyerAuctionBatch:       t.BuyerAuctionBatch,
		SellerAuctionBatch:      t.SellerAuctionBatch,
	}
	return &trade, nil
}
