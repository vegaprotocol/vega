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

	BuyerMakerFeeReferralDiscount           decimal.Decimal
	BuyerMakerFeeVolumeDiscount             decimal.Decimal
	BuyerInfrastructureFeeReferralDiscount  decimal.Decimal
	BuyerInfrastructureFeeVolumeDiscount    decimal.Decimal
	BuyerLiquidityFeeReferralDiscount       decimal.Decimal
	BuyerLiquidityFeeVolumeDiscount         decimal.Decimal
	SellerMakerFeeReferralDiscount          decimal.Decimal
	SellerMakerFeeVolumeDiscount            decimal.Decimal
	SellerInfrastructureFeeReferralDiscount decimal.Decimal
	SellerInfrastructureFeeVolumeDiscount   decimal.Decimal
	SellerLiquidityFeeReferralDiscount      decimal.Decimal
	SellerLiquidityFeeVolumeDiscount        decimal.Decimal

	BuyerAuctionBatch  uint64
	SellerAuctionBatch uint64
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
			MakerFee:                          t.BuyerMakerFee.String(),
			InfrastructureFee:                 t.BuyerInfrastructureFee.String(),
			LiquidityFee:                      t.BuyerLiquidityFee.String(),
			MakerFeeReferrerDiscount:          t.BuyerMakerFeeReferralDiscount.String(),
			MakerFeeVolumeDiscount:            t.BuyerMakerFeeVolumeDiscount.String(),
			InfrastructureFeeReferrerDiscount: t.BuyerInfrastructureFeeReferralDiscount.String(),
			InfrastructureFeeVolumeDiscount:   t.BuyerInfrastructureFeeVolumeDiscount.String(),
			LiquidityFeeReferrerDiscount:      t.BuyerLiquidityFeeReferralDiscount.String(),
			LiquidityFeeVolumeDiscount:        t.BuyerLiquidityFeeVolumeDiscount.String(),
		},
		SellerFee: &vega.Fee{
			MakerFee:                          t.SellerMakerFee.String(),
			InfrastructureFee:                 t.SellerInfrastructureFee.String(),
			LiquidityFee:                      t.SellerLiquidityFee.String(),
			MakerFeeReferrerDiscount:          t.SellerMakerFeeReferralDiscount.String(),
			MakerFeeVolumeDiscount:            t.SellerMakerFeeVolumeDiscount.String(),
			InfrastructureFeeReferrerDiscount: t.SellerInfrastructureFeeReferralDiscount.String(),
			InfrastructureFeeVolumeDiscount:   t.SellerInfrastructureFeeVolumeDiscount.String(),
			LiquidityFeeReferrerDiscount:      t.SellerLiquidityFeeReferralDiscount.String(),
			LiquidityFeeVolumeDiscount:        t.SellerLiquidityFeeVolumeDiscount.String(),
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

	buyerMakerFeeReferrerDiscount := decimal.Zero
	buyerMakerFeeVolumeDiscount := decimal.Zero
	buyerInfraFeeReferrerDiscount := decimal.Zero
	buyerInfraFeeVolumeDiscount := decimal.Zero
	buyerLiquidityFeeReferrerDiscount := decimal.Zero
	buyerLiquidityFeeVolumeDiscount := decimal.Zero

	if t.BuyerFee != nil {
		buyerMakerFee, err = decimal.NewFromString(t.BuyerFee.MakerFee)
		if err != nil {
			return nil, fmt.Errorf("failed to decode buyer maker fee:%w", err)
		}
		if len(t.BuyerFee.MakerFeeReferrerDiscount) > 0 {
			buyerMakerFeeReferrerDiscount, err = decimal.NewFromString(t.BuyerFee.MakerFeeReferrerDiscount)
			if err != nil {
				return nil, fmt.Errorf("failed to decode buyer maker fee referrer discount:%w", err)
			}
		}
		if len(t.BuyerFee.MakerFeeVolumeDiscount) > 0 {
			buyerMakerFeeVolumeDiscount, err = decimal.NewFromString(t.BuyerFee.MakerFeeVolumeDiscount)
			if err != nil {
				return nil, fmt.Errorf("failed to decode buyer maker fee volume discount:%w", err)
			}
		}
		buyerInfraFee, err = decimal.NewFromString(t.BuyerFee.InfrastructureFee)
		if err != nil {
			return nil, fmt.Errorf("failed to decode buyer infrastructure fee:%w", err)
		}
		if len(t.BuyerFee.InfrastructureFeeReferrerDiscount) > 0 {
			buyerInfraFeeReferrerDiscount, err = decimal.NewFromString(t.BuyerFee.InfrastructureFeeReferrerDiscount)
			if err != nil {
				return nil, fmt.Errorf("failed to decode buyer infrastructure fee referrer discount:%w", err)
			}
		}
		if len(t.BuyerFee.InfrastructureFeeVolumeDiscount) > 0 {
			buyerInfraFeeVolumeDiscount, err = decimal.NewFromString(t.BuyerFee.InfrastructureFeeVolumeDiscount)
			if err != nil {
				return nil, fmt.Errorf("failed to decode buyer infrastructure fee volume discount:%w", err)
			}
		}
		buyerLiquidityFee, err = decimal.NewFromString(t.BuyerFee.LiquidityFee)
		if err != nil {
			return nil, fmt.Errorf("failed to decode buyer liquidity fee:%w", err)
		}
		if len(t.BuyerFee.LiquidityFeeReferrerDiscount) > 0 {
			buyerLiquidityFeeReferrerDiscount, err = decimal.NewFromString(t.BuyerFee.LiquidityFeeReferrerDiscount)
			if err != nil {
				return nil, fmt.Errorf("failed to decode buyer liquidity fee referrer discount:%w", err)
			}
		}
		if len(t.BuyerFee.LiquidityFeeVolumeDiscount) > 0 {
			buyerLiquidityFeeVolumeDiscount, err = decimal.NewFromString(t.BuyerFee.LiquidityFeeVolumeDiscount)
			if err != nil {
				return nil, fmt.Errorf("failed to decode buyer liquidity fee volume discount:%w", err)
			}
		}
	}

	sellerMakerFee := decimal.Zero
	sellerInfraFee := decimal.Zero
	sellerLiquidityFee := decimal.Zero

	sellerMakerFeeReferrerDiscount := decimal.Zero
	sellerMakerFeeVolumeDiscount := decimal.Zero
	sellerInfraFeeReferrerDiscount := decimal.Zero
	sellerInfraFeeVolumeDiscount := decimal.Zero
	sellerLiquidityFeeReferrerDiscount := decimal.Zero
	sellerLiquidityFeeVolumeDiscount := decimal.Zero

	if t.SellerFee != nil {
		sellerMakerFee, err = decimal.NewFromString(t.SellerFee.MakerFee)
		if err != nil {
			return nil, fmt.Errorf("failed to decode seller maker fee:%w", err)
		}
		if len(t.SellerFee.MakerFeeReferrerDiscount) > 0 {
			sellerMakerFeeReferrerDiscount, err = decimal.NewFromString(t.SellerFee.MakerFeeReferrerDiscount)
			if err != nil {
				return nil, fmt.Errorf("failed to decode seller maker fee referrer discount:%w", err)
			}
		}
		if len(t.SellerFee.MakerFeeVolumeDiscount) > 0 {
			sellerMakerFeeVolumeDiscount, err = decimal.NewFromString(t.SellerFee.MakerFeeVolumeDiscount)
			if err != nil {
				return nil, fmt.Errorf("failed to decode seller maker fee volume discount:%w", err)
			}
		}
		sellerInfraFee, err = decimal.NewFromString(t.SellerFee.InfrastructureFee)
		if err != nil {
			return nil, fmt.Errorf("failed to decode seller infrastructure fee:%w", err)
		}
		if len(t.SellerFee.InfrastructureFeeReferrerDiscount) > 0 {
			sellerInfraFeeReferrerDiscount, err = decimal.NewFromString(t.SellerFee.InfrastructureFeeReferrerDiscount)
			if err != nil {
				return nil, fmt.Errorf("failed to decode seller infrastructure fee referrer discount:%w", err)
			}
		}
		if len(t.SellerFee.InfrastructureFeeVolumeDiscount) > 0 {
			sellerInfraFeeVolumeDiscount, err = decimal.NewFromString(t.SellerFee.InfrastructureFeeVolumeDiscount)
			if err != nil {
				return nil, fmt.Errorf("failed to decode seller infrastructure fee volume discount:%w", err)
			}
		}
		sellerLiquidityFee, err = decimal.NewFromString(t.SellerFee.LiquidityFee)
		if err != nil {
			return nil, fmt.Errorf("failed to decode seller liquidity fee:%w", err)
		}
		if len(t.SellerFee.LiquidityFeeReferrerDiscount) > 0 {
			sellerLiquidityFeeReferrerDiscount, err = decimal.NewFromString(t.SellerFee.LiquidityFeeReferrerDiscount)
			if err != nil {
				return nil, fmt.Errorf("failed to decode seller liquidity fee referrer discount:%w", err)
			}
		}
		if len(t.SellerFee.LiquidityFeeVolumeDiscount) > 0 {
			sellerLiquidityFeeVolumeDiscount, err = decimal.NewFromString(t.SellerFee.LiquidityFeeVolumeDiscount)
			if err != nil {
				return nil, fmt.Errorf("failed to decode seller liquidity fee volume discount:%w", err)
			}
		}
	}

	trade := Trade{
		SyntheticTime:                           syntheticTime,
		TxHash:                                  txHash,
		VegaTime:                                vegaTime,
		SeqNum:                                  sequenceNumber,
		ID:                                      TradeID(t.Id),
		MarketID:                                MarketID(t.MarketId),
		Price:                                   price,
		Size:                                    t.Size,
		Buyer:                                   PartyID(t.Buyer),
		Seller:                                  PartyID(t.Seller),
		Aggressor:                               t.Aggressor,
		BuyOrder:                                OrderID(t.BuyOrder),
		SellOrder:                               OrderID(t.SellOrder),
		Type:                                    t.Type,
		BuyerMakerFee:                           buyerMakerFee,
		BuyerInfrastructureFee:                  buyerInfraFee,
		BuyerLiquidityFee:                       buyerLiquidityFee,
		BuyerMakerFeeReferralDiscount:           buyerMakerFeeReferrerDiscount,
		BuyerMakerFeeVolumeDiscount:             buyerMakerFeeVolumeDiscount,
		BuyerInfrastructureFeeReferralDiscount:  buyerInfraFeeReferrerDiscount,
		BuyerInfrastructureFeeVolumeDiscount:    buyerInfraFeeVolumeDiscount,
		BuyerLiquidityFeeReferralDiscount:       buyerLiquidityFeeReferrerDiscount,
		BuyerLiquidityFeeVolumeDiscount:         buyerLiquidityFeeVolumeDiscount,
		SellerMakerFee:                          sellerMakerFee,
		SellerInfrastructureFee:                 sellerInfraFee,
		SellerLiquidityFee:                      sellerLiquidityFee,
		SellerMakerFeeReferralDiscount:          sellerMakerFeeReferrerDiscount,
		SellerMakerFeeVolumeDiscount:            sellerMakerFeeVolumeDiscount,
		SellerInfrastructureFeeReferralDiscount: sellerInfraFeeReferrerDiscount,
		SellerInfrastructureFeeVolumeDiscount:   sellerInfraFeeVolumeDiscount,
		SellerLiquidityFeeReferralDiscount:      sellerLiquidityFeeReferrerDiscount,
		SellerLiquidityFeeVolumeDiscount:        sellerLiquidityFeeVolumeDiscount,
		BuyerAuctionBatch:                       t.BuyerAuctionBatch,
		SellerAuctionBatch:                      t.SellerAuctionBatch,
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
