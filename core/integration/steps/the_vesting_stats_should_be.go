package steps

import (
	"errors"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	"github.com/cucumber/godog"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

func TheVestingStatsShouldBe(broker *stubs.BrokerStub, epochStr string, table *godog.Table) error {
	epoch, err := U64(epochStr)
	if err != nil {
		return fmt.Errorf("could not parse epoch: %w", err)
	}

	expectedVestingStatsStats, err := parseVestingStatsShouldBeTable(table)
	if err != nil {
		return fmt.Errorf("table is invalid: %w", err)
	}

	vestingStats := broker.VestingStats()

	foundStreaksForEpoch := map[string]eventspb.PartyVestingStats{}
	for _, stats := range vestingStats {
		if stats.AtEpoch == epoch {
			return compareVestingStats(expectedVestingStatsStats, foundStreaksForEpoch)
		}
	}

	return fmt.Errorf("no vesting stats found for epoch %q", epochStr)
}

func parseVestingStatsShouldBeTable(table *godog.Table) (map[string]eventspb.PartyVestingStats, error) {
	rows := StrictParseTable(table, []string{
		"party",
		"reward bonus multiplier",
	}, []string{})

	stats := map[string]eventspb.PartyVestingStats{}
	for _, row := range rows {
		specificRow := newVestingStatsShouldBeRow(row)
		partyID := specificRow.Party()
		_, alreadyRegistered := stats[partyID]
		if alreadyRegistered {
			return nil, fmt.Errorf("cannot have more than one expectation for party %q", partyID)
		}
		stats[partyID] = eventspb.PartyVestingStats{
			PartyId:               partyID,
			RewardBonusMultiplier: specificRow.RewardBonusMultiplier(),
		}
	}

	return stats, nil
}

func compareVestingStats(expectedVestingStats map[string]eventspb.PartyVestingStats, foundVestingStats map[string]eventspb.PartyVestingStats) error {
	foundVestingStatsIDs := maps.Keys(expectedVestingStats)
	expectedVestingStatsIDs := maps.Keys(expectedVestingStats)

	slices.Sort(foundVestingStatsIDs)
	slices.Sort(expectedVestingStatsIDs)

	unexpectedParties := []string{}
	partiesNotFound := []string{}

	for _, expectedID := range expectedVestingStatsIDs {
		if _, ok := foundVestingStats[expectedID]; !ok {
			partiesNotFound = append(partiesNotFound, expectedID)
		}
	}

	for _, foundID := range foundVestingStatsIDs {
		if _, ok := expectedVestingStats[foundID]; !ok {
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

	for _, party := range expectedVestingStatsIDs {
		foundActivityStreak := foundVestingStats[party]
		expectedActivityStreak := expectedVestingStats[party]

		if expectedActivityStreak.RewardBonusMultiplier != foundActivityStreak.RewardBonusMultiplier {
			return formatDiff(
				fmt.Sprintf("vesting stats did not match for party %q", party),
				map[string]string{
					"reward bonus multiplier": expectedActivityStreak.RewardBonusMultiplier,
				},
				map[string]string{
					"reward bonus multiplier": foundActivityStreak.RewardBonusMultiplier,
				},
			)
		}
	}

	return nil
}

type vestingStatsShouldBeRow struct {
	row RowWrapper
}

func newVestingStatsShouldBeRow(r RowWrapper) vestingStatsShouldBeRow {
	return vestingStatsShouldBeRow{
		row: r,
	}
}

func (r vestingStatsShouldBeRow) Party() string {
	return r.row.MustStr("party")
}

func (r vestingStatsShouldBeRow) RewardBonusMultiplier() string {
	return r.row.MustStr("reward bonus multiplier")
}
