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

package spot

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/idgeneration"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

var ErrCommitmentAmountTooLow = errors.New("commitment amount is too low")

// SubmitLiquidityProvision forwards a LiquidityProvisionSubmission to the Liquidity Engine.
func (m *Market) SubmitLiquidityProvision(ctx context.Context, sub *types.LiquidityProvisionSubmission, party, deterministicID string) (err error) {
	m.idgen = idgeneration.New(deterministicID)
	defer func() { m.idgen = nil }()

	if !m.canSubmitCommitment() {
		return common.ErrCommitmentSubmissionNotAllowed
	}

	if len(sub.Buys) > 0 || len(sub.Sells) > 0 || len(sub.Reference) > 0 {
		return fmt.Errorf("invalid liquidity provision submission for a spot market")
	}

	if err := m.ensureLPCommitmentAmount(sub.CommitmentAmount); err != nil {
		return err
	}

	if err := m.liquidity.SubmitLiquidityProvision(ctx, sub, party, m.idgen); err != nil {
		return err
	}

	// add the party to the list of all parties involved with
	// this market
	m.addParty(party)

	// TODO here we need to hanlde liquidity submission and reject it in failure

	return nil
}

// AmendLiquidityProvision forwards a LiquidityProvisionAmendment to the Liquidity Engine.
func (m *Market) AmendLiquidityProvision(ctx context.Context, lpa *types.LiquidityProvisionAmendment, party string, deterministicID string) (err error) {
	m.idgen = idgeneration.New(deterministicID)
	defer func() { m.idgen = nil }()

	if !m.canSubmitCommitment() {
		return common.ErrCommitmentSubmissionNotAllowed
	}

	if len(lpa.Buys) > 0 || len(lpa.Sells) > 0 || len(lpa.Reference) > 0 {
		return fmt.Errorf("invalid liquidity amendment for spot market")
	}

	if err := m.liquidity.ValidateLiquidityProvisionAmendment(lpa); err != nil {
		return err
	}

	if lpa.CommitmentAmount != nil {
		if err := m.ensureLPCommitmentAmount(lpa.CommitmentAmount); err != nil {
			return err
		}
	}

	if !m.liquidity.IsLiquidityProvider(party) {
		return common.ErrPartyNotLiquidityProvider
	}

	lp := m.liquidity.LiquidityProvisionByPartyID(party)
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

	// TODO not sure this is still relevant
	if lpa.CommitmentAmount.LT(lp.CommitmentAmount) {
		// first - does the market have enough stake
		supplied := m.getSuppliedStake()
		if m.getTargetStake().GTE(supplied) {
			return common.ErrNotEnoughStake
		}

		// now if the stake surplus is > than the change we are OK
		surplus := supplied.Sub(supplied, m.getTargetStake())
		diff := num.UintZero().Sub(lp.CommitmentAmount, lpa.CommitmentAmount)
		if surplus.LT(diff) {
			return common.ErrNotEnoughStake
		}
	}

	return m.amendLiquidityProvision(ctx, lpa, party)
}

// CancelLiquidityProvision forwards a LiquidityProvisionCancel to the Liquidity Engine.
func (m *Market) CancelLiquidityProvision(ctx context.Context, cancel *types.LiquidityProvisionCancellation, party string) (err error) {
	if !m.canSubmitCommitment() {
		return common.ErrCommitmentSubmissionNotAllowed
	}

	if !m.liquidity.IsLiquidityProvider(party) {
		return common.ErrPartyNotLiquidityProvider
	}

	lp := m.liquidity.LiquidityProvisionByPartyID(party)
	if lp == nil {
		return fmt.Errorf("cannot edit liquidity provision from a non liquidity provider party (%v)", party)
	}

	supplied := m.getSuppliedStake()
	if m.getTargetStake().GTE(supplied) {
		return common.ErrNotEnoughStake
	}

	// now if the stake surplus is > than the change we are OK
	surplus := supplied.Sub(supplied, m.getTargetStake())
	if surplus.LT(lp.CommitmentAmount) {
		return common.ErrNotEnoughStake
	}
	return m.cancelLiquidityProvision(ctx, party)
}

// This will be needed eventually so commenting out for now.
// func (m *Market) cancelPendingLiquidityProvision(ctx context.Context, party string) error {
// 	// we will just cancel the party,
// 	// no bond slashing applied
// 	if err := m.cancelLiquidityProvision(ctx, party); err != nil {
// 		m.log.Debug("error cancelling liquidity provision commitment",
// 			logging.PartyID(party),
// 			logging.MarketID(m.GetID()),
// 			logging.Error(err))
// 		return err
// 	}

