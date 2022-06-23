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

package entities

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"code.vegaprotocol.io/protos/vega"
	"github.com/shopspring/decimal"
)

type MarginLevels struct {
	AccountID              int64
	MaintenanceMargin      decimal.Decimal
	SearchLevel            decimal.Decimal
	InitialMargin          decimal.Decimal
	CollateralReleaseLevel decimal.Decimal
	Timestamp              time.Time
	VegaTime               time.Time
}

func MarginLevelsFromProto(ctx context.Context, margin *vega.MarginLevels, accountSource AccountSource, vegaTime time.Time) (MarginLevels, error) {
	var (
		maintenanceMargin, searchLevel, initialMargin, collateralReleaseLevel decimal.Decimal
		err                                                                   error
	)

	marginAccount, err := GetAccountFromMarginLevel(ctx, margin, accountSource, vegaTime)
	if err != nil {
		return MarginLevels{}, fmt.Errorf("failed to obtain accour for margin level: %w", err)
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
		VegaTime:               vegaTime,
	}, nil
}

func GetAccountFromMarginLevel(ctx context.Context, margin *vega.MarginLevels, accountSource AccountSource, vegaTime time.Time) (Account, error) {
	marginAccount := Account{
		ID:       0,
		PartyID:  NewPartyID(margin.PartyId),
		AssetID:  NewAssetID(margin.Asset),
		MarketID: NewMarketID(margin.MarketId),
		Type:     vega.AccountType_ACCOUNT_TYPE_MARGIN,
		VegaTime: vegaTime,
	}

	err := accountSource.Obtain(ctx, &marginAccount)
	return marginAccount, err
}

func (ml *MarginLevels) ToProto(accountSource AccountSource) (*vega.MarginLevels, error) {
	marginAccount, err := accountSource.GetByID(ml.AccountID)
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

func (ml MarginLevels) ToProtoEdge(input ...any) *v2.MarginEdge {
	if len(input) == 0 {
		return nil
	}

	switch as := input[0].(type) {
	case AccountSource:
		mlProto, err := ml.ToProto(as)
		if err != nil {
			return nil
		}
		return &v2.MarginEdge{
			Node:   mlProto,
			Cursor: ml.Cursor().Encode(),
		}
	default:
		return nil
	}
}

type MarginLevelsKey struct {
	AccountID int64
	VegaTime  time.Time
}

func (o MarginLevels) Key() MarginLevelsKey {
	return MarginLevelsKey{o.AccountID, o.VegaTime}
}

func (ml MarginLevels) ToRow() []interface{} {
	return []interface{}{
		ml.AccountID, ml.Timestamp, ml.MaintenanceMargin,
		ml.SearchLevel, ml.InitialMargin, ml.CollateralReleaseLevel, ml.VegaTime,
	}
}

var MarginLevelsColumns = []string{
	"account_id", "timestamp", "maintenance_margin",
	"search_level", "initial_margin", "collateral_release_level", "vega_time",
}

type MarginCursor struct {
	VegaTime  time.Time
	AccountID int64
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
