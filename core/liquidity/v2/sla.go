package liquidity

import (
	"encoding/binary"
	"math/rand"
	"time"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
)

// TODO karel - real Tendermint txs should be used.
type TX struct {
	ID string
}

func (t TX) Hash() []byte {
	return crypto.Hash([]byte(t.ID))
}

// ResetSLAEpoch should be called at the beginning of epoch to reset per epoch performance calculations.
func (e *Engine) ResetSLAEpoch(now time.Time) {
	for party, commitment := range e.slaPerformance {
		if e.doesLPMeetsCommitment(party) {
			commitment.start = now
		}

		commitment.s = 0
	}

	e.slaEpochStart = now
}

func (e *Engine) BeginBlock(txs []TX) {
	e.kSla = e.GenerateKSla(txs)
}

func (e *Engine) TxProcessed(txCount int) {
	// Check if the k transaction has been processed
	if e.kSla != txCount {
		return
	}

	for party, commitment := range e.slaPerformance {
		meetsCommitment := e.doesLPMeetsCommitment(party)

		// if LP started meeting commitment
		if meetsCommitment {
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

// CalculateSLAPenalties should be called at the and of epoch to calculate SLA penalties based on LP performance in the epoch.
func (e *Engine) CalculateSLAPenalties(now time.Time) {
	observedEpochLength := now.Sub(e.slaEpochStart)

	penaltiesPerParty := map[string]*SlaPenalty{}

	for party, commitment := range e.slaPerformance {
		if !commitment.start.IsZero() {
			commitment.s += now.Sub(commitment.start)
		}

		s := num.DecimalFromInt64(commitment.s.Nanoseconds())
		observedEpochLengthD := num.DecimalFromInt64(observedEpochLength.Nanoseconds())
		timeBookFraction := s.Div(observedEpochLengthD)

		var feePenalty, bondPenalty num.Decimal

		// if LP meets commitment
		// else LP does not meet commitment
		if timeBookFraction.LessThan(e.commitmentMinTimeFraction) {
			feePenalty = num.DecimalOne()
			bondPenalty = e.calculateBondPenalty(timeBookFraction)
		} else {
			feePenalty = e.calculateCurrentFeePenalty(timeBookFraction)
			bondPenalty = num.DecimalZero()
		}

		commitment.allEpochsPenalties = append(commitment.allEpochsPenalties, feePenalty)
		previousPenalties := commitment.allEpochsPenalties[:len(commitment.allEpochsPenalties)-1]

		penaltiesPerParty[party] = &SlaPenalty{
			Bond: bondPenalty,
			Fee:  e.calculateHysteresisFeePenalty(feePenalty, previousPenalties),
		}
	}

	e.slaPenalties = penaltiesPerParty
}

func (e *Engine) GetSLAPenalties() map[string]*SlaPenalty {
	return e.slaPenalties
}

func (e *Engine) getMidPrice() (*num.Uint, error) {
	bestBid, err := e.orderBook.GetBestStaticBidPrice()
	if err != nil {
		return nil, err
	}

	bestAsk, err := e.orderBook.GetBestStaticAskPrice()
	if err != nil {
		return nil, err
	}

	two := num.NewUint(2)
	midPrice := num.UintZero()
	if !bestBid.IsZero() && !bestAsk.IsZero() {
		midPrice = midPrice.Div(num.Sum(bestBid, bestAsk), two)
	}

	return midPrice, nil
}

func (e *Engine) doesLPMeetsCommitment(party string) bool {
	lp, ok := e.provisions.Get(party)
	if !ok {
		return false
	}

	one := num.DecimalOne()

	var minPrice, maxPrice num.Decimal
	if e.auctionState.InAuction() {
		minPriceFactor := num.Min(e.orderBook.GetLastTradedPrice(), e.orderBook.GetIndicativePrice()).ToDecimal()
		maxPriceFactor := num.Max(e.orderBook.GetLastTradedPrice(), e.orderBook.GetIndicativePrice()).ToDecimal()

		// (1.0-market.liquidity.priceRange) x min(last trade price, indicative uncrossing price)
		minPrice = one.Sub(e.priceRange).Mul(minPriceFactor)
		// (1.0+market.liquidity.priceRange) x max(last trade price, indicative uncrossing price)
		maxPrice = one.Add(e.priceRange).Mul(maxPriceFactor)
	} else {
		mid, err := e.getMidPrice()
		// if there is no mid price then LP is not meeting their committed volume of notional.
		if err != nil || mid.IsZero() {
			return false
		}

		midD := mid.ToDecimal()
		// (1.0 - market.liquidity.priceRange) x mid
		minPrice = one.Sub(e.priceRange).Mul(midD)
		// (1.0 + market.liquidity.priceRange) x mid
		maxPrice = one.Add(e.priceRange).Mul(midD)
	}

	notionalVolume := num.DecimalZero()
	orders := e.getAllActiveOrders(party)

	for _, o := range orders {
		price := o.Price.ToDecimal()
		// this order is in range and does contribute to the volume on notional
		if price.GreaterThanOrEqual(minPrice) && price.LessThanOrEqual(maxPrice) {
			notionalVolume = notionalVolume.Add(price)
		}
	}

	requiredLiquidity := e.stakeToCcyVolume.Mul(lp.CommitmentAmount.ToDecimal())
	return notionalVolume.GreaterThanOrEqual(requiredLiquidity)
}

func (e *Engine) calculateCurrentFeePenalty(timeBookFraction num.Decimal) num.Decimal {
	one := num.DecimalOne()

	// p = (1-[timeBookFraction-commitmentMinTimeFraction/1-commitmentMinTimeFraction]) * slaCompetitionFactor
	return one.Sub(
		timeBookFraction.Sub(e.commitmentMinTimeFraction).Div(one.Sub(e.commitmentMinTimeFraction)),
	).Mul(e.slaCompetitionFactor)
}

func (e *Engine) calculateBondPenalty(timeBookFraction num.Decimal) num.Decimal {
	// min(nonPerformanceBondPenaltyMax, nonPerformanceBondPenaltySlope * (1-timeBookFraction/commitmentMinTimeFraction))
	min := num.MinD(
		e.nonPerformanceBondPenaltyMax,
		e.nonPerformanceBondPenaltySlope.Mul(num.DecimalOne().Sub(timeBookFraction.Div(e.commitmentMinTimeFraction))),
	)

	// max(0, min)
	return num.MaxD(num.DecimalZero(), min)
}

func (e *Engine) calculateHysteresisFeePenalty(currentPenalty num.Decimal, previousPenalties []num.Decimal) num.Decimal {
	l := len(previousPenalties)
	if l < 1 {
		return currentPenalty
	}

	// Select window windowStart for hysteresis period
	windowStart := l - int(e.performanceHysteresisEpochs)
	if windowStart < 0 {
		windowStart = 0
	}

	periodAveragePenalty := num.DecimalZero()
	for _, p := range previousPenalties[windowStart:] {
		periodAveragePenalty = periodAveragePenalty.Add(p)
	}

	performanceHysteresisEpochsD := num.NewDecimalFromFloat(float64(e.performanceHysteresisEpochs))

	periodAveragePenalty = periodAveragePenalty.Div(performanceHysteresisEpochsD)
	return num.MaxD(currentPenalty, periodAveragePenalty)
}

func (e *Engine) GenerateKSla(txs []TX) int {
	bytes := []byte{}
	for _, tx := range txs {
		bytes = append(bytes, tx.Hash()...)
	}

	hash := crypto.Hash(bytes)
	seed := binary.BigEndian.Uint64(hash)

	rand.Seed(int64(seed))

	min := 1
	max := len(txs)
	return rand.Intn(max-min+1) + min
}
