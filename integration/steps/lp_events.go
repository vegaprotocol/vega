package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/integration/stubs"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/cucumber/godog"
)

func TheFollowingLPEventsShouldBeEmitted(broker *stubs.BrokerStub, table *godog.Table) error {
	lpEvts := broker.GetLPEvents()
	evtsByPartyID := map[string]map[string][]events.LiquidityProvision{}
	for _, e := range lpEvts {
		party := e.PartyID()
		lpID := e.LiquidityProvision().Reference
		m, ok := evtsByPartyID[party]
		if !ok {
			m = map[string][]events.LiquidityProvision{}
		}
		s, ok := m[lpID]
		if !ok {
			s = []events.LiquidityProvision{}
		}
		m[lpID] = append(s, e)
		evtsByPartyID[party] = m
	}
	for _, row := range parseLPEventTable(table) {
		lpe := LPEventWrapper{
			row: row,
		}
		party, id, version, final, amt := lpe.Party(), lpe.ID(), lpe.Version(), lpe.Final(), lpe.CommitmentAmount()
		lpIDMap, ok := evtsByPartyID[party]
		if !ok {
			return fmt.Errorf("no LP events found for party %s", party)
		}
		evts, ok := lpIDMap[id]
		if !ok {
			return fmt.Errorf("no LP events found for LP ID %s (party %s)", id, party)
		}
		if final {
			if err := checkLPEvent(evts[len(evts)-1], party, id, amt, version); err != nil {
				return err
			}
			continue
		}
		// find matching event in slice
		var err error
		// find matching event
		for _, e := range evts {
			if err = checkLPEvent(e, party, id, amt, version); err == nil {
				// match found, break
				break
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func checkLPEvent(last events.LiquidityProvision, party, id string, amt *num.Uint, version uint64) error {
	if last.LiquidityProvision().Version != version {
		return fmt.Errorf("version %d is not the last version for LP %s (party %s), last is %d", version, id, party, last.LiquidityProvision().Version)
	}
	if amt != nil && last.LiquidityProvision().CommitmentAmount != amt.String() {
		fmt.Errorf("commitment amount was %s, expected %s for last event for LP %s, party %s",
			last.LiquidityProvision().CommitmentAmount,
			amt,
			id,
			party,
		)
	}
	return nil
}

type LPEventWrapper struct {
	row RowWrapper
}

func parseLPEventTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"id",
		"version",
	}, []string{
		"commitment amount",
		"final",
	})
}

func (l LPEventWrapper) Party() string {
	return l.row.MustStr("party")
}

func (l LPEventWrapper) ID() string {
	return l.row.MustStr("id")
}

func (l LPEventWrapper) Version() uint64 {
	return l.row.MustU64("version")
}

func (l LPEventWrapper) CommitmentAmount() *num.Uint {
	if !l.row.HasColumn("commitment amount") {
		return nil
	}
	return l.row.MustUint("commitment amount")
}

func (l LPEventWrapper) Final() bool {
	if !l.row.HasColumn("final") {
		return false
	}
	return l.row.MustBool("final")
}
