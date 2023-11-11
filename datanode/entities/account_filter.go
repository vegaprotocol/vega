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
	"code.vegaprotocol.io/vega/core/types"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
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
			filter.AssetID = AssetID(pbFilter.AssetId)
		}
		for _, partyID := range pbFilter.PartyIds {
			filter.PartyIDs = append(filter.PartyIDs, PartyID(partyID))
		}

		filter.AccountTypes = append(filter.AccountTypes, pbFilter.AccountTypes...)

		for _, marketID := range pbFilter.MarketIds {
			filter.MarketIDs = append(filter.MarketIDs, MarketID(marketID))
		}
	}
	return filter, nil
}
