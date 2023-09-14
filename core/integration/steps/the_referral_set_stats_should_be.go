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

			return compareRefereesStats(expectedRefereesStats, stat.RefereesStats)
		}
	}

	return fmt.Errorf("no stats found for set ID %q at epoch %q", code, epochStr)
}

func parseReferralStatsShouldBeTable(table *godog.Table) (map[types.PartyID]*types.RefereeStats, error) {
	rows := StrictParseTable(table, []string{
		"party",
		"discount factor",
		"reward factor",
	}, []string{})

	stats := map[types.PartyID]*types.RefereeStats{}
	for _, row := range rows {
		specificRow := newReferralSetStatsShouldBeRow(row)
		partyID := specificRow.Party()
		_, alreadyRegistered := stats[partyID]
		if alreadyRegistered {
			return nil, fmt.Errorf("cannot have more than one expectation for party %q", partyID)
		}
		stats[partyID] = &types.RefereeStats{
			DiscountFactor: specificRow.DiscountFactor(),
			RewardFactor:   specificRow.RewardFactor(),
		}
	}

	return stats, nil
}

func compareRefereesStats(expectedRefereesStats, foundRefereesStats map[types.PartyID]*types.RefereeStats) error {
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
		errStr = "unexpected parties: " + strings.Join(unexpectedParties, ", ")
	}
	if errStr != "" {
		return errors.New(errStr)
	}

	for _, refereeID := range expectedRefereesIDs {
		refereeIDStr := string(refereeID)
		foundRefereeStats := foundRefereesStats[refereeID]
		expectedRefereeStats := expectedRefereesStats[refereeID]
		if !foundRefereeStats.RewardFactor.Equal(expectedRefereeStats.RewardFactor) {
			return fmt.Errorf("expecting reward factor of %v but got %v for party %q", expectedRefereeStats.RewardFactor.String(), foundRefereeStats.RewardFactor.String(), refereeIDStr)
		}
		if !foundRefereeStats.DiscountFactor.Equal(expectedRefereeStats.DiscountFactor) {
			return fmt.Errorf("expecting discount factor of %v but got %v for party %q", expectedRefereeStats.DiscountFactor.String(), foundRefereeStats.DiscountFactor.String(), refereeIDStr)
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

func (r referralSetStatsShouldBeRow) DiscountFactor() num.Decimal {
	return r.row.MustDecimal("discount factor")
}

func (r referralSetStatsShouldBeRow) RewardFactor() num.Decimal {
	return r.row.MustDecimal("reward factor")
}
