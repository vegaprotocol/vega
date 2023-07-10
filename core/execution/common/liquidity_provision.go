// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package common

import (
	"context"
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/fee"
	"code.vegaprotocol.io/vega/core/idgeneration"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/liquidity/v2"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

var ErrCommitmentAmountTooLow = errors.New("commitment amount is too low")

type MarketLiquidity struct {
	log   *logging.Logger
	idGen IDGenerator

	liquidityEngine       LiquidityEngine
	collateral            Collateral
	broker                Broker
	orderBook             liquidity.OrderBook
	equityShares          EquityLikeShares
	marketActivityTracker *MarketActivityTracker
	fee                   *fee.Engine

	marketID string
	asset    string

	priceFactor *num.Uint

	priceRange                num.Decimal
	earlyExitPenalty          num.Decimal
	minLPStakeQuantumMultiple num.Decimal
	feeDistributionTimeStep   time.Duration

	lastFeeDistribution time.Time
}

func NewMarketLiquidity(
	log *logging.Logger,
	liquidityEngine LiquidityEngine,
	collateral Collateral,
	broker Broker,
	orderBook liquidity.OrderBook,
	equityShares EquityLikeShares,
	marketActivityTracker *MarketActivityTracker,
	fee *fee.Engine,
	marketID string,
	asset string,
	minLPStakeQuantumMultiple num.Decimal,
	priceFactor *num.Uint,
	priceRange num.Decimal,
	earlyExitPenalty num.Decimal,
	feeDistributionTimeStep time.Duration,
) *MarketLiquidity {
	ml := &MarketLiquidity{
		log:                       log,
		liquidityEngine:           liquidityEngine,
		collateral:                collateral,
		broker:                    broker,
		orderBook:                 orderBook,
		equityShares:              equityShares,
		marketActivityTracker:     marketActivityTracker,
		fee:                       fee,
		marketID:                  marketID,
		asset:                     asset,
		minLPStakeQuantumMultiple: minLPStakeQuantumMultiple,
		priceFactor:               priceFactor,
		priceRange:                priceRange,
		earlyExitPenalty:          earlyExitPenalty,
		feeDistributionTimeStep:   feeDistributionTimeStep,
	}

	return ml
}

func (m *MarketLiquidity) applyPendingProvisions(
	ctx context.Context,
	now time.Time,
	targetStake *num.Uint,
) map[string]*types.LiquidityProvision {
	provisions := m.liquidityEngine.ProvisionsPerParty()
	pendingProvisions := m.liquidityEngine.PendingProvision()

	zero := num.DecimalZero()

	// totalStake - targetStake
	totalTargetStakeDifference := m.liquidityEngine.CalculateSuppliedStakeWithoutPending().ToDecimal().Sub(targetStake.ToDecimal())
	maxPenaltyFreeReductionAmount := num.MaxD(zero, totalTargetStakeDifference)

	sumOfCommitmentVariations := num.DecimalZero()
	commitmentVariationPerParty := map[string]num.Decimal{}

	for partyID, provision := range provisions {
		acc, err := m.collateral.GetPartyBondAccount(m.marketID, partyID, m.asset)
		if err != nil {
			// the bond account should be definitely there at this point
			m.log.Panic("can not get LP party bond account", logging.Error(err))
		}

		amendment, ok := pendingProvisions[partyID]
		if !ok {
			continue
		}

		// originalCommitment - amendedCommitment
		proposedCommitmentVariation := provision.CommitmentAmount.ToDecimal().Sub(amendment.CommitmentAmount.ToDecimal())

		zero := num.DecimalZero()

		// if commitment is increased, there is not penalty applied
		if proposedCommitmentVariation.GreaterThanOrEqual(zero) {
			return nil
		}

		// min(-proposedCommitmentVariation, bondAccountBalance)
		commitmentVariation := num.MinD(proposedCommitmentVariation.Neg(), acc.Balance.ToDecimal())
		commitmentVariationPerParty[partyID] = commitmentVariation
		sumOfCommitmentVariations.Add(commitmentVariation)
	}

	ledgerMovements := make([]*types.LedgerMovement, 0, len(commitmentVariationPerParty))

	one := num.DecimalOne()

	keys := sortedKeys(commitmentVariationPerParty)
	for _, partyID := range keys {
		commitmentVariation := commitmentVariationPerParty[partyID]
		// (commitmentVariation/sumOfCommitmentVariations) * maxPenaltyFreeReductionAmount
		partyMaxPenaltyFreeReductionAmount := commitmentVariation.Div(sumOfCommitmentVariations).
			Mul(maxPenaltyFreeReductionAmount)

		// transfer entire decreased commitment to their general account, no penalty will be applied
		if commitmentVariation.LessThanOrEqual(partyMaxPenaltyFreeReductionAmount) {
			commitmentVariationU, _ := num.UintFromDecimal(commitmentVariation)
			if commitmentVariationU.IsZero() {
				continue
			}

			transfer := m.NewTransfer(partyID, types.TransferTypeBondHigh, commitmentVariationU)
			bondLedgerMovement, err := m.collateral.BondSpotUpdate(ctx, m.marketID, transfer)
			if err != nil {
				m.log.Panic("failed to apply SLA penalties to bond account", logging.Error(err))
			}

			ledgerMovements = append(ledgerMovements, bondLedgerMovement)
			continue
		}

		partyMaxPenaltyFreeReductionAmountU, _ := num.UintFromDecimal(partyMaxPenaltyFreeReductionAmount)

		transfer := m.NewTransfer(partyID, types.TransferTypeBondHigh, partyMaxPenaltyFreeReductionAmountU)
		bondLedgerMovement, err := m.collateral.BondSpotUpdate(ctx, m.marketID, transfer)
		if err != nil {
			m.log.Panic("failed to apply SLA penalties to bond account", logging.Error(err))
		}

		ledgerMovements = append(ledgerMovements, bondLedgerMovement)

		penaltyIncurringReductionAmount := commitmentVariation.Sub(partyMaxPenaltyFreeReductionAmount)

		// transfer to general account
		freeAmount := one.Sub(m.earlyExitPenalty).Mul(penaltyIncurringReductionAmount)
		freeAmountU, _ := num.UintFromDecimal(freeAmount)

		transfer = m.NewTransfer(partyID, types.TransferTypeBondHigh, freeAmountU)
		bondLedgerMovement, err = m.collateral.BondSpotUpdate(ctx, m.marketID, transfer)
		if err != nil {
			m.log.Panic("failed to apply SLA penalties to bond account", logging.Error(err))
		}

		ledgerMovements = append(ledgerMovements, bondLedgerMovement)

		slashingAmount := m.earlyExitPenalty.Mul(penaltyIncurringReductionAmount)
		slashingAmountU, _ := num.UintFromDecimal(slashingAmount)

		transfer = m.NewTransfer(partyID, types.TransferTypeBondSlashing, slashingAmountU)
		bondLedgerMovement, err = m.collateral.BondSpotUpdate(ctx, m.marketID, transfer)
		if err != nil {
			m.log.Panic("failed to apply SLA penalties to bond account", logging.Error(err))
		}

		ledgerMovements = append(ledgerMovements, bondLedgerMovement)
	}

	if len(ledgerMovements) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, ledgerMovements))
	}

	return m.liquidityEngine.ApplyPendingProvisions(ctx, now)
}

