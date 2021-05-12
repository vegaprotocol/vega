package steps

import (
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/execution"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/cucumber/godog/gherkin"
)

type MappedMD struct {
	md     types.MarketData
	u64Map map[string]*uint64
	strMap map[string]*string
	i64Map map[string]*int64
	tm     *types.Market_TradingMode
	tr     *types.AuctionTrigger
}

type ErrStack []error

func TheMarketDataShouldBe(engine *execution.Engine, mID string, data *gherkin.DataTable) error {
	actual, err := engine.GetMarketData(mID)
	if err != nil {
		return err
	}
	// create a copy (deep copy), override the values we've gotten with those from the table so we can compare the objects
	expect := mappedMD(actual)
	// special fields first, these need to be compared manually
	u64Set := expect.parseU64(data)
	i64Set := expect.parseI64(data)
	strSet := expect.parseStr(data)
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
	errs := make([]error, 0, len(u64Set)+len(i64Set)+len(strSet)+2)
	if expect.tm != nil && *expect.tm != expect.md.MarketTradingMode {
		errs = append(errs, fmt.Errorf("expected '%s' trading mode, instead got '%s'", *expect.tm, expect.md.MarketTradingMode))
	}
	if expect.tr != nil && *expect.tr != expect.md.Trigger {
		errs = append(errs, fmt.Errorf("expected '%s' auction trigger, instead got '%s'", *expect.tr, expect.md.Trigger))
	}
	// compare uint64
	for _, u := range u64Set {
		e, g := cmp.u64Map[u], parsed.u64Map[u] // get pointers to both fields
		if *e != *g {
			errs = append(errs, fmt.Errorf("expected '%d' for %s, instead got '%d'", e, u, g))
		}
	}
	// compare int64
	for _, i := range i64Set {
		e, g := cmp.i64Map[i], parsed.i64Map[i]
		if *e != *g {
			errs = append(errs, fmt.Errorf("expected '%d' for %s, instead fot '%d'", e, i, g))
		}
	}
	// compare strings
	for _, s := range strSet {
		e, g := cmp.strMap[s], parsed.strMap[s]
		if *e != *g {
			errs = append(errs, fmt.Errorf("expected '%s' for %s, instead got '%s'", *e, s, *g))
		}
	}
	// wrap all errors in a single error type for complete information
	if len(errs) > 0 {
		return ErrStack(errs)
	}
	// compare special fields (trading mode and auction trigger)
	return nil
}

func getPriceBounds(data *gherkin.DataTable) (ret []*types.PriceMonitoringBounds) {
	for _, row := range TableWrapper(*data).Parse() {
		h := row.I64("horizon")
		if h == 0 {
			continue
		}
		expected := &types.PriceMonitoringBounds{
			MinValidPrice: row.MustU64("min bound"),
			MaxValidPrice: row.MustU64("max bound"),
			Trigger: &types.PriceMonitoringTrigger{
				Horizon: h,
			},
		}
		ret = append(ret, expected)
	}
	return ret
}

func getLPFeeShare(data *gherkin.DataTable) (ret []*types.LiquidityProviderFeeShare) {
	for _, r := range TableWrapper(*data).Parse() {
		avg := r.Str("average entry valuation")
		if avg == "" {
			continue
		}
		ret = append(ret, &types.LiquidityProviderFeeShare{
			Party:                 r.MustStr("party"),
			EquityLikeShare:       r.MustStr("equity share"),
			AverageEntryValuation: avg,
		})
	}
	return ret
}

func (m *MappedMD) parseSpecial(data *gherkin.DataTable) {
	todo := map[string]struct{}{
		"trading mode":    {},
		"auction trigger": {},
	}
	for _, r := range TableWrapper(*data).Parse() {
		for k := range todo {
			if _, ok := r.StrB(k); ok {
				switch k {
				case "trading mode":
					tm := r.MustTradingMode(k)
					m.tm = &tm
				case "auction trigger":
					at := r.MustAuctionTrigger(k)
					m.tr = &at
				}
				delete(todo, k)
			}
		}
		if len(todo) == 0 {
			return
		}
	}
}

// parses the data, and returns a slice of keys for the values that were provided
func (m *MappedMD) parseU64(data *gherkin.DataTable) []string {
	set := make([]string, 0, len(m.u64Map))
	for _, r := range TableWrapper(*data).Parse() {
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

func (m *MappedMD) parseI64(data *gherkin.DataTable) []string {
	set := make([]string, 0, len(m.i64Map))
	for _, r := range TableWrapper(*data).Parse() {
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

func (m *MappedMD) parseStr(data *gherkin.DataTable) []string {
	set := make([]string, 0, len(m.strMap))
	for _, r := range TableWrapper(*data).Parse() {
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
	r.u64Map = map[string]*uint64{
		"mark price":               &r.md.MarkPrice,
		"best bid price":           &r.md.BestBidPrice,
		"best bid volume":          &r.md.BestBidVolume,
		"best offer price":         &r.md.BestOfferPrice,
		"best offer volume":        &r.md.BestOfferVolume,
		"best static bid price":    &r.md.BestStaticBidPrice,
		"best static bid volume":   &r.md.BestStaticBidVolume,
		"best static offer price":  &r.md.BestStaticOfferPrice,
		"best static offer volume": &r.md.BestStaticOfferVolume,
		"mid price":                &r.md.MidPrice,
		"static mid price":         &r.md.StaticMidPrice,
		"open interest":            &r.md.OpenInterest,
		"indicative price":         &r.md.IndicativePrice,
		"indicative volume":        &r.md.IndicativeVolume,
	}
	r.strMap = map[string]*string{
		"target stake":       &r.md.TargetStake,
		"supplied stake":     &r.md.SuppliedStake,
		"market value proxy": &r.md.MarketValueProxy,
		"market":             &r.md.Market, // this is a bit pointless, but might as well add it
	}
	r.i64Map = map[string]*int64{
		"timestamp":     &r.md.Timestamp,
		"auction end":   &r.md.AuctionEnd,
		"auction start": &r.md.AuctionStart,
	}
	return r
}

// Error so we print out the wrong matches line by line
func (e ErrStack) Error() string {
	str := make([]string, 0, len(e))
	for _, v := range e {
		str = append(str, v.Error())
	}
	return strings.Join(str, "\n")
}
