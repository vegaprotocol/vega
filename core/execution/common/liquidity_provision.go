// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package common

import (
	"context"
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/fee"
	"code.vegaprotocol.io/vega/core/idgeneration"
	"code.vegaprotocol.io/vega/core/liquidity/v2"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	"golang.org/x/exp/maps"
)

var ErrCommitmentAmountTooLow = errors.New("commitment amount is too low")

type marketType int

const (
	FutureMarketType marketType = iota
	SpotMarketType
)

type MarketLiquidity struct {
	log   *logging.Logger
	idGen IDGenerator

	liquidityEngine       LiquidityEngine
	collateral            Collateral
	broker                Broker
	orderBook             liquidity.OrderBook
	equityShares          EquityLikeShares
	amm                   AMM
	marketActivityTracker *MarketActivityTracker
	fee                   *fee.Engine

	marketType marketType
	marketID   string
	asset      string

	priceFactor *num.Uint

	priceRange                num.Decimal
	earlyExitPenalty          num.Decimal
	minLPStakeQuantumMultiple num.Decimal

	bondPenaltyFactor num.Decimal
	elsFeeFactor      num.Decimal
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
	marketType marketType,
	marketID string,
	asset string,
	priceFactor *num.Uint,
	priceRange num.Decimal,
	amm AMM,
) *MarketLiquidity {
	ml := &MarketLiquidity{
		log:                   log,
		liquidityEngine:       liquidityEngine,
		collateral:            collateral,
		broker:                broker,
		orderBook:             orderBook,
		equityShares:          equityShares,
		marketActivityTracker: marketActivityTracker,
		fee:                   fee,
		marketType:            marketType,
		marketID:              marketID,
		asset:                 asset,
		priceFactor:           priceFactor,
		priceRange:            priceRange,
		amm:                   amm,
	}

	return ml
}

func (m *MarketLiquidity) SetAMM(a AMM) {
	m.amm = a
}

func (m *MarketLiquidity) bondUpdate(ctx context.Context, transfer *types.Transfer) (*types.LedgerMovement, error) {
	switch m.marketType {
	case SpotMarketType:
		return m.collateral.BondSpotUpdate(ctx, m.marketID, transfer)
	default:
		return m.collateral.BondUpdate(ctx, m.marketID, transfer)
	}
}

func (m *MarketLiquidity) transferFees(ctx context.Context, ft events.FeesTransfer) ([]*types.LedgerMovement, error) {
	switch m.marketType {
	case SpotMarketType:
		return m.collateral.TransferSpotFees(ctx, m.marketID, m.asset, ft)
	default:
		return m.collateral.TransferFees(ctx, m.marketID, m.asset, ft)
	}
}

