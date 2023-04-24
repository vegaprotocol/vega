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

package governance_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/governance"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProposalForNewMarket(t *testing.T) {
	t.Run("Submitting a proposal for new market succeeds", testSubmittingProposalForNewMarketSucceeds)
	t.Run("Submitting a proposal with internal time termination for new market succeeds", testSubmittingProposalWithInternalTimeTerminationForNewMarketSucceeds)
	t.Run("Submitting a proposal with internal time termination with `less than equal` condition fails", testSubmittingProposalWithInternalTimeTerminationWithLessThanEqualConditionForNewMarketFails)
	t.Run("Submitting a proposal with internal time settling for new market fails", testSubmittingProposalWithInternalTimeSettlingForNewMarketFails)
	t.Run("Submitting a proposal with external source using internal time termination key for new market succeeds", testSubmittingProposalWithExternalWithInternalTimeTerminationKeyForNewMarketSucceeds)
	t.Run("Submitting a duplicated proposal for new market fails", testSubmittingDuplicatedProposalForNewMarketFails)
	t.Run("Submitting a duplicated proposal with internal time termination for new market fails", testSubmittingDuplicatedProposalWithInternalTimeTerminationForNewMarketFails)
	t.Run("Submitting a proposal for new market with bad risk parameter fails", testSubmittingProposalForNewMarketWithBadRiskParameterFails)
	t.Run("Submitting a proposal for new market with internal time termination with bad risk parameter fails", testSubmittingProposalForNewMarketWithInternalTimeTerminationWithBadRiskParameterFails)

	t.Run("Rejecting a proposal for new market succeeds", testRejectingProposalForNewMarketSucceeds)

	t.Run("Voting for a new market proposal succeeds", testVotingForNewMarketProposalSucceeds)
	t.Run("Voting with a majority of 'yes' makes the new market proposal passed", testVotingWithMajorityOfYesMakesNewMarketProposalPassed)
	t.Run("Voting with a majority of 'no' makes the new market proposal declined", testVotingWithMajorityOfNoMakesNewMarketProposalDeclined)
	t.Run("Voting with insufficient participation makes the new market proposal declined", testVotingWithInsufficientParticipationMakesNewMarketProposalDeclined)
}

func testSubmittingProposalForNewMarketSucceeds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := eng.newValidParty("a-valid-party", 123456789)
	proposal := eng.newProposalForNewMarket(party.Id, eng.tsvc.GetTimeNow(), nil, nil, true)

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.expectOpenProposalEvent(t, party.Id, proposal.ID)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)
	require.NotNil(t, toSubmit)
	assert.True(t, toSubmit.IsNewMarket())
	require.NotNil(t, toSubmit.NewMarket().Market())
}

func testSubmittingProposalWithInternalTimeTerminationForNewMarketSucceeds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := eng.newValidParty("a-valid-party", 123456789)
	proposal := eng.newProposalForNewMarket(party.Id, eng.tsvc.GetTimeNow(), nil, nil, false)

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.expectOpenProposalEvent(t, party.Id, proposal.ID)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)
	require.NotNil(t, toSubmit)
	assert.True(t, toSubmit.IsNewMarket())
	require.NotNil(t, toSubmit.NewMarket().Market())
}

