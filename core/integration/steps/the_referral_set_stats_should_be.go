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

package steps

import (
	"errors"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/cucumber/godog"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

func TheReferralSetStatsShouldBe(broker *stubs.BrokerStub, code, epochStr, volumeStr string, table *godog.Table) error {
	epoch, err := U64(epochStr)
	if err != nil {
		return fmt.Errorf("could not parse epoch: %w", err)
	}

	expectedVolume, overflown := num.UintFromString(volumeStr, 10)
	if overflown {
		return fmt.Errorf("could not parse the expected volume %q", volumeStr)
	}
	setID := types.ReferralSetID(code)

	expectedRefereesStats, err := parseReferralStatsShouldBeTable(table)
	if err != nil {
		return fmt.Errorf("table is invalid: %w", err)
	}

	stats := broker.ReferralSetStats()
	for _, stat := range stats {
		if stat.AtEpoch == epoch && stat.SetID == setID {
			if !stat.ReferralSetRunningVolume.EQ(expectedVolume) {
				return fmt.Errorf("refferal set stats for set ID %q at epoch %q expect a running volume of %v, but got %v", code, epochStr, volumeStr, stat.ReferralSetRunningVolume)
			}

			return compareRefereesStats(expectedRefereesStats, stat.RefereesStats, stat.RewardFactors)
		}
	}

	return fmt.Errorf("no stats found for set ID %q at epoch %q", code, epochStr)
}

type refereeStats struct {
	DiscountFactor types.Factors
	RewardFactor   types.Factors
}

func parseReferralStatsShouldBeTable(table *godog.Table) (map[types.PartyID]*refereeStats, error) {
	rows := StrictParseTable(table, []string{
		"party",
		"discount infra factor",
		"discount maker factor",
		"discount liquidity factor",
		"reward infra factor",
		"reward maker factor",
		"reward liquidity factor",
	}, []string{})

	stats := map[types.PartyID]*refereeStats{}
	for _, row := range rows {
		specificRow := newReferralSetStatsShouldBeRow(row)
		partyID := specificRow.Party()
		_, alreadyRegistered := stats[partyID]
		if alreadyRegistered {
			return nil, fmt.Errorf("cannot have more than one expectation for party %q", partyID)
		}
		stats[partyID] = &refereeStats{
			DiscountFactor: types.Factors{
				Infra:     specificRow.DiscountInfraFactor(),
				Maker:     specificRow.DiscountMakerFactor(),
				Liquidity: specificRow.DiscountLiqFactor(),
			},
			RewardFactor: types.Factors{
				Infra:     specificRow.RewardInfraFactor(),
				Maker:     specificRow.RewardMakerFactor(),
				Liquidity: specificRow.RewardLiqFactor(),
			},
		}
	}

	return stats, nil
}

func compareRefereesStats(
	expectedRefereesStats map[types.PartyID]*refereeStats,
	foundRefereesStats map[types.PartyID]*types.RefereeStats,
	foundRewardFactor types.Factors,
) error {
	foundRefereesIDs := maps.Keys(foundRefereesStats)
	expectedRefereesIDs := maps.Keys(expectedRefereesStats)

	slices.Sort(foundRefereesIDs)
	slices.Sort(expectedRefereesIDs)

	unexpectedParties := []string{}
	partiesNotFound := []string{}

	for _, expectedID := range expectedRefereesIDs {
		if _, ok := foundRefereesStats[expectedID]; !ok {
			partiesNotFound = append(partiesNotFound, string(expectedID))
		}
	}

	for _, foundID := range foundRefereesIDs {
		if _, ok := expectedRefereesStats[foundID]; !ok {
			unexpectedParties = append(unexpectedParties, string(foundID))
		}
	}

	var errStr string
	if len(partiesNotFound) > 0 {
		errStr = "parties not found: " + strings.Join(partiesNotFound, ", ")
	}
	if len(unexpectedParties) > 0 {
		if errStr != "" {
			errStr += ", and "
		}
		errStr += "unexpected parties: " + strings.Join(unexpectedParties, ", ")
	}
	if errStr != "" {
		return errors.New(errStr)
	}

	for _, refereeID := range expectedRefereesIDs {
		refereeIDStr := string(refereeID)
		foundRefereeStats := foundRefereesStats[refereeID]
		expectedRefereeStats := expectedRefereesStats[refereeID]
		if !expectedRefereeStats.RewardFactor.Equal(foundRewardFactor) {
			return fmt.Errorf("expecting reward factor of %v but got %v for party %q", expectedRefereeStats.RewardFactor.String(), foundRewardFactor.String(), refereeIDStr)
		}
		if !foundRefereeStats.DiscountFactors.Equal(expectedRefereeStats.DiscountFactor) {
			return fmt.Errorf("expecting discount factor of %v but got %v for party %q", expectedRefereeStats.DiscountFactor.String(), foundRefereeStats.DiscountFactors.String(), refereeIDStr)
		}
	}

	return nil
}

type referralSetStatsShouldBeRow struct {
	row RowWrapper
}

func newReferralSetStatsShouldBeRow(r RowWrapper) referralSetStatsShouldBeRow {
	row := referralSetStatsShouldBeRow{
		row: r,
	}
	return row
}

func (r referralSetStatsShouldBeRow) Party() types.PartyID {
	return types.PartyID(r.row.MustStr("party"))
}

func (r referralSetStatsShouldBeRow) DiscountInfraFactor() num.Decimal {
	return r.row.MustDecimal("discount infra factor")
}

func (r referralSetStatsShouldBeRow) DiscountMakerFactor() num.Decimal {
	return r.row.MustDecimal("discount maker factor")
}

func (r referralSetStatsShouldBeRow) DiscountLiqFactor() num.Decimal {
	return r.row.MustDecimal("discount liquidity factor")
}

func (r referralSetStatsShouldBeRow) RewardInfraFactor() num.Decimal {
	return r.row.MustDecimal("reward infra factor")
}

func (r referralSetStatsShouldBeRow) RewardMakerFactor() num.Decimal {
	return r.row.MustDecimal("reward maker factor")
}

func (r referralSetStatsShouldBeRow) RewardLiqFactor() num.Decimal {
	return r.row.MustDecimal("reward liquidity factor")
}