func (m *MarketLiquidity) syncPartyCommitmentWithBondAccount(appliedLiquidityProvisions map[string]*types.LiquidityProvision) {
	if len(appliedLiquidityProvisions) == 0 {
		appliedLiquidityProvisions = map[string]*types.LiquidityProvision{}
	}

	for partyID, provision := range m.liquidityEngine.ProvisionsPerParty() {
		acc, err := m.collateral.GetPartyBondAccount(m.marketID, partyID, m.asset)
		if err != nil {
			// the bond account should be definitely there at this point
			m.log.Panic("can not get LP party bond account", logging.Error(err))
		}

		// lp provision and bond account are in sync, no need to change
		if provision.CommitmentAmount.EQ(acc.Balance) {
			continue
		}

		updatedProvision, err := m.liquidityEngine.UpdatePartyCommitment(partyID, acc.Balance)
		if err != nil {
			m.log.Panic("failed to update party commitment", logging.Error(err))
		}
		appliedLiquidityProvisions[partyID] = updatedProvision
	}

	for party, provision := range appliedLiquidityProvisions {
		// now we can setup our party stake to calculate equities
		m.equityShares.SetPartyStake(party, provision.CommitmentAmount.Clone())
		// force update of shares so they are updated for all
		_ = m.equityShares.AllShares()
	}
}