func testSubmittingProposalWithInternalTimeSettlingForNewMarketFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := eng.newValidParty("a-valid-party", 123456789)
	now := eng.tsvc.GetTimeNow()
	id := eng.newProposalID()
	tm := time.Now().Add(time.Hour * 24 * 365)
	_, termBinding := produceTimeTriggeredDataSourceSpec(tm)

	proposal := types.Proposal{
		ID:        id,
		Reference: "ref-" + id,
		Party:     party.Id,
		State:     types.ProposalStateOpen,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    now.Add(48 * time.Hour).Unix(),
			EnactmentTimestamp:  now.Add(2 * 48 * time.Hour).Unix(),
			ValidationTimestamp: now.Add(1 * time.Hour).Unix(),
			Change: &types.ProposalTermsNewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Name: "June 2020 GBP vs VUSD future",
							Code: "CRYPTO:GBPVUSD/JUN20",
							Product: &types.InstrumentConfigurationFuture{
								Future: &types.FutureProduct{
									SettlementAsset: "VUSD",
									QuoteName:       "VUSD",
									DataSourceSpecForSettlementData: *types.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeInt,
									).SetTimeTriggerConditionConfig(
										[]*types.DataSourceSpecCondition{
											{
												Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
												Value:    "0",
											},
										},
									),
									DataSourceSpecForTradingTermination: *types.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeInt,
									).SetTimeTriggerConditionConfig(
										[]*types.DataSourceSpecCondition{
											{
												Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
												Value:    fmt.Sprintf("%d", tm.UnixNano()),
											},
										}),
									DataSourceSpecBinding: termBinding,
								},
							},
						},
						RiskParameters: &types.NewMarketConfigurationLogNormal{
							LogNormal: &types.LogNormalRiskModel{
								RiskAversionParameter: num.DecimalFromFloat(0.01),
								Tau:                   num.DecimalFromFloat(0.00011407711613050422),
								Params: &types.LogNormalModelParams{
									Mu:    num.DecimalZero(),
									R:     num.DecimalFromFloat(0.016),
									Sigma: num.DecimalFromFloat(0.09),
								},
							},
						},
						Metadata:                []string{"asset_class:fx/crypto", "product:futures"},
						DecimalPlaces:           0,
						LpPriceRange:            num.DecimalFromFloat(0.95),
						LinearSlippageFactor:    num.DecimalFromFloat(0.1),
						QuadraticSlippageFactor: num.DecimalFromFloat(0.1),
					},
				},
			},
		},
		Rationale: &types.ProposalRationale{
			Description: "some description",
		},
	}

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.expectRejectedProposalEvent(t, party.Id, proposal.ID, types.ProposalErrorInvalidFutureProduct)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	assert.Error(t, err, governance.ErrSettlementWithInternalDataSourceIsNotAllowed)
	require.Nil(t, toSubmit)
}