func (m *MarketLiquidity) applyPendingProvisions(
	ctx context.Context,
	now time.Time,
	targetStake *num.Uint,
) liquidity.Provisions {
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

		amendment, foundIdx := pendingProvisions.Get(partyID)
		if foundIdx < 0 {
			continue
		}

		// amendedCommitment - originalCommitment
		proposedCommitmentVariation := amendment.CommitmentAmount.ToDecimal().Sub(provision.CommitmentAmount.ToDecimal())

		// if commitment is increased or not changed, there is not penalty applied
		if !proposedCommitmentVariation.IsNegative() {
			continue
		}

		// min(-proposedCommitmentVariation, bondAccountBalance)
		commitmentVariation := num.MinD(proposedCommitmentVariation.Neg(), acc.Balance.ToDecimal())
		if commitmentVariation.IsZero() {
			continue
		}

		commitmentVariationPerParty[partyID] = commitmentVariation
		sumOfCommitmentVariations = sumOfCommitmentVariations.Add(commitmentVariation)
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
			bondLedgerMovement, err := m.bondUpdate(ctx, transfer)
			if err != nil {
				m.log.Panic("failed to apply SLA penalties to bond account", logging.Error(err))
			}

			ledgerMovements = append(ledgerMovements, bondLedgerMovement)
			continue
		}

		partyMaxPenaltyFreeReductionAmountU, _ := num.UintFromDecimal(partyMaxPenaltyFreeReductionAmount)

		if !partyMaxPenaltyFreeReductionAmountU.IsZero() {
			transfer := m.NewTransfer(partyID, types.TransferTypeBondHigh, partyMaxPenaltyFreeReductionAmountU)
			bondLedgerMovement, err := m.bondUpdate(ctx, transfer)
			if err != nil {
				m.log.Panic("failed to apply SLA penalties to bond account", logging.Error(err))
			}

			ledgerMovements = append(ledgerMovements, bondLedgerMovement)
		}

		penaltyIncurringReductionAmount := commitmentVariation.Sub(partyMaxPenaltyFreeReductionAmount)

		// transfer to general account
		freeAmount := one.Sub(m.earlyExitPenalty).Mul(penaltyIncurringReductionAmount)
		freeAmountU, _ := num.UintFromDecimal(freeAmount)

		if !freeAmountU.IsZero() {
			transfer := m.NewTransfer(partyID, types.TransferTypeBondHigh, freeAmountU)
			bondLedgerMovement, err := m.bondUpdate(ctx, transfer)
			if err != nil {
				m.log.Panic("failed to apply SLA penalties to bond account", logging.Error(err))
			}

			ledgerMovements = append(ledgerMovements, bondLedgerMovement)
		}

		slashingAmount := m.earlyExitPenalty.Mul(penaltyIncurringReductionAmount)
		slashingAmountU, _ := num.UintFromDecimal(slashingAmount)

		if !slashingAmountU.IsZero() {
			transfer := m.NewTransfer(partyID, types.TransferTypeBondSlashing, slashingAmountU)
			bondLedgerMovement, err := m.bondUpdate(ctx, transfer)
			if err != nil {
				m.log.Panic("failed to apply SLA penalties to bond account", logging.Error(err))
			}

			ledgerMovements = append(ledgerMovements, bondLedgerMovement)
		}
	}

	if len(ledgerMovements) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, ledgerMovements))
	}

	return m.liquidityEngine.ApplyPendingProvisions(ctx, now)
}

func (m *MarketLiquidity) syncPartyCommitmentWithBondAccount(
	ctx context.Context,
	appliedLiquidityProvisions liquidity.Provisions,
) {
	if len(appliedLiquidityProvisions) == 0 {
		appliedLiquidityProvisions = liquidity.Provisions{}
	}

	for partyID, provision := range m.liquidityEngine.ProvisionsPerParty() {
		acc, err := m.collateral.GetPartyBondAccount(m.marketID, partyID, m.asset)
		if err != nil {
			// the bond account should be definitely there at this point
			m.log.Panic("can not get LP party bond account",
				logging.Error(err),
				logging.PartyID(partyID),
			)
		}

		// lp provision and bond account are in sync, no need to change
		if provision.CommitmentAmount.EQ(acc.Balance) {
			continue
		}

		if acc.Balance.IsZero() {
			if err := m.liquidityEngine.CancelLiquidityProvision(ctx, partyID); err != nil {
				// the commitment should exists
				m.log.Panic("can not cancel liquidity provision commitment",
					logging.Error(err),
					logging.PartyID(partyID),
				)
			}

			provision.CommitmentAmount = acc.Balance.Clone()
			appliedLiquidityProvisions.Set(provision)
			continue
		}

		updatedProvision, err := m.liquidityEngine.UpdatePartyCommitment(partyID, acc.Balance)
		if err != nil {
			m.log.Panic("failed to update party commitment", logging.Error(err))
		}
		appliedLiquidityProvisions.Set(updatedProvision)
	}

	for _, provision := range appliedLiquidityProvisions {
		// now we can setup our party stake to calculate equities
		m.equityShares.SetPartyStake(provision.Party, provision.CommitmentAmount.Clone())
		// force update of shares so they are updated for all
		_ = m.equityShares.AllShares()
	}
}

