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

package entities

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/shopspring/decimal"
)

type MarginLevels struct {
	AccountID              AccountID
	OrderMarginAccountID   AccountID
	MaintenanceMargin      decimal.Decimal
	SearchLevel            decimal.Decimal
	InitialMargin          decimal.Decimal
	CollateralReleaseLevel decimal.Decimal
	OrderMargin            decimal.Decimal
	Timestamp              time.Time
	TxHash                 TxHash
	VegaTime               time.Time
	MarginMode             MarginMode
	MarginFactor           decimal.Decimal
}

func MarginLevelsFromProto(ctx context.Context, margin *vega.MarginLevels, accountSource AccountSource, txHash TxHash, vegaTime time.Time) (MarginLevels, error) {
	var (
		maintenanceMargin, searchLevel, initialMargin, collateralReleaseLevel, orderMargin decimal.Decimal
		err                                                                                error
	)
	marginFactor := decimal.NewFromInt32(0)
	if len(margin.MarginFactor) > 0 {
		marginFactor, err = decimal.NewFromString(margin.MarginFactor)
		if err != nil {
			return MarginLevels{}, fmt.Errorf("failed to obtain margin factor for margin level: %w", err)
		}
	}
	marginMode := MarginModeCrossMargin
	if margin.MarginMode == vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN {
		marginMode = MarginMode(margin.MarginMode)
	}

	marginAccount, err := GetAccountFromMarginLevel(ctx, margin, accountSource, txHash, vegaTime)
	if err != nil {
		return MarginLevels{}, fmt.Errorf("failed to obtain account for margin level: %w", err)
	}

	orderMarginAccount, err := GetAccountFromOrderMarginLevel(ctx, margin, accountSource, txHash, vegaTime)
	var orderMarginAccountID AccountID
	if margin.MarginMode == vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN && err != nil {
		return MarginLevels{}, fmt.Errorf("failed to obtain account for order margin level: %w", err)
	} else {
		orderMarginAccountID = orderMarginAccount.ID
	}

	if maintenanceMargin, err = decimal.NewFromString(margin.MaintenanceMargin); err != nil {
		return MarginLevels{}, fmt.Errorf("invalid maintenance margin: %w", err)
	}

	if searchLevel, err = decimal.NewFromString(margin.SearchLevel); err != nil {
		return MarginLevels{}, fmt.Errorf("invalid search level: %w", err)
	}

	if initialMargin, err = decimal.NewFromString(margin.InitialMargin); err != nil {
		return MarginLevels{}, fmt.Errorf("invalid initial margin: %w", err)
	}

	if collateralReleaseLevel, err = decimal.NewFromString(margin.CollateralReleaseLevel); err != nil {
		return MarginLevels{}, fmt.Errorf("invalid collateralReleaseLevel: %w", err)
	}

	if len(margin.OrderMargin) == 0 {
		orderMargin = decimal.NewFromInt32(0)
	} else if orderMargin, err = decimal.NewFromString(margin.OrderMargin); err != nil {
		return MarginLevels{}, fmt.Errorf("invalid orderMarginLevel: %w", err)
	}

	return MarginLevels{
		AccountID:              marginAccount.ID,
		OrderMarginAccountID:   orderMarginAccountID,
		MaintenanceMargin:      maintenanceMargin,
		SearchLevel:            searchLevel,
		InitialMargin:          initialMargin,
		CollateralReleaseLevel: collateralReleaseLevel,
		OrderMargin:            orderMargin,
		Timestamp:              time.Unix(0, vegaTime.UnixNano()),
		TxHash:                 txHash,
		VegaTime:               vegaTime,
		MarginMode:             marginMode,
		MarginFactor:           marginFactor,
	}, nil
}

func GetAccountFromMarginLevel(ctx context.Context, margin *vega.MarginLevels, accountSource AccountSource, txHash TxHash, vegaTime time.Time) (Account, error) {
	marginAccount := Account{
		ID:       "",
		PartyID:  PartyID(margin.PartyId),
		AssetID:  AssetID(margin.Asset),
		MarketID: MarketID(margin.MarketId),
		Type:     vega.AccountType_ACCOUNT_TYPE_MARGIN,
		TxHash:   txHash,
		VegaTime: vegaTime,
	}

	err := accountSource.Obtain(ctx, &marginAccount)
	return marginAccount, err
}