func testSubmittingProposalWithInternalTimeTerminationWithLessThanEqualConditionForNewMarketFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := eng.newValidParty("a-valid-party", 123456789)
	now := eng.tsvc.GetTimeNow()
	id := eng.newProposalID()
	tm := time.Now().Add(time.Hour * 24 * 365)
	_, termBinding := produceTimeTriggeredDataSourceSpec(tm)

	settl := types.NewDataSourceDefinition(
		vegapb.DataSourceDefinitionTypeExt,
	).SetOracleConfig(
		&types.DataSourceSpecConfiguration{
			Signers: []*types.Signer{types.CreateSignerFromString("0xDEADBEEF", types.DataSignerTypePubKey)},
			Filters: []*types.DataSourceSpecFilter{
				{
					Key: &types.DataSourceSpecPropertyKey{
						Name: "prices.ETH.value",
						Type: datapb.PropertyKey_TYPE_INTEGER,
					},
					Conditions: []*types.DataSourceSpecCondition{
						{
							Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
							Value:    "0",
						},
					},
				},
			},
		},
	)

	term := types.NewDataSourceDefinition(
		vegapb.DataSourceDefinitionTypeInt,
	).SetTimeTriggerConditionConfig(
		[]*types.DataSourceSpecCondition{
			{
				Operator: datapb.Condition_OPERATOR_LESS_THAN,
				Value:    fmt.Sprintf("%d", tm.UnixNano()),
			},
		})

	riskParameters := types.NewMarketConfigurationLogNormal{
		LogNormal: &types.LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(0.01),
			Tau:                   num.DecimalFromFloat(0.00011407711613050422),
			Params: &types.LogNormalModelParams{
				Mu:    num.DecimalZero(),
				R:     num.DecimalFromFloat(0.016),
				Sigma: num.DecimalFromFloat(0.09),
			},
		},
	}

	proposal := types.Proposal{
		ID:        id,
		Reference: "ref-" + id,
		Party:     party.Id,
		State:     types.ProposalStateOpen,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    now.Add(48 * time.Hour).Unix(),
			EnactmentTimestamp:  now.Add(2 * 48 * time.Hour).Unix(),
			ValidationTimestamp: now.Add(1 * time.Hour).Unix(),
			Change: &types.ProposalTermsNewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Name: "June 2020 GBP vs VUSD future",
							Code: "CRYPTO:GBPVUSD/JUN20",
							Product: &types.InstrumentConfigurationFuture{
								Future: &types.FutureProduct{
									SettlementAsset:                     "VUSD",
									QuoteName:                           "VUSD",
									DataSourceSpecForSettlementData:     *settl,
									DataSourceSpecForTradingTermination: *term,
									DataSourceSpecBinding:               termBinding,
								},
							},
						},
						RiskParameters:          &riskParameters,
						Metadata:                []string{"asset_class:fx/crypto", "product:futures"},
						DecimalPlaces:           0,
						LpPriceRange:            num.DecimalFromFloat(0.95),
						LinearSlippageFactor:    num.DecimalFromFloat(0.1),
						QuadraticSlippageFactor: num.DecimalFromFloat(0.1),
					},
				},
			},
		},
		Rationale: &types.ProposalRationale{
			Description: "some description",
		},
	}

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.expectRejectedProposalEvent(t, party.Id, proposal.ID, types.ProposalErrorInvalidFutureProduct)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	assert.Error(t, err, types.ErrDataSourceSpecHasInvalidTimeCondition)
	require.Nil(t, toSubmit)

	term = types.NewDataSourceDefinition(
		vegapb.DataSourceDefinitionTypeInt,
	).SetTimeTriggerConditionConfig(
		[]*types.DataSourceSpecCondition{
			{
				Operator: datapb.Condition_OPERATOR_LESS_THAN_OR_EQUAL,
				Value:    fmt.Sprintf("%d", tm.UnixNano()),
			},
		})

	proposal = types.Proposal{
		ID:        id,
		Reference: "ref-" + id,
		Party:     party.Id,
		State:     types.ProposalStateOpen,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    now.Add(48 * time.Hour).Unix(),
			EnactmentTimestamp:  now.Add(2 * 48 * time.Hour).Unix(),
			ValidationTimestamp: now.Add(1 * time.Hour).Unix(),
			Change: &types.ProposalTermsNewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Name: "June 2020 GBP vs VUSD future",
							Code: "CRYPTO:GBPVUSD/JUN20",
							Product: &types.InstrumentConfigurationFuture{
								Future: &types.FutureProduct{
									SettlementAsset:                     "VUSD",
									QuoteName:                           "VUSD",
									DataSourceSpecForSettlementData:     *settl,
									DataSourceSpecForTradingTermination: *term,
									DataSourceSpecBinding:               termBinding,
								},
							},
						},
						RiskParameters:          &riskParameters,
						Metadata:                []string{"asset_class:fx/crypto", "product:futures"},
						DecimalPlaces:           0,
						LpPriceRange:            num.DecimalFromFloat(0.95),
						LinearSlippageFactor:    num.DecimalFromFloat(0.1),
						QuadraticSlippageFactor: num.DecimalFromFloat(0.1),
					},
				},
			},
		},
		Rationale: &types.ProposalRationale{
			Description: "some description",
		},
	}

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.expectRejectedProposalEvent(t, party.Id, proposal.ID, types.ProposalErrorInvalidFutureProduct)

	// when
	toSubmit, err = eng.submitProposal(t, proposal)

	// then
	assert.Error(t, err, types.ErrDataSourceSpecHasInvalidTimeCondition)
	require.Nil(t, toSubmit)
}

func testSubmittingProposalWithExternalWithInternalTimeTerminationKeyForNewMarketSucceeds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := eng.newValidParty("a-valid-party", 123456789)
	filter, binding := produceTimeTriggeredDataSourceSpec(time.Now())
	proposal := eng.newProposalForNewMarket(party.Id, eng.tsvc.GetTimeNow(), filter, binding, true)

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.expectOpenProposalEvent(t, party.Id, proposal.ID)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)
	require.NotNil(t, toSubmit)
	assert.True(t, toSubmit.IsNewMarket())
	require.NotNil(t, toSubmit.NewMarket().Market())
}

