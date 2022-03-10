package entities

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"code.vegaprotocol.io/protos/vega"

	"github.com/holiman/uint256"

	"github.com/shopspring/decimal"
)

type Trade struct {
	VegaTime                time.Time
	SeqNum                  uint64
	ID                      []byte
	MarketID                []byte
	Price                   decimal.Decimal
	Size                    decimal.Decimal
	Buyer                   []byte
	Seller                  []byte
	Aggressor               Side
	BuyOrder                []byte
	SellOrder               []byte
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
		Id:        strings.ToUpper(hex.EncodeToString(t.ID)),
		MarketId:  hex.EncodeToString(t.MarketID),
		Price:     t.Price.String(),
		Size:      t.Size.UintNO().Uint64(),
		Buyer:     Party{ID: t.Buyer}.HexID(),
		Seller:    Party{ID: t.Seller}.HexID(),
		Aggressor: t.Aggressor,
		BuyOrder:  strings.ToUpper(hex.EncodeToString(t.BuyOrder)),
		SellOrder: strings.ToUpper(hex.EncodeToString(t.SellOrder)),
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
	id, err := hex.DecodeString(t.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to decode trade id:%w", err)
	}
	marketId, err := hex.DecodeString(t.MarketId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode market id:%w", err)
	}

	price, err := decimal.NewFromString(t.Price)
	if err != nil {
		return nil, fmt.Errorf("failed to decode price:%w", err)
	}

	size := decimal.NewFromUint(uint256.NewInt(t.Size))

	buyer, err := MakePartyID(t.Buyer)
	if err != nil {
		return nil, fmt.Errorf("failed to decode buyer id:%w", err)
	}

	seller, err := MakePartyID(t.Seller)
	if err != nil {
		return nil, fmt.Errorf("failed to decode seller id:%w", err)
	}

	buyOrderId, err := MakeOrderID(t.BuyOrder)
	if err != nil {
		return nil, fmt.Errorf("failed to decode buy order id:%w", err)
	}

	sellOrderId, err := MakeOrderID(t.SellOrder)
	if err != nil {
		return nil, fmt.Errorf("failed to decode sell order id:%w", err)
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
		VegaTime:                vegaTime,
		SeqNum:                  sequenceNumber,
		ID:                      id,
		MarketID:                marketId,
		Price:                   price,
		Size:                    size,
		Buyer:                   buyer,
		Seller:                  seller,
		Aggressor:               t.Aggressor,
		BuyOrder:                buyOrderId,
		SellOrder:               sellOrderId,
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
