package entities

import (
	"context"
	"fmt"
	"time"

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

	marginAccount := Account{
		ID:       0,
		PartyID:  NewPartyID(margin.PartyId),
		AssetID:  NewAssetID(margin.Asset),
		MarketID: NewMarketID(margin.MarketId),
		Type:     vega.AccountType_ACCOUNT_TYPE_MARGIN,
		VegaTime: vegaTime,
	}

	err = accountSource.Obtain(ctx, &marginAccount)
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
		Timestamp:              time.Unix(0, margin.Timestamp),
		VegaTime:               vegaTime,
	}, nil
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
		PartyId:                marginAccount.String(),
		MarketId:               marginAccount.String(),
		Asset:                  marginAccount.String(),
		Timestamp:              ml.Timestamp.UnixNano(),
	}, nil
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