func testSubmittingDuplicatedProposalForNewMarketFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewMarket(party, eng.tsvc.GetTimeNow(), nil, nil, true)

	// setup
	eng.ensureTokenBalanceForParty(t, party, 1000)
	eng.ensureAllAssetEnabled(t)

	// expect
	eng.expectOpenProposalEvent(t, party, proposal.ID)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)

	// given
	duplicatedProposal := proposal
	duplicatedProposal.Reference = "this-is-a-copy"

	// when
	_, err = eng.submitProposal(t, duplicatedProposal)

	// then
	require.Error(t, err)
	assert.EqualError(t, governance.ErrProposalIsDuplicate, err.Error())

	// given
	duplicatedProposal = proposal
	duplicatedProposal.State = types.ProposalStatePassed

	// when
	_, err = eng.submitProposal(t, duplicatedProposal)

	// then
	require.Error(t, err)
	assert.EqualError(t, governance.ErrProposalIsDuplicate, err.Error(), "reject attempt to change state indirectly")
}

func testSubmittingDuplicatedProposalWithInternalTimeTerminationForNewMarketFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewMarket(party, eng.tsvc.GetTimeNow(), nil, nil, false)

	// setup
	eng.ensureTokenBalanceForParty(t, party, 1000)
	eng.ensureAllAssetEnabled(t)

	// expect
	eng.expectOpenProposalEvent(t, party, proposal.ID)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)

	// given
	duplicatedProposal := proposal
	duplicatedProposal.Reference = "this-is-a-copy"

	// when
	_, err = eng.submitProposal(t, duplicatedProposal)

	// then
	require.Error(t, err)
	assert.EqualError(t, governance.ErrProposalIsDuplicate, err.Error())

	// given
	duplicatedProposal = proposal
	duplicatedProposal.State = types.ProposalStatePassed

	// when
	_, err = eng.submitProposal(t, duplicatedProposal)

	// then
	require.Error(t, err)
	assert.EqualError(t, governance.ErrProposalIsDuplicate, err.Error(), "reject attempt to change state indirectly")
}

func testSubmittingProposalForNewMarketWithBadRiskParameterFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := eng.newValidParty("a-valid-party", 1)
	eng.ensureAllAssetEnabled(t)

	proposal := eng.newProposalForNewMarket(party.Id, eng.tsvc.GetTimeNow(), nil, nil, true)
	proposal.Terms.GetNewMarket().Changes.RiskParameters = &types.NewMarketConfigurationLogNormal{
		LogNormal: &types.LogNormalRiskModel{
			Params: nil, // it's nil by zero value, but eh, let's show that's what we test
		},
	}

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid risk parameter")
}

func testSubmittingProposalForNewMarketWithInternalTimeTerminationWithBadRiskParameterFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := eng.newValidParty("a-valid-party", 1)
	eng.ensureAllAssetEnabled(t)

	proposal := eng.newProposalForNewMarket(party.Id, eng.tsvc.GetTimeNow(), nil, nil, false)
	proposal.Terms.GetNewMarket().Changes.RiskParameters = &types.NewMarketConfigurationLogNormal{
		LogNormal: &types.LogNormalRiskModel{
			Params: nil, // it's nil by zero value, but eh, let's show that's what we test
		},
	}

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid risk parameter")
}

func testOutOfRangeRiskParamFail(t *testing.T, lnm *types.LogNormalRiskModel) {
	t.Helper()
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := eng.newValidParty("a-valid-party", 1)
	eng.ensureAllAssetEnabled(t)

	proposal := eng.newProposalForNewMarket(party.Id, eng.tsvc.GetTimeNow(), nil, nil, true)
	proposal.Terms.GetNewMarket().Changes.RiskParameters = &types.NewMarketConfigurationLogNormal{LogNormal: lnm}

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid risk parameter")
}