func (m *MarketLiquidity) OnEpochStart(ctx context.Context, now time.Time, markPrice, targetStake *num.Uint, positionFactor num.Decimal) {
	m.liquidityEngine.ResetSLAEpoch(now, markPrice, positionFactor)

	appliedProvisions := m.applyPendingProvisions(ctx, now, targetStake)
	m.syncPartyCommitmentWithBondAccount(appliedProvisions)
}

func (m *MarketLiquidity) OnEpochEnd(ctx context.Context, t time.Time) {
	penalties := m.liquidityEngine.CalculateSLAPenalties(t)
	m.distributeFeesBonusesAndApplyPenalties(ctx, penalties)
}

// lp -> general per market fee account.
func (m *MarketLiquidity) OnTick(ctx context.Context, t time.Time) {
	// distribute liquidity fees each feeDistributionTimeStep
	if m.readyForFeesAllocation(t) {
		if err := m.allocateFees(ctx); err != nil {
			m.log.Panic("liquidity fee distribution error", logging.Error(err))
		}

		// reset next distribution period
		m.liquidityEngine.ResetAverageLiquidityScores()
		m.lastFeeDistribution = t
		return
	}

	m.updateLiquidityScores()
}

func (m *MarketLiquidity) EndBlock(markPrice, midPrice *num.Uint, positionFactor num.Decimal) {
	m.liquidityEngine.EndBlock(markPrice, midPrice, positionFactor)
}

func (m *MarketLiquidity) updateLiquidityScores() {
	minLpPrice, maxLpPrice, err := m.validOrdersPriceRange()
	if err != nil {
		m.log.Debug("liquidity score update error", logging.Error(err))
		return
	}
	bestBid, bestAsk, err := m.getBestStaticPricesDecimal()
	if err != nil {
		m.log.Debug("liquidity score update error", logging.Error(err))
		return
	}

	m.liquidityEngine.UpdateAverageLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
}

func (m *MarketLiquidity) getBestStaticPricesDecimal() (bid, ask num.Decimal, err error) {
	bid = num.DecimalZero()
	ask = num.DecimalZero()

	binUint, err := m.orderBook.GetBestStaticBidPrice()
	if err != nil {
		return
	}
	bid = binUint.ToDecimal()

	askUint, err := m.orderBook.GetBestStaticAskPrice()
	if err != nil {
		return
	}
	ask = askUint.ToDecimal()

	return bid, ask, nil
}

// updateSharesWithLiquidityScores multiplies each LP i share with their score and divides all LP i share with total shares amount.
func (m *MarketLiquidity) updateSharesWithLiquidityScores(shares, scores map[string]num.Decimal) map[string]num.Decimal {
	total := num.DecimalZero()

	for partyID, share := range shares {
		score, ok := scores[partyID]
		if !ok {
			continue
		}

		shares[partyID] = share.Mul(score)
		total = total.Add(shares[partyID])
	}

	// normalize - share i / total
	if !total.IsZero() {
		for k, v := range shares {
			shares[k] = v.Div(total)
		}
	}

	return shares
}

func (m *MarketLiquidity) canSubmitCommitment(marketState types.MarketState) bool {
	switch marketState {
	case types.MarketStateActive, types.MarketStatePending, types.MarketStateSuspended, types.MarketStateProposed:
		return true
	}

	return false
}

// SubmitLiquidityProvision forwards a LiquidityProvisionSubmission to the Liquidity Engine.
func (m *MarketLiquidity) SubmitLiquidityProvision(
	ctx context.Context,
	sub *types.LiquidityProvisionSubmission,
	party string,
	deterministicID string,
	marketState types.MarketState,
) (err error) {
	m.idGen = idgeneration.New(deterministicID)
	defer func() { m.idGen = nil }()

	if !m.canSubmitCommitment(marketState) {
		return ErrCommitmentSubmissionNotAllowed
	}

	if err := m.ensureMinCommitmentAmount(sub.CommitmentAmount); err != nil {
		return err
	}

	submittedImmediately, err := m.liquidityEngine.SubmitLiquidityProvision(ctx, sub, party, m.idGen)
	if err != nil {
		return err
	}

	if err := m.makePerPartyAccountsAndTransfers(ctx, party, sub.CommitmentAmount); err != nil {
		if newErr := m.liquidityEngine.RejectLiquidityProvision(ctx, party); newErr != nil {
			m.log.Debug("unable to submit cancel liquidity provision submission",
				logging.String("party", party),
				logging.String("id", deterministicID),
				logging.Error(newErr))

			err = fmt.Errorf("%v, %w", err, newErr)
		}

		return err
	}

	if submittedImmediately {
		// now we can setup our party stake to calculate equities
		m.equityShares.SetPartyStake(party, sub.CommitmentAmount.Clone())
		// force update of shares so they are updated for all
		_ = m.equityShares.AllShares()
	}

	return nil
}

