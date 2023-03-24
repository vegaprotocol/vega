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

package entities

import (
	"encoding/json"
	"fmt"
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/shopspring/decimal"
)

type _Trade struct{}

type TradeID = ID[_Trade]

type Trade struct {
	SyntheticTime           time.Time
	TxHash                  TxHash
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

func (t Trade) ToProto() *vega.Trade {
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

func (t Trade) Cursor() *Cursor {
	return NewCursor(TradeCursor{SyntheticTime: t.SyntheticTime}.String())
}

func (t Trade) ToProtoEdge(_ ...any) (*v2.TradeEdge, error) {
	return &v2.TradeEdge{
		Node:   t.ToProto(),
		Cursor: t.Cursor().Encode(),
	}, nil
}

func TradeFromProto(t *vega.Trade, txHash TxHash, vegaTime time.Time, sequenceNumber uint64) (*Trade, error) {
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
		TxHash:                  txHash,
		VegaTime:                vegaTime,
		SeqNum:                  sequenceNumber,
		ID:                      TradeID(t.Id),
		MarketID:                MarketID(t.MarketId),
		Price:                   price,
		Size:                    t.Size,
		Buyer:                   PartyID(t.Buyer),
		Seller:                  PartyID(t.Seller),
		Aggressor:               t.Aggressor,
		BuyOrder:                OrderID(t.BuyOrder),
		SellOrder:               OrderID(t.SellOrder),
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

type TradeCursor struct {
	SyntheticTime time.Time `json:"synthetic_time"`
}

func (c TradeCursor) String() string {
	bs, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Errorf("could not marshal Trade cursor: %w", err))
	}
	return string(bs)
}

func (c *TradeCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), c)
}
