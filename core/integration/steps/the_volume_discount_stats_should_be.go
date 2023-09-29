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

func TheVolumeDiscountStatsShouldBe(broker *stubs.BrokerStub, epochStr string, table *godog.Table) error {
	epoch, err := U64(epochStr)
	if err != nil {
		return fmt.Errorf("could not parse epoch: %w", err)
	}

	expectedVolumeDiscountStatsStats, err := parseVolumeDiscountStatsShouldBeTable(table)
	if err != nil {
		return fmt.Errorf("table is invalid: %w", err)
	}

	VolumeDiscountStats := broker.VolumeDiscountStats()

	foundStreaksForEpoch := map[string]eventspb.PartyVolumeDiscountStats{}
	for _, stats := range VolumeDiscountStats {
		if stats.AtEpoch == epoch {
			return compareVolumeDiscountStats(expectedVolumeDiscountStatsStats, foundStreaksForEpoch)
		}
	}

	return fmt.Errorf("no volume discount stats found for epoch %q", epochStr)
}

func parseVolumeDiscountStatsShouldBeTable(table *godog.Table) (map[string]eventspb.PartyVolumeDiscountStats, error) {
	rows := StrictParseTable(table, []string{
		"party",
		"running volume",
		"discount factor",
	}, []string{})

	stats := map[string]eventspb.PartyVolumeDiscountStats{}
	for _, row := range rows {
		specificRow := newVolumeDiscountStatsShouldBeRow(row)
		partyID := specificRow.Party()
		_, alreadyRegistered := stats[partyID]
		if alreadyRegistered {
			return nil, fmt.Errorf("cannot have more than one expectation for party %q", partyID)
		}
		stats[partyID] = eventspb.PartyVolumeDiscountStats{
			PartyId:        partyID,
			DiscountFactor: specificRow.DiscountFactor(),
			RunningVolume:  specificRow.RunningVolume(),
		}
	}

	return stats, nil
}

func compareVolumeDiscountStats(expectedStats, foundStats map[string]eventspb.PartyVolumeDiscountStats) error {
	foundStatsIDs := maps.Keys(expectedStats)
	expectedStatsIDs := maps.Keys(expectedStats)

	slices.Sort(foundStatsIDs)
	slices.Sort(expectedStatsIDs)

	unexpectedParties := []string{}
	partiesNotFound := []string{}

	for _, expectedID := range expectedStatsIDs {
		if _, ok := foundStats[expectedID]; !ok {
			partiesNotFound = append(partiesNotFound, expectedID)
		}
	}

	for _, foundID := range foundStatsIDs {
		if _, ok := expectedStats[foundID]; !ok {
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

	for _, party := range expectedStatsIDs {
		foundActivityStreak := foundStats[party]
		expectedActivityStreak := expectedStats[party]

		if expectedActivityStreak.RunningVolume != foundActivityStreak.RunningVolume ||
			expectedActivityStreak.DiscountFactor != foundActivityStreak.DiscountFactor {
			return formatDiff(
				fmt.Sprintf("Volume discount stats did not match for party %q", party),
				map[string]string{
					"running volume":  expectedActivityStreak.RunningVolume,
					"discount factor": expectedActivityStreak.DiscountFactor,
				},
				map[string]string{
					"running volume":  foundActivityStreak.RunningVolume,
					"discount factor": foundActivityStreak.DiscountFactor,
				},
			)
		}
	}

	return nil
}

type volumeDiscountStatsShouldBeRow struct {
	row RowWrapper
}

func newVolumeDiscountStatsShouldBeRow(r RowWrapper) volumeDiscountStatsShouldBeRow {
	return volumeDiscountStatsShouldBeRow{
		row: r,
	}
}

func (r volumeDiscountStatsShouldBeRow) Party() string {
	return r.row.MustStr("party")
}

func (r volumeDiscountStatsShouldBeRow) DiscountFactor() string {
	return r.row.MustStr("discount factor")
}

func (r volumeDiscountStatsShouldBeRow) RunningVolume() string {
	return r.row.MustStr("running volume")
}