// makePerPartyAccountsAndTransfers create a party specific per market accounts for bond, margin and fee.
// It also transfers required commitment amount to per market bond account.
func (m *MarketLiquidity) makePerPartyAccountsAndTransfers(ctx context.Context, party string, commitmentAmount *num.Uint) error {
	bondAcc, err := m.collateral.GetOrCreatePartyBondAccount(ctx, party, m.marketID, m.asset)
	if err != nil {
		return err
	}

	_, err = m.collateral.CreatePartyMarginAccount(ctx, party, m.marketID, m.asset)
	if err != nil {
		return err
	}

	_, err = m.collateral.GetOrCreatePartyLiquidityFeeAccount(ctx, party, m.marketID, m.asset)
	if err != nil {
		return err
	}

	// calculates the amount that needs to be moved into the bond account
	amount, neg := num.UintZero().Delta(commitmentAmount, bondAcc.Balance)
	ty := types.TransferTypeBondLow
	if neg {
		ty = types.TransferTypeBondHigh
	}
	transfer := &types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Amount: amount.Clone(),
			Asset:  m.asset,
		},
		Type:      ty,
		MinAmount: amount.Clone(),
	}

	tresp, err := m.collateral.BondSpotUpdate(ctx, m.marketID, transfer)
	if err != nil {
		m.log.Debug("bond update error", logging.Error(err))
		return err
	}
	m.broker.Send(events.NewLedgerMovements(ctx, []*types.LedgerMovement{tresp}))

	return nil
}

// AmendLiquidityProvision forwards a LiquidityProvisionAmendment to the Liquidity Engine.
func (m *MarketLiquidity) AmendLiquidityProvision(
	ctx context.Context,
	lpa *types.LiquidityProvisionAmendment,
	party string,
	deterministicID string,
	marketState types.MarketState,
) error {
	m.idGen = idgeneration.New(deterministicID)
	defer func() { m.idGen = nil }()

	if !m.canSubmitCommitment(marketState) {
		return ErrCommitmentSubmissionNotAllowed
	}

	if err := m.liquidityEngine.ValidateLiquidityProvisionAmendment(lpa); err != nil {
		return err
	}

	if lpa.CommitmentAmount != nil {
		if err := m.ensureMinCommitmentAmount(lpa.CommitmentAmount); err != nil {
			return err
		}
	}

	if !m.liquidityEngine.IsLiquidityProvider(party) {
		return ErrPartyNotLiquidityProvider
	}

	lp := m.liquidityEngine.LiquidityProvisionByPartyID(party)
	if lp == nil {
		return fmt.Errorf("cannot edit liquidity provision from a non liquidity provider party (%v)", party)
	}

	// If commitment amount is not provided we keep the same
	if lpa.CommitmentAmount == nil || lpa.CommitmentAmount.IsZero() {
		lpa.CommitmentAmount = lp.CommitmentAmount
	}

	// If commitment amount is not provided we keep the same
	if lpa.Fee.IsZero() {
		lpa.Fee = lp.Fee
	}

	// If commitment amount is not provided we keep the same
	if lpa.Reference == "" {
		lpa.Reference = lp.Reference
	}

	// if pending commitment is being decreased then release the bond collateral
	existingAmendment := m.liquidityEngine.PendingProvisionByPartyID(party)
	if existingAmendment != nil && !lpa.CommitmentAmount.IsZero() && lpa.CommitmentAmount.LT(existingAmendment.CommitmentAmount) {
		amountToRelease := num.UintZero().Sub(existingAmendment.CommitmentAmount, lpa.CommitmentAmount)
		_, err := m.releasePendingBondCollateral(ctx, amountToRelease, party)
		if err != nil {
			m.log.Debug("could not submit update bond for lp amendment",
				logging.PartyID(party),
				logging.MarketID(m.marketID),
				logging.Error(err))

			return err
		}
	}

	proposedCommitmentVariation := lp.CommitmentAmount.ToDecimal().Sub(lpa.CommitmentAmount.ToDecimal())

	// if increase commitment transfer funds to bond account
	if proposedCommitmentVariation.GreaterThan(num.DecimalZero()) {
		_, err := m.ensureAndTransferCollateral(ctx, lpa.CommitmentAmount, party)
		if err != nil {
			m.log.Debug("could not submit update bond for lp amendment",
				logging.PartyID(party),
				logging.MarketID(m.marketID),
				logging.Error(err))

			return err
		}
	}

	err := m.liquidityEngine.AmendLiquidityProvision(ctx, lpa, party)
	if err != nil {
		m.log.Panic("error while amending liquidity provision, this should not happen at this point, the LP was validated earlier",
			logging.Error(err))
	}

	return nil
}