func TestSubmittingProposalForNewMarketWithOutOfRangeRiskParameterFails(t *testing.T) {
	lnm := &types.LogNormalRiskModel{}
	lnm.RiskAversionParameter = num.DecimalFromFloat(1e-8 - 1e-12)
	testOutOfRangeRiskParamFail(t, lnm)
	lnm.RiskAversionParameter = num.DecimalFromFloat(1e1 + 1e-12)
	testOutOfRangeRiskParamFail(t, lnm)
	lnm.RiskAversionParameter = num.DecimalFromFloat(1e-6)
	lnm.Tau = num.DecimalFromFloat(1e-8 - 1e-12)
	testOutOfRangeRiskParamFail(t, lnm)
	lnm.Tau = num.DecimalFromFloat(1 + 1e-12)
	testOutOfRangeRiskParamFail(t, lnm)
	lnm.Tau = num.DecimalOne()
	lnm.Params = &types.LogNormalModelParams{}
	lnm.Params.Mu = num.DecimalFromFloat(-1e-6 - 1e-12)
	testOutOfRangeRiskParamFail(t, lnm)
	lnm.Params.Mu = num.DecimalFromFloat(1e-6 + 1e-12)
	testOutOfRangeRiskParamFail(t, lnm)
	lnm.Params.Mu = num.DecimalFromFloat(0.0)
	lnm.Params.R = num.DecimalFromFloat(-1 - 1e-12)
	testOutOfRangeRiskParamFail(t, lnm)
	lnm.Params.R = num.DecimalFromFloat(1 + 1e-12)
	testOutOfRangeRiskParamFail(t, lnm)
	lnm.Params.R = num.DecimalFromFloat(0.0)
	lnm.Params.Sigma = num.DecimalFromFloat(1e-3 - 1e-12)
	testOutOfRangeRiskParamFail(t, lnm)
	lnm.Params.Sigma = num.DecimalFromFloat(50 + 1e-12)
	testOutOfRangeRiskParamFail(t, lnm)
	lnm.Params.Sigma = num.DecimalFromFloat(1.0)

	// now all risk params are valid
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := eng.newValidParty("a-valid-party", 1)
	eng.ensureAllAssetEnabled(t)

	proposal := eng.newProposalForNewMarket(party.Id, eng.tsvc.GetTimeNow(), nil, nil, true)
	proposal.Terms.GetNewMarket().Changes.RiskParameters = &types.NewMarketConfigurationLogNormal{LogNormal: lnm}

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)
}

func testRejectingProposalForNewMarketSucceeds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewMarket(party, eng.tsvc.GetTimeNow(), nil, nil, true)

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.ensureTokenBalanceForParty(t, party, 10000)

	// expect
	eng.expectOpenProposalEvent(t, party, proposal.ID)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)
	require.NotNil(t, toSubmit)

	// expect
	eng.expectRejectedProposalEvent(t, party, proposal.ID, types.ProposalErrorCouldNotInstantiateMarket)

	// when
	err = eng.RejectProposal(context.Background(), toSubmit.Proposal(), types.ProposalErrorCouldNotInstantiateMarket, assert.AnError)

	// then
	require.NoError(t, err)

	// when
	// Just one more time to make sure it was removed from proposals.
	err = eng.RejectProposal(context.Background(), toSubmit.Proposal(), types.ProposalErrorCouldNotInstantiateMarket, assert.AnError)

	// then
	assert.EqualError(t, err, governance.ErrProposalDoesNotExist.Error())
}

func testVotingForNewMarketProposalSucceeds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewMarket(proposer, eng.tsvc.GetTimeNow(), nil, nil, true)

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.ensureTokenBalanceForParty(t, proposer, 1)

	// expect
	eng.expectOpenProposalEvent(t, proposer, proposal.ID)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)

	// given
	voter := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voter, 1)

	// expect
	eng.expectVoteEvent(t, voter, proposal.ID)

	// when
	err = eng.addYesVote(t, voter, proposal.ID)

	// then
	require.NoError(t, err)
}

