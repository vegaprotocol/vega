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

package types

import (
	"sort"

	"code.vegaprotocol.io/vega/libs/num"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"golang.org/x/exp/maps"
)

type PaidLiquidityFeesStats struct {
	TotalFeesPaid    *num.Uint
	FeesPaidPerParty map[string]*num.Uint
}

func NewLiquidityFeeStats() *PaidLiquidityFeesStats {
	return &PaidLiquidityFeesStats{
		TotalFeesPaid:    num.UintZero(),
		FeesPaidPerParty: map[string]*num.Uint{},
	}
}

func NewPaidLiquidityFeesStatsFromProto(fsp *eventspb.PaidLiquidityFeesStats) *PaidLiquidityFeesStats {
	fs := NewLiquidityFeeStats()

	fs.TotalFeesPaid = num.MustUintFromString(fsp.TotalFeesPaid, 10)

	for _, fpp := range fsp.FeesPaidPerParty {
		fs.FeesPaidPerParty[fpp.Party] = num.MustUintFromString(fpp.Amount, 10)
	}

	return fs
}

func (f *PaidLiquidityFeesStats) RegisterTotalFeesAmountPerParty(totalFeesAmountPerParty map[string]*num.Uint) {
	for party, amount := range totalFeesAmountPerParty {
		f.TotalFeesPaid.AddSum(amount)

		if _, ok := f.FeesPaidPerParty[party]; !ok {
			f.FeesPaidPerParty[party] = amount.Clone()
			continue
		}
		f.FeesPaidPerParty[party].AddSum(amount)
	}
}

func (f *PaidLiquidityFeesStats) ToProto(marketID, asset string, epochSeq uint64) *eventspb.PaidLiquidityFeesStats {
	fs := &eventspb.PaidLiquidityFeesStats{
		Market:           marketID,
		Asset:            asset,
		EpochSeq:         epochSeq,
		FeesPaidPerParty: make([]*eventspb.PartyAmount, 0, len(f.FeesPaidPerParty)),
		TotalFeesPaid:    f.TotalFeesPaid.String(),
	}

	allParties := maps.Keys(f.FeesPaidPerParty)
	sort.Strings(allParties)

	for _, party := range allParties {
		amount := f.FeesPaidPerParty[party]
		fs.FeesPaidPerParty = append(fs.FeesPaidPerParty, &eventspb.PartyAmount{
			Party:  party,
			Amount: amount.String(),
		})
	}

	return fs
}