func (m *MarketLiquidity) CancelLiquidityProvision(ctx context.Context, party string) error {
	return m.liquidityEngine.CancelLiquidityProvision(ctx, party)
}

func (m *MarketLiquidity) StopAllLiquidityProvision(ctx context.Context) {
	for _, p := range m.liquidityEngine.ProvisionsPerParty().Slice() {
		m.liquidityEngine.StopLiquidityProvision(ctx, p.Party)
	}
}

// checks that party has enough collateral to support the commitment increase.
func (m *MarketLiquidity) ensureAndTransferCollateral(
	ctx context.Context, commitmentAmount *num.Uint, party string,
) (*types.Transfer, error) {
	bondAcc, err := m.collateral.GetOrCreatePartyBondAccount(
		ctx, party, m.marketID, m.asset)
	if err != nil {
		return nil, err
	}

	// first check if there's enough funds in the gen + bond
	// account to cover the new commitment
	if !m.collateral.CanCoverBond(m.marketID, party, m.asset, commitmentAmount.Clone()) {
		return nil, ErrCommitmentSubmissionNotAllowed
	}

	// build our transfer to be sent to collateral
	amount, neg := num.UintZero().Delta(commitmentAmount, bondAcc.Balance)
	ty := types.TransferTypeBondLow
	if neg {
		ty = types.TransferTypeBondHigh
	}
	transfer := &types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Amount: amount,
			Asset:  m.asset,
		},
		Type:      ty,
		MinAmount: amount.Clone(),
	}

	// move our bond
	tresp, err := m.collateral.BondSpotUpdate(ctx, m.marketID, transfer)
	if err != nil {
		return nil, err
	}
	m.broker.Send(events.NewLedgerMovements(
		ctx, []*types.LedgerMovement{tresp}))

	// now we will use the actual transfer as a rollback later on eventually
	// so let's just change from HIGH to LOW and inverse
	if transfer.Type == types.TransferTypeBondHigh {
		transfer.Type = types.TransferTypeBondLow
	} else {
		transfer.Type = types.TransferTypeBondHigh
	}

	return transfer, nil
}

// releasePendingCollateral releases pending amount collateral from bond to general account.
func (m *MarketLiquidity) releasePendingBondCollateral(
	ctx context.Context, releaseAmount *num.Uint, party string,
) (*types.Transfer, error) {
	transfer := &types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Amount: releaseAmount,
			Asset:  m.asset,
		},
		Type:      types.TransferTypeBondHigh,
		MinAmount: releaseAmount.Clone(),
	}

	ledgerMovement, err := m.collateral.BondSpotUpdate(ctx, m.marketID, transfer)
	if err != nil {
		return nil, err
	}
	m.broker.Send(events.NewLedgerMovements(
		ctx, []*types.LedgerMovement{ledgerMovement}))

	// now we will use the actual transfer as a rollback later on eventually
	transfer.Type = types.TransferTypeBondLow

	return transfer, nil
}

func (m *MarketLiquidity) ensureMinCommitmentAmount(amount *num.Uint) error {
	quantum, err := m.collateral.GetAssetQuantum(m.asset)
	if err != nil {
		m.log.Panic("could not get quantum for asset, this should never happen",
			logging.AssetID(m.asset),
			logging.Error(err),
		)
	}
	minStake := quantum.Mul(m.minLPStakeQuantumMultiple)
	if amount.ToDecimal().LessThan(minStake) {
		return ErrCommitmentAmountTooLow
	}

	return nil
}

