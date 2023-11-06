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
	"strconv"
	"strings"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/cucumber/godog"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

func TheActivityStreaksShouldBe(broker *stubs.BrokerStub, epochStr string, table *godog.Table) error {
	epoch, err := U64(epochStr)
	if err != nil {
		return fmt.Errorf("could not parse epoch: %w", err)
	}

	expectedActivityStreaksStats, err := parseActivityStreaksShouldBeTable(table)
	if err != nil {
		return fmt.Errorf("table is invalid: %w", err)
	}

	allStreaks := broker.PartyActivityStreaks()

	foundStreaksForEpoch := map[string]eventspb.PartyActivityStreak{}
	for _, streak := range allStreaks {
		if streak.Epoch == epoch {
			foundStreaksForEpoch[streak.Party] = streak
		}
	}

	if len(foundStreaksForEpoch) == 0 && len(expectedActivityStreaksStats) > 0 {
		return fmt.Errorf("no activity streaks found at epoch %v", epochStr)
	}

	return compareActivityStreaks(expectedActivityStreaksStats, foundStreaksForEpoch)
}

func parseActivityStreaksShouldBeTable(table *godog.Table) (map[string]eventspb.PartyActivityStreak, error) {
	rows := StrictParseTable(table, []string{
		"party",
		"active for",
		"inactive for",
		"reward multiplier",
		"vesting multiplier",
	}, []string{})

	stats := map[string]eventspb.PartyActivityStreak{}
	for _, row := range rows {
		specificRow := newActivityStreaksShouldBeRow(row)
		partyID := specificRow.Party()
		_, alreadyRegistered := stats[partyID]
		if alreadyRegistered {
			return nil, fmt.Errorf("cannot have more than one expectation for party %q", partyID)
		}
		stats[partyID] = eventspb.PartyActivityStreak{
			Party:                                partyID,
			ActiveFor:                            specificRow.ActiveFor(),
			InactiveFor:                          specificRow.InactiveFor(),
			RewardDistributionActivityMultiplier: specificRow.RewardMultiplier(),
			RewardVestingActivityMultiplier:      specificRow.VestingMultiplier(),
		}
	}

	return stats, nil
}

func compareActivityStreaks(expectedActivityStreaks map[string]eventspb.PartyActivityStreak, foundActivityStreaks map[string]eventspb.PartyActivityStreak) error {
	foundActivityStreaksIDs := maps.Keys(expectedActivityStreaks)
	expectedActivityStreaksIDs := maps.Keys(expectedActivityStreaks)

	slices.Sort(foundActivityStreaksIDs)
	slices.Sort(expectedActivityStreaksIDs)

	unexpectedParties := []string{}
	partiesNotFound := []string{}

	for _, expectedID := range expectedActivityStreaksIDs {
		if _, ok := foundActivityStreaks[expectedID]; !ok {
			partiesNotFound = append(partiesNotFound, expectedID)
		}
	}

	for _, foundID := range foundActivityStreaksIDs {
		if _, ok := expectedActivityStreaks[foundID]; !ok {
			unexpectedParties = append(unexpectedParties, foundID)
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

	for _, party := range expectedActivityStreaksIDs {
		foundActivityStreak := foundActivityStreaks[party]
		expectedActivityStreak := expectedActivityStreaks[party]

		if expectedActivityStreak.ActiveFor != foundActivityStreak.ActiveFor ||
			expectedActivityStreak.InactiveFor != foundActivityStreak.InactiveFor ||
			expectedActivityStreak.RewardDistributionActivityMultiplier != foundActivityStreak.RewardDistributionActivityMultiplier ||
			expectedActivityStreak.RewardVestingActivityMultiplier != foundActivityStreak.RewardVestingActivityMultiplier {
			return formatDiff(
				fmt.Sprintf("activity streak did not match for party %q", party),
				map[string]string{
					"active for":         strconv.FormatUint(expectedActivityStreak.ActiveFor, 10),
					"inactive for":       strconv.FormatUint(expectedActivityStreak.InactiveFor, 10),
					"reward multiplier":  expectedActivityStreak.RewardDistributionActivityMultiplier,
					"vesting multiplier": expectedActivityStreak.RewardVestingActivityMultiplier,
				},
				map[string]string{
					"active for":         strconv.FormatUint(foundActivityStreak.ActiveFor, 10),
					"inactive for":       strconv.FormatUint(foundActivityStreak.InactiveFor, 10),
					"reward multiplier":  foundActivityStreak.RewardDistributionActivityMultiplier,
					"vesting multiplier": foundActivityStreak.RewardVestingActivityMultiplier,
				},
			)
		}
	}

	return nil
}

type activityStreaksShouldBeRow struct {
	row RowWrapper
}

func newActivityStreaksShouldBeRow(r RowWrapper) activityStreaksShouldBeRow {
	row := activityStreaksShouldBeRow{
		row: r,
	}
	return row
}

func (r activityStreaksShouldBeRow) Party() string {
	return r.row.MustStr("party")
}

func (r activityStreaksShouldBeRow) ActiveFor() uint64 {
	return r.row.MustU64("active for")
}

func (r activityStreaksShouldBeRow) InactiveFor() uint64 {
	return r.row.MustU64("inactive for")
}

func (r activityStreaksShouldBeRow) RewardMultiplier() string {
	return r.row.MustStr("reward multiplier")
}

func (r activityStreaksShouldBeRow) VestingMultiplier() string {
	return r.row.MustStr("vesting multiplier")
}