func (m *MarketLiquidity) OnEpochStart(
	ctx context.Context, now time.Time,
	markPrice, midPrice, targetStake *num.Uint,
	positionFactor num.Decimal,
) {
	m.liquidityEngine.ResetSLAEpoch(now, markPrice, midPrice, positionFactor)

	appliedProvisions := m.applyPendingProvisions(ctx, now, targetStake)
	m.syncPartyCommitmentWithBondAccount(ctx, appliedProvisions)
}

func (m *MarketLiquidity) OnEpochEnd(ctx context.Context, t time.Time, epoch types.Epoch) {
	m.calculateAndDistribute(ctx, t)

	// report liquidity fees allocation stats
	feeStats := m.liquidityEngine.PaidLiquidityFeesStats()
	if !feeStats.TotalFeesPaid.IsZero() {
		m.broker.Send(events.NewPaidLiquidityFeesStatsEvent(ctx, feeStats.ToProto(m.marketID, m.asset, epoch.Seq)))
	}
}

func (m *MarketLiquidity) OnMarketClosed(ctx context.Context, t time.Time) {
	m.calculateAndDistribute(ctx, t)
}

func (m *MarketLiquidity) calculateAndDistribute(ctx context.Context, t time.Time) {
	penalties := m.liquidityEngine.CalculateSLAPenalties(t)

	if m.amm != nil {
		for _, subAccountID := range maps.Keys(m.amm.GetAMMPoolsBySubAccount()) {
			// set penalty to zero for pool sub accounts as they always meet their obligations for SLA
			penalties.PenaltiesPerParty[subAccountID] = &liquidity.SlaPenalty{
				Fee:  num.DecimalZero(),
				Bond: num.DecimalZero(),
			}
		}
	}

	m.distributeFeesBonusesAndApplyPenalties(ctx, penalties)
}

func (m *MarketLiquidity) OnTick(ctx context.Context, t time.Time) {
	// distribute liquidity fees each feeDistributionTimeStep
	if m.liquidityEngine.ReadyForFeesAllocation(t) {
		if err := m.AllocateFees(ctx); err != nil {
			m.log.Panic("liquidity fee distribution error", logging.Error(err))
		}

		// reset next distribution period
		m.liquidityEngine.ResetFeeAllocationPeriod(t)
		return
	}

	m.updateLiquidityScores()
}

func (m *MarketLiquidity) EndBlock(markPrice, midPrice *num.Uint, positionFactor num.Decimal) {
	m.liquidityEngine.EndBlock(markPrice, midPrice, positionFactor)
}

