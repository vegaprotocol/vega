package entities

import (
	"encoding/hex"
	"fmt"

	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/types"
)

type AccountFilter struct {
	Asset        Asset
	Parties      []Party
	AccountTypes []types.AccountType
	Markets      []Market
}

func AccountFilterFromProto(pbFilter *v2.AccountFilter) (AccountFilter, error) {
	filter := AccountFilter{}
	if pbFilter != nil {
		if pbFilter.AssetId != "" {
			filter.Asset.ID = MakeAssetID(pbFilter.AssetId)
		}
		for _, partyID := range pbFilter.PartyIds {
			hexID, err := hex.DecodeString(partyID)
			if err != nil {
				return filter, fmt.Errorf("party ID is not a hex string: %v", partyID)
			}
			filter.Parties = append(filter.Parties, Party{ID: hexID})
		}

		filter.AccountTypes = append(filter.AccountTypes, pbFilter.AccountTypes...)

		for _, marketID := range pbFilter.MarketIds {
			hexID, err := hex.DecodeString(marketID)
			if err != nil {
				return filter, fmt.Errorf("market ID is not a hex string: %v", marketID)
			}
			filter.Markets = append(filter.Markets, Market{ID: hexID})
		}
	}
	return filter, nil
}
