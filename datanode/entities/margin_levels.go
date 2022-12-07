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
	MaintenanceMargin      decimal.Decimal
	SearchLevel            decimal.Decimal
	InitialMargin          decimal.Decimal
	CollateralReleaseLevel decimal.Decimal
	Timestamp              time.Time
	TxHash                 TxHash
	VegaTime               time.Time
}

func MarginLevelsFromProto(ctx context.Context, margin *vega.MarginLevels, accountSource AccountSource, txHash TxHash, vegaTime time.Time) (MarginLevels, error) {
	var (
		maintenanceMargin, searchLevel, initialMargin, collateralReleaseLevel decimal.Decimal
		err                                                                   error
	)

	marginAccount, err := GetAccountFromMarginLevel(ctx, margin, accountSource, txHash, vegaTime)
	if err != nil {
		return MarginLevels{}, fmt.Errorf("failed to obtain account for margin level: %w", err)
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

	return MarginLevels{
		AccountID:              marginAccount.ID,
		MaintenanceMargin:      maintenanceMargin,
		SearchLevel:            searchLevel,
		InitialMargin:          initialMargin,
		CollateralReleaseLevel: collateralReleaseLevel,
		Timestamp:              time.Unix(0, vegaTime.UnixNano()),
		TxHash:                 txHash,
		VegaTime:               vegaTime,
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
		PartyId:                marginAccount.PartyID.String(),
		MarketId:               marginAccount.MarketID.String(),
		Asset:                  marginAccount.AssetID.String(),
		Timestamp:              ml.Timestamp.UnixNano(),
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
		ml.AccountID, ml.Timestamp, ml.MaintenanceMargin,
		ml.SearchLevel, ml.InitialMargin, ml.CollateralReleaseLevel, ml.TxHash, ml.VegaTime,
	}
}

var MarginLevelsColumns = []string{
	"account_id", "timestamp", "maintenance_margin",
	"search_level", "initial_margin", "collateral_release_level", "tx_hash", "vega_time",
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