func (m *MarketLiquidity) updateLiquidityScores() {
	minLpPrice, maxLpPrice, err := m.ValidOrdersPriceRange()
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
	case types.MarketStateActive, types.MarketStatePending, types.MarketStateSuspended, types.MarketStateProposed, types.MarketStateSuspendedViaGovernance:
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

	tresp, err := m.bondUpdate(ctx, transfer)
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

	pendingAmendment := m.liquidityEngine.PendingProvisionByPartyID(party)
	currentProvision := m.liquidityEngine.LiquidityProvisionByPartyID(party)

	provisionToCopy := currentProvision
	if currentProvision == nil {
		if pendingAmendment == nil {
			m.log.Panic(
				"cannot edit liquidity provision from a non liquidity provider party",
				logging.PartyID(party),
			)
		}

		provisionToCopy = pendingAmendment
	}

	// If commitment amount is not provided we keep the same
	if lpa.CommitmentAmount == nil || lpa.CommitmentAmount.IsZero() {
		lpa.CommitmentAmount = provisionToCopy.CommitmentAmount
	}

	// If commitment amount is not provided we keep the same
	if lpa.Fee.IsZero() {
		lpa.Fee = provisionToCopy.Fee
	}

	// If commitment amount is not provided we keep the same
	if lpa.Reference == "" {
		lpa.Reference = provisionToCopy.Reference
	}

	var proposedCommitmentVariation num.Decimal

	// if pending commitment is being decreased then release the bond collateral
	if pendingAmendment != nil && !lpa.CommitmentAmount.IsZero() && lpa.CommitmentAmount.LT(pendingAmendment.CommitmentAmount) {
		amountToRelease := num.UintZero().Sub(pendingAmendment.CommitmentAmount, lpa.CommitmentAmount)
		if err := m.releasePendingBondCollateral(ctx, amountToRelease, party); err != nil {
			m.log.Debug("could not submit update bond for lp amendment",
				logging.PartyID(party),
				logging.MarketID(m.marketID),
				logging.Error(err))

			return err
		}

		proposedCommitmentVariation = pendingAmendment.CommitmentAmount.ToDecimal().Sub(lpa.CommitmentAmount.ToDecimal())
	} else {
		if currentProvision != nil {
			proposedCommitmentVariation = currentProvision.CommitmentAmount.ToDecimal().Sub(lpa.CommitmentAmount.ToDecimal())
		} else {
			proposedCommitmentVariation = pendingAmendment.CommitmentAmount.ToDecimal().Sub(lpa.CommitmentAmount.ToDecimal())
		}
	}

	// if increase commitment transfer funds to bond account
	if proposedCommitmentVariation.IsNegative() {
		_, err := m.ensureAndTransferCollateral(ctx, lpa.CommitmentAmount, party)
		if err != nil {
			m.log.Debug("could not submit update bond for lp amendment",
				logging.PartyID(party),
				logging.MarketID(m.marketID),
				logging.Error(err))

			return err
		}
	}

	applied, err := m.liquidityEngine.AmendLiquidityProvision(ctx, lpa, party, false)
	if err != nil {
		m.log.Panic("error while amending liquidity provision, this should not happen at this point, the LP was validated earlier",
			logging.Error(err))
	}

	if currentProvision != nil && applied && proposedCommitmentVariation.IsPositive() && !lpa.CommitmentAmount.IsZero() {
		amountToRelease := num.UintZero().Sub(currentProvision.CommitmentAmount, lpa.CommitmentAmount)
		if err := m.releasePendingBondCollateral(ctx, amountToRelease, party); err != nil {
			m.log.Debug("could not submit update bond for lp amendment",
				logging.PartyID(party),
				logging.MarketID(m.marketID),
				logging.Error(err))

			// rollback the amendment - TODO karel
			lpa.CommitmentAmount = currentProvision.CommitmentAmount
			m.liquidityEngine.AmendLiquidityProvision(ctx, lpa, party, false)

			return err
		}
	}

	return nil
}

// CancelLiquidityProvision amends liquidity provision to 0.
func (m *MarketLiquidity) CancelLiquidityProvision(ctx context.Context, party string) error {
	currentProvision := m.liquidityEngine.LiquidityProvisionByPartyID(party)
	pendingAmendment := m.liquidityEngine.PendingProvisionByPartyID(party)

	if currentProvision == nil && pendingAmendment == nil {
		return ErrPartyHasNoExistingLiquidityProvision
	}

	if pendingAmendment != nil && !pendingAmendment.CommitmentAmount.IsZero() {
		if err := m.releasePendingBondCollateral(ctx, pendingAmendment.CommitmentAmount.Clone(), party); err != nil {
			m.log.Debug("could release bond collateral for pending amendment",
				logging.PartyID(party),
				logging.MarketID(m.marketID),
				logging.Error(err))

			return err
		}
	}

	amendment := &types.LiquidityProvisionAmendment{
		MarketID:         m.marketID,
		CommitmentAmount: num.UintZero(),
		Fee:              num.DecimalZero(),
	}

	applied, err := m.liquidityEngine.AmendLiquidityProvision(ctx, amendment, party, true)
	if err != nil {
		m.log.Panic("error while amending liquidity provision, this should not happen at this point, the LP was validated earlier",
			logging.Error(err))
	}

	if applied && currentProvision != nil && !currentProvision.CommitmentAmount.IsZero() {
		if err := m.releasePendingBondCollateral(ctx, currentProvision.CommitmentAmount.Clone(), party); err != nil {
			m.log.Debug("could not submit update bond for lp amendment",
				logging.PartyID(party),
				logging.MarketID(m.marketID),
				logging.Error(err))

			// rollback amendment
			amendment.CommitmentAmount = currentProvision.CommitmentAmount
			amendment.Fee = currentProvision.Fee
			m.liquidityEngine.AmendLiquidityProvision(ctx, amendment, party, false)

			return err
		}
	}

	return nil
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
		return nil, ErrNotEnoughStake
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
	tresp, err := m.bondUpdate(ctx, transfer)
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
) error {
	transfer := &types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Amount: releaseAmount,
			Asset:  m.asset,
		},
		Type:      types.TransferTypeBondHigh,
		MinAmount: releaseAmount.Clone(),
	}

	ledgerMovement, err := m.bondUpdate(ctx, transfer)
	if err != nil {
		return err
	}
	m.broker.Send(events.NewLedgerMovements(
		ctx, []*types.LedgerMovement{ledgerMovement}))

	return nil
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