func testVotingWithMajorityOfYesMakesNewMarketProposalPassed(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// when
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewMarket(proposer, eng.tsvc.GetTimeNow(), nil, nil, true)

	// setup
	eng.ensureStakingAssetTotalSupply(t, 9)
	eng.ensureAllAssetEnabled(t)
	eng.ensureTokenBalanceForParty(t, proposer, 1)

	// expect
	eng.expectOpenProposalEvent(t, proposer, proposal.ID)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)

	// given
	voter1 := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voter1, 7)

	// expect
	eng.expectVoteEvent(t, voter1, proposal.ID)

	// then
	err = eng.addYesVote(t, voter1, proposal.ID)

	// then
	require.NoError(t, err)

	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.ensureTokenBalanceForParty(t, voter1, 7)

	// expect
	eng.expectPassedProposalEvent(t, proposal.ID)
	eng.expectTotalGovernanceTokenFromVoteEvents(t, "1", "7")
	eng.expectGetMarketState(t, proposal.ID)

	// when
	eng.OnTick(context.Background(), afterClosing)

	// given
	voter2 := vgrand.RandomStr(5)

	// when
	err = eng.addNoVote(t, voter2, proposal.ID)

	// then
	require.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalNotOpenForVotes.Error())

	// given
	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)

	// when
	// no calculations, no state change, simply removed from governance engine
	toBeEnacted, _ := eng.OnTick(context.Background(), afterEnactment)

	// then
	require.Len(t, toBeEnacted, 1)
	assert.Equal(t, proposal.ID, toBeEnacted[0].Proposal().ID)

	// when
	err = eng.addNoVote(t, voter2, proposal.ID)

	// then
	require.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalDoesNotExist.Error())
}

func testVotingWithMajorityOfNoMakesNewMarketProposalDeclined(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewMarket(proposer, eng.tsvc.GetTimeNow(), nil, nil, true)

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.ensureStakingAssetTotalSupply(t, 200)
	eng.ensureTokenBalanceForParty(t, proposer, 100)

	// expect
	eng.expectOpenProposalEvent(t, proposer, proposal.ID)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)

	// given
	voter := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voter, 100)

	// expect
	eng.expectVoteEvent(t, voter, proposal.ID)

	// when
	err = eng.addYesVote(t, voter, proposal.ID)

	// then
	require.NoError(t, err)

	// setup
	eng.ensureTokenBalanceForParty(t, voter, 100)

	// setup
	eng.expectVoteEvent(t, voter, proposal.ID)

	// when
	err = eng.addNoVote(t, voter, proposal.ID)

	// then
	require.NoError(t, err)

	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.ensureTokenBalanceForParty(t, voter, 100)

	// expect
	eng.expectDeclinedProposalEvent(t, proposal.ID, types.ProposalErrorMajorityThresholdNotReached)
	eng.expectTotalGovernanceTokenFromVoteEvents(t, "1", "100")
	eng.expectGetMarketState(t, proposal.ID)

	// when
	_, voteClosed := eng.OnTick(context.Background(), afterClosing)

	// then
	require.Len(t, voteClosed, 1)
	vc := voteClosed[0]
	require.NotNil(t, vc.NewMarket())
	assert.True(t, vc.NewMarket().Rejected())

	// given
	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)

	// when
	toBeEnacted, _ := eng.OnTick(context.Background(), afterEnactment)

	// then
	assert.Empty(t, toBeEnacted)
}

func testVotingWithInsufficientParticipationMakesNewMarketProposalDeclined(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewMarket(proposer, eng.tsvc.GetTimeNow(), nil, nil, true)

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.ensureStakingAssetTotalSupply(t, 800)
	eng.ensureTokenBalanceForParty(t, proposer, 100)

	// expect
	eng.expectOpenProposalEvent(t, proposer, proposal.ID)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)

	// given
	voter := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voter, 100)

	// expect
	eng.expectVoteEvent(t, voter, proposal.ID)

	// when
	err = eng.addYesVote(t, voter, proposal.ID)

	// then
	require.NoError(t, err)

	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.ensureTokenBalanceForParty(t, voter, 100)

	// expect
	eng.expectDeclinedProposalEvent(t, proposal.ID, types.ProposalErrorParticipationThresholdNotReached)
	eng.expectTotalGovernanceTokenFromVoteEvents(t, "1", "100")
	eng.expectGetMarketState(t, proposal.ID)
	// when
	_, voteClosed := eng.OnTick(context.Background(), afterClosing)

	// then
	require.Len(t, voteClosed, 1)
	vc := voteClosed[0]
	require.NotNil(t, vc.NewMarket())
	assert.True(t, vc.NewMarket().Rejected())

	// given
	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)

	// when
	toBeEnacted, _ := eng.OnTick(context.Background(), afterEnactment)

	// then
	assert.Empty(t, toBeEnacted)
}
