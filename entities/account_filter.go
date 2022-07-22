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
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/types"
)

type AccountFilter struct {
	AssetID      AssetID
	PartyIDs     []PartyID
	AccountTypes []types.AccountType
	MarketIDs    []MarketID
}

func AccountFilterFromProto(pbFilter *v2.AccountFilter) (AccountFilter, error) {
	filter := AccountFilter{}
	if pbFilter != nil {
		if pbFilter.AssetId != "" {
			filter.AssetID = NewAssetID(pbFilter.AssetId)
		}
		for _, partyID := range pbFilter.PartyIds {
			filter.PartyIDs = append(filter.PartyIDs, NewPartyID(partyID))
		}

		filter.AccountTypes = append(filter.AccountTypes, pbFilter.AccountTypes...)

		for _, marketID := range pbFilter.MarketIds {
			filter.MarketIDs = append(filter.MarketIDs, NewMarketID(marketID))
		}
	}
	return filter, nil
}