// ValidOrdersPriceRange returns min and max valid prices range for LP orders.
func (m *MarketLiquidity) ValidOrdersPriceRange() (*num.Uint, *num.Uint, error) {
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

func (m *MarketLiquidity) UpdateMarketConfig(risk liquidity.RiskModel, monitor liquidity.PriceMonitor) {
	m.liquidityEngine.UpdateMarketConfig(risk, monitor)
}

func (m *MarketLiquidity) UpdateSLAParameters(slaParams *types.LiquiditySLAParams) {
	m.priceRange = slaParams.PriceRange
	m.liquidityEngine.UpdateSLAParameters(slaParams)
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

func (m *MarketLiquidity) OnBondPenaltyFactorUpdate(bondPenaltyFactor num.Decimal) {
	m.bondPenaltyFactor = bondPenaltyFactor
}

func (m *MarketLiquidity) OnNonPerformanceBondPenaltySlopeUpdate(nonPerformanceBondPenaltySlope num.Decimal) {
	m.liquidityEngine.OnNonPerformanceBondPenaltySlopeUpdate(nonPerformanceBondPenaltySlope)
}

func (m *MarketLiquidity) OnNonPerformanceBondPenaltyMaxUpdate(nonPerformanceBondPenaltyMax num.Decimal) {
	m.liquidityEngine.OnNonPerformanceBondPenaltyMaxUpdate(nonPerformanceBondPenaltyMax)
}

func (m *MarketLiquidity) OnMaximumLiquidityFeeFactorLevelUpdate(liquidityFeeFactorLevelUpdate num.Decimal) {
	m.liquidityEngine.OnMaximumLiquidityFeeFactorLevelUpdate(liquidityFeeFactorLevelUpdate)
}

func (m *MarketLiquidity) OnEarlyExitPenalty(earlyExitPenalty num.Decimal) {
	m.earlyExitPenalty = earlyExitPenalty
}

func (m *MarketLiquidity) OnStakeToCcyVolumeUpdate(stakeToCcyVolume num.Decimal) {
	m.liquidityEngine.OnStakeToCcyVolumeUpdate(stakeToCcyVolume)
}

func (m *MarketLiquidity) OnProvidersFeeCalculationTimeStep(d time.Duration) {
	m.liquidityEngine.OnProvidersFeeCalculationTimeStep(d)
}

func (m *MarketLiquidity) IsProbabilityOfTradingInitialised() bool {
	return m.liquidityEngine.IsProbabilityOfTradingInitialised()
}

func (m *MarketLiquidity) GetAverageLiquidityScores() map[string]num.Decimal {
	return m.liquidityEngine.GetAverageLiquidityScores()
}

func (m *MarketLiquidity) ProvisionsPerParty() liquidity.ProvisionsPerParty {
	return m.liquidityEngine.ProvisionsPerParty()
}

func (m *MarketLiquidity) CalculateSuppliedStake() *num.Uint {
	return m.liquidityEngine.CalculateSuppliedStake()
}
