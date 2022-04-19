package entities

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/protos/vega"
	"github.com/shopspring/decimal"
)

type MarginLevels struct {
	MarketID               MarketID
	AssetID                AssetID
	PartyID                PartyID
	MaintenanceMargin      decimal.Decimal
	SearchLevel            decimal.Decimal
	InitialMargin          decimal.Decimal
	CollateralReleaseLevel decimal.Decimal
	Timestamp              time.Time
	VegaTime               time.Time
}

func MarginLevelsFromProto(margin *vega.MarginLevels, vegaTime time.Time) (MarginLevels, error) {
	var (
		maintenanceMargin, searchLevel, initialMargin, collateralReleaseLevel decimal.Decimal
		err                                                                   error
	)

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
		MarketID:               NewMarketID(margin.MarketId),
		AssetID:                NewAssetID(margin.Asset),
		PartyID:                NewPartyID(margin.PartyId),
		MaintenanceMargin:      maintenanceMargin,
		SearchLevel:            searchLevel,
		InitialMargin:          initialMargin,
		CollateralReleaseLevel: collateralReleaseLevel,
		Timestamp:              time.Unix(0, margin.Timestamp),
		VegaTime:               vegaTime,
	}, nil
}

func (ml *MarginLevels) ToProto() *vega.MarginLevels {
	return &vega.MarginLevels{
		MaintenanceMargin:      ml.MaintenanceMargin.String(),
		SearchLevel:            ml.SearchLevel.String(),
		InitialMargin:          ml.InitialMargin.String(),
		CollateralReleaseLevel: ml.CollateralReleaseLevel.String(),
		PartyId:                ml.PartyID.String(),
		MarketId:               ml.MarketID.String(),
		Asset:                  ml.AssetID.String(),
		Timestamp:              ml.Timestamp.UnixNano(),
	}
}

type MarginLevelsKey struct {
	MarketID MarketID
	AssetID  AssetID
	PartyID  PartyID
	VegaTime time.Time
}

func (o MarginLevels) Key() MarginLevelsKey {
	return MarginLevelsKey{o.MarketID, o.AssetID, o.PartyID, o.VegaTime}
}

func (ml MarginLevels) ToRow() []interface{} {
	return []interface{}{
		ml.MarketID, ml.AssetID, ml.PartyID, ml.Timestamp, ml.MaintenanceMargin,
		ml.SearchLevel, ml.InitialMargin, ml.CollateralReleaseLevel, ml.VegaTime,
	}
}

var MarginLevelsColumns = []string{
	"market_id", "asset_id", "party_id", "timestamp", "maintenance_margin",
	"search_level", "initial_margin", "collateral_release_level", "vega_time",
}
