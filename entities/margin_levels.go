package entities

import (
	"encoding/hex"
	"fmt"
	"time"

	"code.vegaprotocol.io/protos/vega"
	"github.com/shopspring/decimal"
)

type MarginLevels struct {
	MarketID               []byte
	AssetID                []byte
	PartyID                []byte
	MaintenanceMargin      decimal.Decimal
	SearchLevel            decimal.Decimal
	InitialMargin          decimal.Decimal
	CollateralReleaseLevel decimal.Decimal
	Timestamp              time.Time
	VegaTime               time.Time
}

func MarginLevelsFromProto(margin *vega.MarginLevels, vegaTime time.Time) (*MarginLevels, error) {
	var (
		marketID, assetID, partyID                                            []byte
		maintenanceMargin, searchLevel, initialMargin, collateralReleaseLevel decimal.Decimal
		err                                                                   error
	)

	if marketID, err = makeID(margin.MarketId); err != nil {
		return nil, fmt.Errorf("invalid market ID: %w", err)
	}

	if assetID, err = makeID(margin.Asset); err != nil {
		return nil, fmt.Errorf("invalid asset ID: %w", err)
	}

	if partyID, err = makeID(margin.PartyId); err != nil {
		return nil, fmt.Errorf("invalid party ID: %w", err)
	}

	if maintenanceMargin, err = decimal.NewFromString(margin.MaintenanceMargin); err != nil {
		return nil, fmt.Errorf("invalid maintenance margin: %w", err)
	}

	if searchLevel, err = decimal.NewFromString(margin.SearchLevel); err != nil {
		return nil, fmt.Errorf("invalid search level: %w", err)
	}

	if initialMargin, err = decimal.NewFromString(margin.InitialMargin); err != nil {
		return nil, fmt.Errorf("invalid initial margin: %w", err)
	}

	if collateralReleaseLevel, err = decimal.NewFromString(margin.CollateralReleaseLevel); err != nil {
		return nil, fmt.Errorf("invalid collateralReleaseLevel: %w", err)
	}

	return &MarginLevels{
		MarketID:               marketID,
		AssetID:                assetID,
		PartyID:                partyID,
		MaintenanceMargin:      maintenanceMargin,
		SearchLevel:            searchLevel,
		InitialMargin:          initialMargin,
		CollateralReleaseLevel: collateralReleaseLevel,
		Timestamp:              time.Unix(0, margin.Timestamp),
		VegaTime:               vegaTime,
	}, nil
}

func (ml *MarginLevels) ToProto() *vega.MarginLevels {
	marketID := hex.EncodeToString(ml.MarketID)
	assetID := hex.EncodeToString(ml.AssetID)
	partyID := hex.EncodeToString(ml.PartyID)

	return &vega.MarginLevels{
		MaintenanceMargin:      ml.MaintenanceMargin.String(),
		SearchLevel:            ml.SearchLevel.String(),
		InitialMargin:          ml.InitialMargin.String(),
		CollateralReleaseLevel: ml.CollateralReleaseLevel.String(),
		PartyId:                partyID,
		MarketId:               marketID,
		Asset:                  assetID,
		Timestamp:              ml.Timestamp.UnixNano(),
	}
}
