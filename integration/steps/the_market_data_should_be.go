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

package steps

import (
	"fmt"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/cucumber/godog"
)

type MappedMD struct {
	md      types.MarketData
	uintMap map[string]**num.Uint
	u64Map  map[string]*uint64
	strMap  map[string]*string
	tMap    map[string]*int64
	i64Map  map[string]*int64
	tm      *types.MarketTradingMode
	tr      *types.AuctionTrigger
	et      *types.AuctionTrigger
}

type ErrStack []error

func TheMarketDataShouldBe(engine Execution, mID string, data *godog.Table) error {
	actual, err := engine.GetMarketData(mID)
	if err != nil {
		return err
	}
	// create a copy (deep copy), override the values we've gotten with those from the table so we can compare the objects
	expect := mappedMD(actual)
	// special fields first, these need to be compared manually
	u64Set := expect.parseU64(data)
	i64Set := expect.parseI64(data)
	tSet := expect.parseTimes(data)
	strSet := expect.parseStr(data)
	uintSet := expect.parseUint(data)
	expect.parseSpecial(data)
	if pm := getPriceBounds(data); len(pm) > 0 {
		expect.md.PriceMonitoringBounds = pm
	}
	// this might be a sparse check
	if lp := getLPFeeShare(data); len(lp) > 0 {
		expect.md.LiquidityProviderFeeShare = lp
	}
	cmp := mappedMD(actual)
	parsed := mappedMD(expect.md)
	errs := make([]error, 0, len(u64Set)+len(i64Set)+len(strSet)+2+len(uintSet))
	if expect.tm != nil && *expect.tm != expect.md.MarketTradingMode {
		errs = append(errs, fmt.Errorf("expected '%s' trading mode, instead got '%s'", *expect.tm, expect.md.MarketTradingMode))
	}
	if expect.tr != nil && *expect.tr != expect.md.Trigger {
		errs = append(errs, fmt.Errorf("expected '%s' auction trigger, instead got '%s'", *expect.tr, expect.md.Trigger))
	}
	if expect.et != nil && *expect.et != expect.md.ExtensionTrigger {
		errs = append(errs, fmt.Errorf("expected '%s' extension trigger, instead got '%s'", *expect.et, expect.md.ExtensionTrigger))
	}
	// compare uints as strings
	for _, u := range uintSet {
		e, g := parsed.uintMap[u], cmp.uintMap[u]
		if (*e).String() != (*g).String() {
			errs = append(errs, fmt.Errorf("expected '%s' for %s, instead got '%s'", (*e).String(), u, (*g).String()))
		}
	}

	// compare uint64
	for _, u := range u64Set {
		e, g := parsed.u64Map[u], cmp.u64Map[u] // get pointers to both fields
		if *e != *g {
			errs = append(errs, fmt.Errorf("expected '%d' for %s, instead got '%d'", *e, u, *g))
		}
	}
	// compare int64
	for _, i := range i64Set {
		e, g := parsed.i64Map[i], cmp.i64Map[i]
		if *e != *g {
			errs = append(errs, fmt.Errorf("expected '%d' for %s, instead got '%d'", *e, i, *g))
		}
	}
	// compare times, which is basically identical to comparing i64
	for _, i := range tSet {
		e, g := parsed.tMap[i], cmp.tMap[i]
		if *e != *g {
			errs = append(errs, fmt.Errorf("expected '%d' for %s, instead got '%d'", *e, i, *g))
		}
	}
	// compare strings
	for _, s := range strSet {
		e, g := parsed.strMap[s], cmp.strMap[s]
		if *e != *g {
			errs = append(errs, fmt.Errorf("expected '%s' for %s, instead got '%s'", *e, s, *g))
		}
	}
	if err := cmpPriceBounds(expect, actual); len(err) > 0 {
		errs = append(errs, err...)
	}
	if err := cmpLPFeeShare(expect, actual); len(err) > 0 {
		errs = append(errs, err...)
	}
	// wrap all errors in a single error type for complete information
	if len(errs) > 0 {
		return ErrStack(errs)
	}
	// compare special fields (trading mode and auction trigger)
	return nil
}