// validOrdersPriceRange returns min and max valid prices range for LP orders.
func (m *MarketLiquidity) validOrdersPriceRange() (*num.Uint, *num.Uint, error) {
	bestBid, err := m.orderBook.GetBestStaticBidPrice()
	if err != nil {
		return num.UintOne(), num.MaxUint(), err
	}

	bestAsk, err := m.orderBook.GetBestStaticAskPrice()
	if err != nil {
		return num.UintOne(), num.MaxUint(), err
	}

	// (bestBid + bestAsk) / 2
	midPrice := bestBid.ToDecimal().Add(bestAsk.ToDecimal()).Div(num.DecimalFromFloat(2))
	// (1 - priceRange) * midPrice
	lowerBoundPriceD := num.DecimalOne().Sub(m.priceRange).Mul(midPrice)
	// (1 + priceRange) * midPrice
	upperBoundPriceD := num.DecimalOne().Add(m.priceRange).Mul(midPrice)

	priceFactor := m.priceFactor.ToDecimal()

	// ceil lower bound
	ceiledLowerBound, rL := lowerBoundPriceD.QuoRem(priceFactor, int32(0))
	if !rL.IsZero() {
		ceiledLowerBound = ceiledLowerBound.Add(num.DecimalOne())
	}
	lowerBoundPriceD = ceiledLowerBound.Mul(priceFactor)

	// floor upper bound
	flooredUpperBound, _ := upperBoundPriceD.QuoRem(priceFactor, int32(0))
	upperBoundPriceD = flooredUpperBound.Mul(priceFactor)

	lowerBound, _ := num.UintFromDecimal(lowerBoundPriceD)
	upperBound, _ := num.UintFromDecimal(upperBoundPriceD)

	// floor at 1 to avoid non-positive value
	if lowerBound.IsNegative() || lowerBound.IsZero() {
		lowerBound = m.priceFactor
	}

	if lowerBound.GTE(upperBound) {
		// if we ended up with overlapping upper and lower bound we set the upper bound to lower bound plus one tick.
		upperBound = upperBound.Add(lowerBound, m.priceFactor)
	}

	// we can't have lower bound >= best static ask as then a buy order with that price would trade on entry
	// so place it one tick to the left
	if lowerBound.GTE(bestAsk) {
		lowerBound = num.UintZero().Sub(bestAsk, m.priceFactor)
	}

	// we can't have upper bound <= best static bid as then a sell order with that price would trade on entry
	// so place it one tick to the right
	if upperBound.LTE(bestAsk) {
		upperBound = num.UintZero().Add(bestAsk, m.priceFactor)
	}

	return lowerBound, upperBound, nil
}

func (m *MarketLiquidity) UpdateMarketConfig(risk liquidity.RiskModel, monitor PriceMonitor, slaParams *types.LiquiditySLAParams) {
	m.priceRange = slaParams.PriceRange
	m.feeDistributionTimeStep = slaParams.ProvidersFeeCalculationTimeStep
	m.liquidityEngine.UpdateMarketConfig(risk, monitor, slaParams)
}

func (m *MarketLiquidity) OnEarlyExitPenalty(earlyExitPenalty num.Decimal) {
	m.earlyExitPenalty = earlyExitPenalty
}

func (m *MarketLiquidity) OnMinLPStakeQuantumMultiple(minLPStakeQuantumMultiple num.Decimal) {
	m.minLPStakeQuantumMultiple = minLPStakeQuantumMultiple
}

func (m *MarketLiquidity) OnMinProbabilityOfTradingLPOrdersUpdate(v num.Decimal) {
	m.liquidityEngine.OnMinProbabilityOfTradingLPOrdersUpdate(v)
}

func (m *MarketLiquidity) OnProbabilityOfTradingTauScalingUpdate(v num.Decimal) {
	m.liquidityEngine.OnProbabilityOfTradingTauScalingUpdate(v)
}

func (m *MarketLiquidity) OnMaximumLiquidityFeeFactorLevelUpdate(f num.Decimal) {
	m.liquidityEngine.OnMaximumLiquidityFeeFactorLevelUpdate(f)
}

func (m *MarketLiquidity) OnStakeToCcyVolumeUpdate(stakeToCcyVolume num.Decimal) {
	m.liquidityEngine.OnStakeToCcyVolumeUpdate(stakeToCcyVolume)
}

func (m *MarketLiquidity) OnNonPerformanceBondPenaltySlopeUpdate(nonPerformanceBondPenaltySlope num.Decimal) {
	m.liquidityEngine.OnNonPerformanceBondPenaltySlopeUpdate(nonPerformanceBondPenaltySlope)
}

func (m *MarketLiquidity) OnNonPerformanceBondPenaltyMaxUpdate(nonPerformanceBondPenaltyMax num.Decimal) {
	m.liquidityEngine.OnNonPerformanceBondPenaltyMaxUpdate(nonPerformanceBondPenaltyMax)
}
