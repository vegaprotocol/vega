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
	"code.vegaprotocol.io/protos/vega"
	"github.com/shopspring/decimal"
)

type RewardSummary struct {
	PartyID PartyID
	AssetID AssetID
	Amount  decimal.Decimal
}

func (r *RewardSummary) ToProto() *vega.RewardSummary {
	protoRewardSummary := vega.RewardSummary{
		PartyId: r.PartyID.String(),
		AssetId: r.AssetID.String(),
		Amount:  r.Amount.String(),
	}
	return &protoRewardSummary
}
