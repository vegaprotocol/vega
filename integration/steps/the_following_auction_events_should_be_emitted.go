package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/integration/stubs"
	"github.com/cucumber/godog/gherkin"
)

func TheAuctionExtensionTriggerShouldBe(broker *stubs.BrokerStub, market, extension string) error {
	e := broker.GetLastAuctionEvent(market)
	if e == nil {
		return fmt.Errorf("no auction event for market %s", market)
	}
	trigger, err := AuctionTrigger(extension)
	if err != nil {
		return err
	}
	if pet := e.Proto().ExtensionTrigger; pet != trigger {
		return fmt.Errorf("expected extension trigger to be %s instead got %s", trigger, pet)
	}
	return nil
}

func TheFollowingAuctionEventsShouldBeEmitted(broker *stubs.BrokerStub, table *gherkin.DataTable) error {
	parsed := TableWrapper(*table).Parse()
	if len(parsed) == 1 {
		row := parsed[0]
		market := row.MustStr("market id")
		e := broker.GetLastAuctionEvent(market)
		if e == nil {
			return fmt.Errorf("No auction event for market %s", market)
		}
		return checkEvent(row, *e)
	}
	evts := broker.GetAuctionEvents()
	for _, row := range TableWrapper(*table).Parse() {
		market := row.MustStr("market id")
		last := len(evts) - 1
		var err error
		for i, e := range evts {
			if e.MarketID() != market {
				continue
			}
			if err = checkEvent(row, e); err == nil {
				if i == last {
					evts = evts[:i]
				} else {
					evts = append(evts[:i], evts[i+1:]...)
				}
				break
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func checkEvent(row RowWrapper, e events.Auction) error {
	market := row.MustStr("market id")
	isAuction := e.Auction()
	if row.Has("is auction") {
		isAuction = row.MustBool("is auction")
	}
	if e.Auction() != isAuction {
		return fmt.Errorf("market %s expected in auction %t instead got %t", market, isAuction, e.Auction())
	}
	proto := e.Proto()
	trigger := proto.Trigger
	if row.Has("auction trigger") {
		trigger = row.MustAuctionTrigger("auction trigger")
	}
	if proto.Trigger != trigger {
		return fmt.Errorf("expected auction trigger %s instead got %s", trigger, proto.Trigger)
	}
	extension := proto.ExtensionTrigger
	if row.Has("extension trigger") {
		extension = row.MustAuctionTrigger("extension trigger")
	}
	if extension != proto.ExtensionTrigger {
		return fmt.Errorf("Expected extension trigger %s instead got %s", extension, proto.ExtensionTrigger)
	}
	opening := proto.OpeningAuction
	if row.Has("is opening") {
		opening = row.MustBool("is opening")
	}
	if opening != proto.OpeningAuction {
		return fmt.Errorf("Expected opening auction to be %v istead got %v", opening, proto.OpeningAuction)
	}
	return nil
}