func GetAccountFromOrderMarginLevel(ctx context.Context, margin *vega.MarginLevels, accountSource AccountSource, txHash TxHash, vegaTime time.Time) (Account, error) {
	orderMarginAccount := Account{
		ID:       "",
		PartyID:  PartyID(margin.PartyId),
		AssetID:  AssetID(margin.Asset),
		MarketID: MarketID(margin.MarketId),
		Type:     vega.AccountType_ACCOUNT_TYPE_ORDER_MARGIN,
		TxHash:   txHash,
		VegaTime: vegaTime,
	}

	err := accountSource.Obtain(ctx, &orderMarginAccount)
	return orderMarginAccount, err
}

func (ml *MarginLevels) ToProto(ctx context.Context, accountSource AccountSource) (*vega.MarginLevels, error) {
	marginAccount, err := accountSource.GetByID(ctx, ml.AccountID)
	if err != nil {
		return nil, fmt.Errorf("getting from account for transfer proto:%w", err)
	}

	return &vega.MarginLevels{
		MaintenanceMargin:      ml.MaintenanceMargin.String(),
		SearchLevel:            ml.SearchLevel.String(),
		InitialMargin:          ml.InitialMargin.String(),
		CollateralReleaseLevel: ml.CollateralReleaseLevel.String(),
		OrderMargin:            ml.OrderMargin.String(),
		PartyId:                marginAccount.PartyID.String(),
		MarketId:               marginAccount.MarketID.String(),
		Asset:                  marginAccount.AssetID.String(),
		Timestamp:              ml.Timestamp.UnixNano(),
		MarginMode:             vega.MarginMode(ml.MarginMode),
		MarginFactor:           ml.MarginFactor.String(),
	}, nil
}

func (ml MarginLevels) Cursor() *Cursor {
	cursor := MarginCursor{
		VegaTime:  ml.VegaTime,
		AccountID: ml.AccountID,
	}
	return NewCursor(cursor.String())
}

func (ml MarginLevels) ToProtoEdge(input ...any) (*v2.MarginEdge, error) {
	if len(input) != 2 {
		return nil, fmt.Errorf("expected account source and context argument")
	}

	ctx, ok := input[0].(context.Context)
	if !ok {
		return nil, fmt.Errorf("first argument must be a context.Context, got: %v", input[0])
	}

	as, ok := input[1].(AccountSource)
	if !ok {
		return nil, fmt.Errorf("second argument must be an AccountSource, got: %v", input[1])
	}

	mlProto, err := ml.ToProto(ctx, as)
	if err != nil {
		return nil, err
	}

	return &v2.MarginEdge{
		Node:   mlProto,
		Cursor: ml.Cursor().Encode(),
	}, nil
}

type MarginLevelsKey struct {
	AccountID AccountID
	VegaTime  time.Time
}

func (ml MarginLevels) Key() MarginLevelsKey {
	return MarginLevelsKey{ml.AccountID, ml.VegaTime}
}

func (ml MarginLevels) ToRow() []interface{} {
	return []interface{}{
		ml.AccountID, ml.OrderMarginAccountID, ml.Timestamp, ml.MaintenanceMargin,
		ml.SearchLevel, ml.InitialMargin, ml.CollateralReleaseLevel, ml.OrderMargin, ml.TxHash, ml.VegaTime,
		ml.MarginMode, ml.MarginFactor,
	}
}

var MarginLevelsColumns = []string{
	"account_id", "order_margin_account_id", "timestamp", "maintenance_margin",
	"search_level", "initial_margin", "collateral_release_level", "order_margin", "tx_hash",
	"vega_time", "margin_mode", "margin_factor",
}

type MarginCursor struct {
	VegaTime  time.Time
	AccountID AccountID
}

func (mc MarginCursor) String() string {
	bs, err := json.Marshal(mc)
	if err != nil {
		// This should never happen
		panic(fmt.Errorf("failed to marshal margin cursor: %w", err))
	}
	return string(bs)
}

func (mc *MarginCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), mc)
}