// 	return nil
// }

func (m *Market) cancelLiquidityProvision(ctx context.Context, party string) error {
	// cancel the liquidity provision
	m.liquidity.CancelLiquidityProvision(ctx, party)

	// TODO when do the orders actually get cancelled
	// is all of that still relevant?
	m.updateLiquidityFee(ctx)
	// and remove the party from the equity share like calculation
	m.equityShares.SetPartyStake(party, nil)
	// force update of shares so they are updated for all
	_ = m.equityShares.SharesExcept(m.liquidity.GetInactiveParties())

	m.checkForReferenceMoves(ctx, []*types.Order{}, true)
	return nil
}

func (m *Market) amendLiquidityProvision(ctx context.Context, sub *types.LiquidityProvisionAmendment, party string) (err error) {
	if m.as.InAuction() {
		return m.finalizeLiquidityProvisionAmendmentAuction(ctx, sub, party)
	}
	return m.finalizeLiquidityProvisionAmendmentContinuous(ctx, sub, party)
}

func (m *Market) finalizeLiquidityProvisionAmendmentAuction(
	ctx context.Context, sub *types.LiquidityProvisionAmendment, party string,
) error {
	// first parameter is the update to the orders, but we know that during
	// auction no orders shall be return, so let's just look at the error
	err := m.liquidity.AmendLiquidityProvision(ctx, sub, party, m.idgen)
	if err != nil {
		m.log.Panic("error while amending liquidity provision, this should not happen at this point, the LP was validated earlier",
			logging.Error(err))
	}

	defer func() {
		m.updateMarketValueProxy()
		// now we can update the liquidity fee to be taken
		m.updateLiquidityFee(ctx)
		// now we can setup our party stake to calculate equities
		m.equityShares.SetPartyStake(party, sub.CommitmentAmount.Clone())
		// force update of shares so they are updated for all
		_ = m.equityShares.SharesExcept(m.liquidity.GetInactiveParties())
	}()

	return nil
}

func (m *Market) finalizeLiquidityProvisionAmendmentContinuous(ctx context.Context, sub *types.LiquidityProvisionAmendment, party string) error {
	err := m.liquidity.AmendLiquidityProvision(ctx, sub, party, m.idgen)
	if err != nil {
		m.log.Panic("error while amending liquidity provision, this should not happen at this point, the LP was validated earlier",
			logging.Error(err))
	}

	defer func() {
		m.updateMarketValueProxy()
		// now we can update the liquidity fee to be taken
		m.updateLiquidityFee(ctx)
		// now we can setup our party stake to calculate equities
		m.equityShares.SetPartyStake(party, sub.CommitmentAmount)
		// force update of shares so they are updated for all
		_ = m.equityShares.SharesExcept(m.liquidity.GetInactiveParties())
	}()

	// this workd but we definitely trigger some recursive loop which
	// are unlikely to be fine.
	m.checkForReferenceMoves(ctx, []*types.Order{}, true)

	return nil
}

func (m *Market) ensureLPCommitmentAmount(amount *num.Uint) error {
	quantum, err := m.collateral.GetAssetQuantum(m.quoteAsset)
	if err != nil {
		m.log.Panic("could not get quantum for asset, this should never happen",
			logging.AssetID(m.quoteAsset),
			logging.Error(err),
		)
	}
	minStake := quantum.Mul(m.minLPStakeQuantumMultiple)
	if amount.ToDecimal().LessThan(minStake) {
		return ErrCommitmentAmountTooLow
	}

	return nil
}

func (m *Market) updateSharesWithLiquidityScores(shares map[string]num.Decimal) map[string]num.Decimal {
	lScores := m.liquidity.GetAverageLiquidityScores()

	total := num.DecimalZero()
	for k, v := range shares {
		l, ok := lScores[k]
		if !ok {
			continue
		}
		adjusted := v.Mul(l)
		shares[k] = adjusted

		total = total.Add(adjusted)
	}

	// normalise
	if !total.IsZero() {
		for k, v := range shares {
			shares[k] = v.Div(total)
		}
	}

	// reset for next period
	m.liquidity.ResetAverageLiquidityScores()

	return shares
}
