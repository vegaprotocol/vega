package liquidity

import (
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

var DefaultSLAParameters = types.LiquiditySLAParams{
	PriceRange:                  num.MustDecimalFromString("0.05"),
	CommitmentMinTimeFraction:   num.MustDecimalFromString("0.95"),
	PerformanceHysteresisEpochs: 1,
	SlaCompetitionFactor:        num.MustDecimalFromString("0.90"),
}

// ResetSLAEpoch should be called at the beginning of epoch to reset per epoch performance calculations.
// Returns a newly added/amended liquidity provisions (pending provisions are automatically applied and the start of a new epoch).
func (e *Engine) ResetSLAEpoch(
	now time.Time,
	markPrice *num.Uint,
	midPrice *num.Uint,
	positionFactor num.Decimal,
) {
	e.slaEpochStart = now
	if e.auctionState.IsOpeningAuction() {
		return
	}

	for party, commitment := range e.slaPerformance {
		if e.doesLPMeetsCommitment(party, markPrice, midPrice, positionFactor) {
			commitment.start = now
		}

		commitment.s = 0
	}
}

func (e *Engine) EndBlock(markPrice *num.Uint, midPrice *num.Uint, positionFactor num.Decimal) {
	// Check if the k transaction has been processed
	if e.auctionState.IsOpeningAuction() {
		return
	}

	for party, commitment := range e.slaPerformance {
		if meetsCommitment := e.doesLPMeetsCommitment(party, markPrice, midPrice, positionFactor); meetsCommitment {
			// if LP started meeting commitment
			if commitment.start.IsZero() {
				commitment.start = e.timeService.GetTimeNow()
			}
			continue
		}
		// else if LP stopped meeting commitment
		if !commitment.start.IsZero() {
			commitment.s += e.timeService.GetTimeNow().Sub(commitment.start)
			commitment.start = time.Time{}
		}
	}
}

func (e *Engine) calculateCurrentTimeBookFraction(now, start time.Time, s time.Duration) num.Decimal {
	if !start.IsZero() {
		s += now.Sub(start)
	}

	observedEpochLength := now.Sub(e.slaEpochStart)
	lNano := observedEpochLength.Nanoseconds()
	timeBookFraction := num.DecimalZero()
	if lNano > 0 {
		timeBookFraction = num.DecimalFromInt64(s.Nanoseconds()).Div(num.DecimalFromInt64(lNano))
	}

	return timeBookFraction
}

// CalculateSLAPenalties should be called at the and of epoch to calculate SLA penalties based on LP performance in the epoch.
func (e *Engine) CalculateSLAPenalties(now time.Time) SlaPenalties {
	penaltiesPerParty := map[string]*SlaPenalty{}

	// Do not apply any penalties during opening auction
	if e.auctionState.IsOpeningAuction() {
		return SlaPenalties{
			AllPartiesHaveFullFeePenalty: false,
			PenaltiesPerParty:            penaltiesPerParty,
		}
	}

	one := num.DecimalOne()
	partiesWithFullFeePenaltyCount := 0

	for party, commitment := range e.slaPerformance {
		timeBookFraction := e.calculateCurrentTimeBookFraction(now, commitment.start, commitment.s)

		var feePenalty, bondPenalty num.Decimal

		// if LP meets commitment
		// else LP does not meet commitment
		if timeBookFraction.LessThan(e.slaParams.CommitmentMinTimeFraction) {
			feePenalty = one
			bondPenalty = e.calculateBondPenalty(timeBookFraction)
		} else {
			feePenalty = e.calculateCurrentFeePenalty(timeBookFraction)
			bondPenalty = num.DecimalZero()
		}

		penaltiesPerParty[party] = &SlaPenalty{
			Bond: bondPenalty,
			Fee:  e.calculateHysteresisFeePenalty(feePenalty, commitment.previousPenalties.Slice()),
		}

		commitment.previousPenalties.Add(&feePenalty)

		if penaltiesPerParty[party].Fee.Equal(one) {
			partiesWithFullFeePenaltyCount++
		}

		// safe for next epoch stats
		e.slaPerformance[party].lastEpochBondPenalty = penaltiesPerParty[party].Bond.String()
		e.slaPerformance[party].lastEpochFeePenalty = penaltiesPerParty[party].Fee.String()
		e.slaPerformance[party].lastEpochTimeBookFraction = timeBookFraction.String()
	}

	return SlaPenalties{
		AllPartiesHaveFullFeePenalty: partiesWithFullFeePenaltyCount == len(penaltiesPerParty),
		PenaltiesPerParty:            penaltiesPerParty,
	}
}

func (e *Engine) doesLPMeetsCommitment(
	party string,
	markPrice *num.Uint,
	midPrice *num.Uint,
	positionFactor num.Decimal,
) bool {
	lp, ok := e.provisions.Get(party)
	if !ok {
		return false
	}

	var minPrice, maxPrice num.Decimal
	if e.auctionState.InAuction() {
		minPriceFactor := num.Min(e.orderBook.GetLastTradedPrice(), e.orderBook.GetIndicativePrice()).ToDecimal()
		maxPriceFactor := num.Max(e.orderBook.GetLastTradedPrice(), e.orderBook.GetIndicativePrice()).ToDecimal()

		// (1.0-market.liquidity.priceRange) x min(last trade price, indicative uncrossing price)
		minPrice = e.openMinusPriceRange.Mul(minPriceFactor)
		// (1.0+market.liquidity.priceRange) x max(last trade price, indicative uncrossing price)
		maxPrice = e.openPlusPriceRange.Mul(maxPriceFactor)
	} else {
		// if there is no mid price then LP is not meeting their committed volume of notional.
		if midPrice.IsZero() {
			return false
		}
		midD := midPrice.ToDecimal()
		// (1.0 - market.liquidity.priceRange) x mid
		minPrice = e.openMinusPriceRange.Mul(midD)
		// (1.0 + market.liquidity.priceRange) x mid
		maxPrice = e.openPlusPriceRange.Mul(midD)
	}

	notionalVolumeBuys := num.DecimalZero()
	notionalVolumeSells := num.DecimalZero()
	orders := e.getAllActiveOrders(party)

	for _, o := range orders {
		price := o.Price.ToDecimal()
		// this order is in range and does contribute to the volume on notional
		if price.GreaterThanOrEqual(minPrice) && price.LessThanOrEqual(maxPrice) {
			orderVolume := num.UintZero().Mul(markPrice, num.NewUint(o.TrueRemaining())).ToDecimal().Div(positionFactor)

			if o.Side == types.SideSell {
				notionalVolumeSells = notionalVolumeSells.Add(orderVolume)
			} else {
				notionalVolumeBuys = notionalVolumeBuys.Add(orderVolume)
			}
		}
	}

	requiredLiquidity := e.stakeToCcyVolume.Mul(lp.CommitmentAmount.ToDecimal())

	return notionalVolumeBuys.GreaterThanOrEqual(requiredLiquidity) &&
		notionalVolumeSells.GreaterThanOrEqual(requiredLiquidity)
}

func (e *Engine) calculateCurrentFeePenalty(timeBookFraction num.Decimal) num.Decimal {
	one := num.DecimalOne()

	if timeBookFraction.LessThan(e.slaParams.CommitmentMinTimeFraction) {
		return one
	}

	if timeBookFraction.Equal(e.slaParams.CommitmentMinTimeFraction) && timeBookFraction.Equal(one) {
		return num.DecimalZero()
	}

	// p = (1-[timeBookFraction-commitmentMinTimeFraction/1-commitmentMinTimeFraction]) * slaCompetitionFactor
	return one.Sub(
		timeBookFraction.Sub(e.slaParams.CommitmentMinTimeFraction).Div(one.Sub(e.slaParams.CommitmentMinTimeFraction)),
	).Mul(e.slaParams.SlaCompetitionFactor)
}

func (e *Engine) calculateBondPenalty(timeBookFraction num.Decimal) num.Decimal {
	// min(nonPerformanceBondPenaltyMax, nonPerformanceBondPenaltySlope * (1-timeBookFraction/commitmentMinTimeFraction))
	min := num.MinD(
		e.nonPerformanceBondPenaltyMax,
		e.nonPerformanceBondPenaltySlope.Mul(num.DecimalOne().Sub(timeBookFraction.Div(e.slaParams.CommitmentMinTimeFraction))),
	)

	// max(0, min)
	return num.MaxD(num.DecimalZero(), min)
}

func (e *Engine) calculateHysteresisFeePenalty(currentPenalty num.Decimal, previousPenalties []*num.Decimal) num.Decimal {
	one := num.DecimalOne()
	previousPenaltiesCount := num.DecimalZero()
	periodAveragePenalty := num.DecimalZero()

	for _, p := range previousPenalties {
		if p == nil {
			continue
		}

		periodAveragePenalty = periodAveragePenalty.Add(*p)
		previousPenaltiesCount = previousPenaltiesCount.Add(one)
	}

	if previousPenaltiesCount.IsZero() {
		return currentPenalty
	}

	periodAveragePenalty = periodAveragePenalty.Div(previousPenaltiesCount)

	return num.MaxD(currentPenalty, periodAveragePenalty)
}

func (e *Engine) LiquidityProviderSLAStats(now time.Time) []*types.LiquidityProviderSLA {
	stats := make([]*types.LiquidityProviderSLA, 0, len(e.slaPerformance))

	for partyID, commitment := range e.slaPerformance {
		currentTimeBookFraction := e.calculateCurrentTimeBookFraction(now, commitment.start, commitment.s)

		previousPenalties := commitment.previousPenalties.Slice()
		hysteresisPeriodFeePenalties := make([]string, 0, len(previousPenalties))
		for _, penalty := range previousPenalties {
			if penalty == nil {
				continue
			}
			hysteresisPeriodFeePenalties = append(hysteresisPeriodFeePenalties, penalty.String())
		}

		stats = append(stats, &types.LiquidityProviderSLA{
			Party:                            partyID,
			CurrentEpochFractionOfTimeOnBook: currentTimeBookFraction.String(),
			LastEpochFractionOfTimeOnBook:    commitment.lastEpochTimeBookFraction,
			LastEpochFeePenalty:              commitment.lastEpochFeePenalty,
			LastEpochBondPenalty:             commitment.lastEpochBondPenalty,
			HysteresisPeriodFeePenalties:     hysteresisPeriodFeePenalties,
		})
	}

	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Party == stats[j].Party {
			return stats[i].CurrentEpochFractionOfTimeOnBook > stats[j].CurrentEpochFractionOfTimeOnBook
		}
		return stats[i].Party > stats[j].Party
	})

	return stats
}