func cmpLPFeeShare(expect *MappedMD, got types.MarketData) []error {
	errs := make([]error, 0, len(expect.md.LiquidityProviderFeeShare))
	for _, lpfs := range expect.md.LiquidityProviderFeeShare {
		match := false
		var found *types.LiquidityProviderFeeShare
		for _, g := range got.LiquidityProviderFeeShare {
			if lpfs.Party == g.Party {
				found = g
				match = lpfs.AverageEntryValuation == g.AverageEntryValuation && lpfs.EquityLikeShare == g.EquityLikeShare
				break
			}
		}
		if !match {
			if found == nil {
				errs = append(errs, fmt.Errorf("no LP fee share found for party %s", lpfs.Party))
			} else {
				errs = append(errs, fmt.Errorf(
					"expected LP fee share for party %s with avg valuation %s and equity like share %s, instead got avg. valuation %s and equity %s",
					lpfs.Party,
					lpfs.AverageEntryValuation,
					lpfs.EquityLikeShare,
					found.AverageEntryValuation,
					found.EquityLikeShare,
				))
			}
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

func cmpPriceBounds(expect *MappedMD, got types.MarketData) []error {
	errs := make([]error, 0, len(expect.md.PriceMonitoringBounds))
	for _, pmb := range expect.md.PriceMonitoringBounds {
		var bounds *types.PriceMonitoringBounds
		match := false
		for _, g := range got.PriceMonitoringBounds {
			if g.Trigger.Horizon == pmb.Trigger.Horizon {
				bounds = g
				match = pmb.MaxValidPrice.EQ(g.MaxValidPrice) && pmb.MinValidPrice.EQ(g.MinValidPrice)
				if !pmb.ReferencePrice.IsZero() {
					match = match && pmb.ReferencePrice.Equal(g.ReferencePrice)
				}
				break
			}
		}
		if !match {
			if bounds == nil {
				errs = append(errs, fmt.Errorf("no price bound for horizon %d found", pmb.Trigger.Horizon))
			} else {
				errs = append(errs, fmt.Errorf(
					"expected price bounds %d-%d (ref price=%s) for horizon %d, instead got %d-%d (ref price=%s)",
					pmb.MinValidPrice,
					pmb.MaxValidPrice,
					pmb.ReferencePrice.String(),
					pmb.Trigger.Horizon,
					bounds.MinValidPrice,
					bounds.MaxValidPrice,
					bounds.ReferencePrice.String(),
				))
			}
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

func getPriceBounds(data *godog.Table) (ret []*types.PriceMonitoringBounds) {
	for _, row := range ParseTable(data) {
		h, ok := row.I64B("horizon")
		if !ok {
			return nil
		}

		referencePrice := num.DecimalZero()
		if row.HasColumn("ref price") {
			referencePrice = row.Decimal("ref price")
		}

		expected := &types.PriceMonitoringBounds{
			MinValidPrice:  row.MustUint("min bound"),
			MaxValidPrice:  row.MustUint("max bound"),
			ReferencePrice: referencePrice,
			Trigger: &types.PriceMonitoringTrigger{
				Horizon: h,
			},
		}
		ret = append(ret, expected)
	}
	return ret
}

func getLPFeeShare(data *godog.Table) (ret []*types.LiquidityProviderFeeShare) {
	for _, r := range ParseTable(data) {
		avg, ok := r.StrB("average entry valuation")
		if !ok {
			return nil
		}
		ret = append(ret, &types.LiquidityProviderFeeShare{
			Party:                 r.MustStr("party"),
			EquityLikeShare:       r.MustStr("equity share"),
			AverageEntryValuation: avg,
		})
	}
	return ret
}

func (m *MappedMD) parseSpecial(data *godog.Table) {
	todo := map[string]struct{}{
		"trading mode":      {},
		"auction trigger":   {},
		"extension trigger": {},
	}
	for _, r := range ParseTable(data) {
		for k := range todo {
			if _, ok := r.StrB(k); ok {
				switch k {
				case "trading mode":
					tm := r.MustTradingMode(k)
					m.tm = &tm
				case "auction trigger":
					at := r.MustAuctionTrigger(k)
					m.tr = &at
				case "extension trigger":
					et := r.MustAuctionTrigger(k)
					m.et = &et
				}
				delete(todo, k)
			}
		}
		if len(todo) == 0 {
			return
		}
	}
}

// parses the data, and returns a slice of keys for the values that were provided.
func (m *MappedMD) parseU64(data *godog.Table) []string {
	set := make([]string, 0, len(m.u64Map))
	for _, r := range ParseTable(data) {
		for k, ptr := range m.u64Map {
			if u, ok := r.U64B(k); ok {
				*ptr = u
				set = append(set, k)
				// avoid reassignments in following iterations
				delete(m.u64Map, k)
			}
		}
	}
	return set
}

func (m *MappedMD) parseTimes(data *godog.Table) []string {
	// already set start based off of the value in the map
	// does some trickery WRT auction end time, so we can check if the auction duration is N seconds
	var (
		end   int64
		start = *m.tMap["auction start"]
	)
	set := make([]string, 0, len(m.tMap))
	for _, r := range ParseTable(data) {
		for k, ptr := range m.tMap {
			if i, ok := r.I64B(k); ok {
				if k == "auction end" {
					end = i
					if end < start {
						i = start + int64(time.Duration(end)*time.Second)
					}
				}
				*ptr = i
				set = append(set, k)
				// again: avoid reassignments when parsing the next row
				delete(m.i64Map, k)
			}
		}
	}
	return set
}

func (m *MappedMD) parseUint(data *godog.Table) []string {
	set := make([]string, 0, len(m.uintMap))
	for _, r := range ParseTable(data) {
		for k, ptr := range m.uintMap {
			if i, ok := r.StrB(k); ok {
				n, _ := num.UintFromString(i, 10)
				*ptr = n
				set = append(set, k)
				// again: avoid reassignments when parsing the next row
				delete(m.uintMap, k)
			}
		}
	}
	return set
}

func (m *MappedMD) parseI64(data *godog.Table) []string {
	set := make([]string, 0, len(m.i64Map))
	for _, r := range ParseTable(data) {
		for k, ptr := range m.i64Map {
			if i, ok := r.I64B(k); ok {
				*ptr = i
				set = append(set, k)
				// again: avoid reassignments when parsing the next row
				delete(m.i64Map, k)
			}
		}
	}
	return set
}

func (m *MappedMD) parseStr(data *godog.Table) []string {
	set := make([]string, 0, len(m.strMap))
	for _, r := range ParseTable(data) {
		for k, ptr := range m.strMap {
			if i, ok := r.StrB(k); ok {
				*ptr = i
				set = append(set, k)
				// again: avoid reassignments when parsing the next row
				delete(m.strMap, k)
			}
		}
	}
	return set
}

func mappedMD(md types.MarketData) *MappedMD {
	r := &MappedMD{
		md: md,
	}

	// no need to clone here, the values are already cloned
	r.uintMap = map[string]**num.Uint{
		"mark price":              &(r.md.MarkPrice),
		"best bid price":          &r.md.BestBidPrice,
		"best offer price":        &r.md.BestOfferPrice,
		"best static bid price":   &r.md.BestStaticBidPrice,
		"best static offer price": &r.md.BestStaticOfferPrice,
		"mid price":               &r.md.MidPrice,
		"static mid price":        &r.md.StaticMidPrice,
		"indicative price":        &r.md.IndicativePrice,
	}
	r.u64Map = map[string]*uint64{
		"best bid volume":          &r.md.BestBidVolume,
		"best offer volume":        &r.md.BestOfferVolume,
		"best static bid volume":   &r.md.BestStaticBidVolume,
		"best static offer volume": &r.md.BestStaticOfferVolume,
		"open interest":            &r.md.OpenInterest,
		"indicative volume":        &r.md.IndicativeVolume,
	}
	r.strMap = map[string]*string{
		"target stake":       &r.md.TargetStake,
		"supplied stake":     &r.md.SuppliedStake,
		"market value proxy": &r.md.MarketValueProxy,
		"market":             &r.md.Market, // this is a bit pointless, but might as well add it
	}
	r.tMap = map[string]*int64{
		"timestamp":     &r.md.Timestamp,
		"auction end":   &r.md.AuctionEnd,
		"auction start": &r.md.AuctionStart,
	}
	return r
}

// Error so we print out the wrong matches line by line.
func (e ErrStack) Error() string {
	str := make([]string, 0, len(e))
	for _, v := range e {
		str = append(str, v.Error())
	}
	return strings.Join(str, "\n")
}
