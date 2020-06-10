package governance_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/governance/mocks"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type tstEngine struct {
	*governance.Engine
	ctrl *gomock.Controller
	accs *mocks.MockAccounts
	buf  *mocks.MockBuffer
	vbuf *mocks.MockVoteBuf
}

func TestProposals(t *testing.T) {
	t.Run("Submit a valid proposal - success", testSubmitValidProposalSuccess)
	t.Run("Submit a valid proposal - duplicate", testSubmitValidProposalDuplicate)
	t.Run("Validate proposer stake", testProposerStake)
	t.Run("Validate closing time", testClosingTime)
	t.Run("Validate enactment time", testEnactmentTime)
}

func testSubmitValidProposalSuccess(t *testing.T) {
	eng := getTestEngine(t)

	defer eng.ctrl.Finish()
	partyID := "party1"
	now := time.Now()
	acc := types.Account{
		Id:      "acc-1",
		Owner:   partyID,
		Balance: 1000,
		Asset:   collateral.TokenAsset,
	}
	prop := types.Proposal{
		ID:        "prop-1",
		Reference: "test-prop-1",
		PartyID:   partyID,
		State:     types.Proposal_STATE_OPEN,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:   now.Add(48 * time.Hour).Unix(),
			EnactmentTimestamp: now.Add(48 * time.Hour).Unix(),
		},
	}
	eng.accs.EXPECT().GetTotalTokens().Times(1).Return(acc.Balance)
	eng.accs.EXPECT().GetPartyTokenAccount(partyID).Times(1).Return(&acc, nil)
	eng.buf.EXPECT().Add(gomock.Any()).Times(1)
	err := eng.SubmitProposal(prop)
	assert.NoError(t, err)
}

func testSubmitValidProposalDuplicate(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()
	partyID := "party1"
	now := time.Now()
	acc := types.Account{
		Id:      "acc-1",
		Owner:   partyID,
		Balance: 1000,
		Asset:   collateral.TokenAsset,
	}
	eng.accs.EXPECT().GetTotalTokens().Times(1).Return(acc.Balance)
	eng.accs.EXPECT().GetPartyTokenAccount(partyID).Times(1).Return(&acc, nil)
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
	})

	original := types.Proposal{
		ID:        "prop-1",
		Reference: "test-prop-1",
		PartyID:   partyID,
		State:     types.Proposal_STATE_OPEN,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:   now.Add(100 * time.Hour).Unix(),
			EnactmentTimestamp: now.Add(240 * time.Hour).Unix(),
		},
	}
	err := eng.SubmitProposal(original)
	assert.NoError(t, err)

	aCopy := original
	aCopy.Reference = "this-is-a-copy"
	err = eng.SubmitProposal(aCopy)
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalIsDuplicate, err.Error())
}

func testProposerStake(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	party := "party1"
	eng.buf.EXPECT().Add(gomock.Any()).Times(2).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_REJECTED, p.State)
	})
	eng.accs.EXPECT().GetTotalTokens().Times(2).Return(uint64(1))

	notFoundError := errors.New("account not found")
	eng.accs.EXPECT().GetPartyTokenAccount(party).Times(1).Return(nil, notFoundError)

	err := eng.SubmitProposal(types.Proposal{
		ID:        "account-not-found",
		Reference: "1",
		PartyID:   party,
		State:     types.Proposal_STATE_OPEN,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:   time.Now().Add(3 * 24 * time.Hour).Unix(),
			EnactmentTimestamp: time.Now().Add(3 * 24 * time.Hour).Unix(),
		},
	})
	assert.Error(t, err)
	assert.EqualError(t, err, notFoundError.Error())

	emptyAccount := types.Account{
		Id:      "emptyAccount",
		Owner:   party,
		Balance: 0,
		Asset:   collateral.TokenAsset,
	}
	eng.accs.EXPECT().GetPartyTokenAccount(party).Times(1).Return(&emptyAccount, nil)

	err = eng.SubmitProposal(types.Proposal{
		ID:        "empty-account",
		Reference: "2",
		PartyID:   party,
		State:     types.Proposal_STATE_OPEN,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:   time.Now().Add(3 * 24 * time.Hour).Unix(),
			EnactmentTimestamp: time.Now().Add(3 * 24 * time.Hour).Unix(),
		},
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalInsufficientTokens.Error())

	validAccount := types.Account{
		Id:      "validAccount",
		Owner:   party,
		Balance: 1,
		Asset:   collateral.TokenAsset,
	}
	eng.accs.EXPECT().GetPartyTokenAccount(party).Times(1).Return(&validAccount, nil)
	goodProposalID := "good-prop-account-with-min-allowed-stake"
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, goodProposalID, p.ID)
	})

	err = eng.SubmitProposal(types.Proposal{
		ID:        goodProposalID,
		Reference: "3",
		PartyID:   party,
		State:     types.Proposal_STATE_OPEN,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:   time.Now().Add(3 * 24 * time.Hour).Unix(),
			EnactmentTimestamp: time.Now().Add(3 * 24 * time.Hour).Unix(),
		},
	})
	assert.NoError(t, err)
}

func testClosingTime(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	party := "party1"

	account := types.Account{
		Id:      "account",
		Owner:   party,
		Balance: 1,
		Asset:   collateral.TokenAsset,
	}

	eng.accs.EXPECT().GetPartyTokenAccount(party).Times(3).Return(&account, nil)

	eng.buf.EXPECT().Add(gomock.Any()).Times(2).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_REJECTED, p.State)
	})
	eng.accs.EXPECT().GetTotalTokens().Times(3).Return(uint64(1))

	now := time.Now()
	err := eng.SubmitProposal(types.Proposal{
		ID:        "before-what-network-param-allows",
		Reference: "1",
		PartyID:   party,
		State:     types.Proposal_STATE_OPEN,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:   now.Unix(),
			EnactmentTimestamp: now.Add(300 * time.Hour).Unix(),
		},
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalCloseTimeTooSoon.Error())

	err = eng.SubmitProposal(types.Proposal{
		ID:        "after-what-network-param-allows",
		PartyID:   party,
		Reference: "2",
		State:     types.Proposal_STATE_OPEN,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:   now.Add(3 * 365 * 24 * time.Hour).Unix(),
			EnactmentTimestamp: now.Add(300 * time.Hour).Unix(),
		},
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalCloseTimeTooLate.Error())

	goodProposalID := "good-prop"
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, goodProposalID, p.ID)
	})
	err = eng.SubmitProposal(types.Proposal{
		ID:        goodProposalID,
		Reference: "3",
		PartyID:   party,
		State:     types.Proposal_STATE_OPEN,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:   now.Add(3 * 24 * time.Hour).Unix(),
			EnactmentTimestamp: now.Add(3 * 24 * time.Hour).Unix(),
		},
	})
	assert.NoError(t, err)
}

func testEnactmentTime(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	party := "party1"

	account := types.Account{
		Id:      "account",
		Owner:   party,
		Balance: 1,
		Asset:   collateral.TokenAsset,
	}

	eng.accs.EXPECT().GetPartyTokenAccount(party).Times(4).Return(&account, nil)
	eng.buf.EXPECT().Add(gomock.Any()).Times(2).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_REJECTED, p.State)
	})
	eng.accs.EXPECT().GetTotalTokens().Times(4).Return(uint64(1))

	now := time.Now()
	err := eng.SubmitProposal(types.Proposal{
		ID:        "before-closing-time",
		PartyID:   party,
		Reference: "1",
		State:     types.Proposal_STATE_OPEN,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:   now.Add(3 * 24 * time.Hour).Unix(),
			EnactmentTimestamp: now.Unix(),
		},
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalEnactTimeTooSoon.Error())

	err = eng.SubmitProposal(types.Proposal{
		ID:        "too-late",
		PartyID:   party,
		Reference: "2",
		State:     types.Proposal_STATE_OPEN,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:   now.Add(3 * 24 * time.Hour).Unix(),
			EnactmentTimestamp: now.Add(3 * 365 * 24 * time.Hour).Unix(),
		},
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalEnactTimeTooLate.Error())

	goodProposalID1 := "good-prop-at-closing"
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, goodProposalID1, p.ID)
	})

	err = eng.SubmitProposal(types.Proposal{
		ID:        goodProposalID1,
		PartyID:   party,
		Reference: "3",
		State:     types.Proposal_STATE_OPEN,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:   now.Add(3 * 24 * time.Hour).Unix(),
			EnactmentTimestamp: now.Add(3 * 24 * time.Hour).Unix(),
		},
	})
	assert.NoError(t, err)

	goodProposalID2 := "good-prop-after-closing"
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, goodProposalID2, p.ID)
	})

	err = eng.SubmitProposal(types.Proposal{
		ID:        goodProposalID2,
		PartyID:   party,
		Reference: "4",
		State:     types.Proposal_STATE_OPEN,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:   now.Add(3 * 24 * time.Hour).Unix(),
			EnactmentTimestamp: now.Add(5 * 24 * time.Hour).Unix(),
		},
	})
	assert.NoError(t, err)
}

func TestVoteValidation(t *testing.T) {
	t.Run("Test voter stake validation", testVoterStake)
	t.Run("Test proposal id on a vote", testVoteProposalID)
	t.Run("Test voting on a declined proposal", testVotingDeclinedProposal)
	t.Run("Test voting on a passed proposal", testVotingPassedProposal)
}

func makeGoodProposal(t *testing.T, eng *tstEngine) *types.Proposal {
	proposer := "proposer"
	account := types.Account{
		Id:      "account",
		Owner:   proposer,
		Balance: 1,
		Asset:   collateral.TokenAsset,
	}
	eng.accs.EXPECT().GetPartyTokenAccount(proposer).Times(1).Return(&account, nil)
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
	})
	proposal := &types.Proposal{
		ID:        "good-proposal",
		Reference: "1",
		PartyID:   proposer,
		State:     types.Proposal_STATE_OPEN,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:   time.Now().Add(3 * 24 * time.Hour).Unix(),
			EnactmentTimestamp: time.Now().Add(5 * 24 * time.Hour).Unix(),
			Change:             newValidMarketTerms(),
		},
	}
	err := eng.SubmitProposal(*proposal)
	assert.NoError(t, err)
	return proposal
}

func testVoterStake(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	eng.accs.EXPECT().GetTotalTokens().Times(3).Return(uint64(1))

	proposal := makeGoodProposal(t, eng)

	voter := "voter"
	notFoundError := errors.New("account not found")
	eng.accs.EXPECT().GetPartyTokenAccount(voter).Times(1).Return(nil, notFoundError)
	err := eng.AddVote(types.Vote{
		PartyID:    voter,
		Value:      types.Vote_VALUE_YES, // does not matter
		ProposalID: proposal.ID,
	})
	assert.Error(t, err)
	assert.EqualError(t, err, notFoundError.Error())

	emptyAccount := types.Account{
		Id:      "emptyAccount",
		Owner:   voter,
		Balance: 0,
		Asset:   collateral.TokenAsset,
	}
	eng.accs.EXPECT().GetPartyTokenAccount(voter).Times(1).Return(&emptyAccount, nil)
	err = eng.AddVote(types.Vote{
		PartyID:    voter,
		Value:      types.Vote_VALUE_YES, // does not matter
		ProposalID: proposal.ID,
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrVoterInsufficientTokens.Error())

	validAccount := types.Account{
		Id:      "validAccount",
		Owner:   voter,
		Balance: 1,
		Asset:   collateral.TokenAsset,
	}
	eng.accs.EXPECT().GetPartyTokenAccount(voter).Times(1).Return(&validAccount, nil)

	eng.vbuf.EXPECT().Add(gomock.Any()).Times(1).Do(func(vote types.Vote) {
		assert.Equal(t, proposal.ID, vote.ProposalID)
		assert.Equal(t, voter, vote.PartyID)
	})
	err = eng.AddVote(types.Vote{
		PartyID:    voter,
		Value:      types.Vote_VALUE_YES, // does not matter
		ProposalID: proposal.ID,
	})
	assert.NoError(t, err)
}

func testVoteProposalID(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// GetTotalTokens called for every SubmitProposal(2), AddVote(2)
	eng.accs.EXPECT().GetTotalTokens().Times(4).Return(uint64(2))

	voter := "voter"
	account := types.Account{
		Id:      "account",
		Owner:   voter,
		Balance: 1,
		Asset:   collateral.TokenAsset,
	}
	eng.accs.EXPECT().GetPartyTokenAccount(voter).Times(3).Return(&account, nil)

	err := eng.AddVote(types.Vote{
		PartyID:    voter,
		Value:      types.Vote_VALUE_YES, // does not matter
		ProposalID: "id-of-non-existent-proposal",
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalNotFound.Error())

	proposer := "proposer"
	notFoundError := errors.New("account not found")
	eng.accs.EXPECT().GetPartyTokenAccount(proposer).Times(1).Return(nil, notFoundError)
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_REJECTED, p.State)
	})
	badProposal := &types.Proposal{
		ID:        "bad-proposal",
		Reference: "baddy",
		PartyID:   proposer,
		State:     types.Proposal_STATE_OPEN,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:   time.Now().Add(3 * 24 * time.Hour).Unix(),
			EnactmentTimestamp: time.Now().Add(3 * 24 * time.Hour).Unix(),
		},
	}
	err = eng.SubmitProposal(*badProposal)
	assert.Error(t, err)
	assert.EqualError(t, err, notFoundError.Error())

	err = eng.AddVote(types.Vote{
		PartyID:    voter,
		Value:      types.Vote_VALUE_YES, // does not matter
		ProposalID: badProposal.ID,
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalNotFound.Error())

	proposal := makeGoodProposal(t, eng)
	eng.vbuf.EXPECT().Add(gomock.Any()).Times(1).Do(func(vote types.Vote) {
		assert.Equal(t, proposal.ID, vote.ProposalID)
		assert.Equal(t, voter, vote.PartyID)
	})
	err = eng.AddVote(types.Vote{
		PartyID:    voter,
		Value:      types.Vote_VALUE_YES, // does not matter
		ProposalID: proposal.ID,
	})
	assert.NoError(t, err)
}

func testVotingDeclinedProposal(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// GetTotalTokens called for every SubmitProposal(1), AddVote(1)
	eng.accs.EXPECT().GetTotalTokens().Times(2).Return(uint64(2))

	proposal := makeGoodProposal(t, eng)
	eng.accs.EXPECT().GetTotalTokens().Times(1).Return(uint64(1))
	afterClose := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Hour)
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_DECLINED, p.State)
	})
	accepted := eng.OnChainTimeUpdate(afterClose)
	assert.Empty(t, accepted)

	voter := "voter"
	account := types.Account{
		Id:      "account",
		Owner:   voter,
		Balance: 1,
		Asset:   collateral.TokenAsset,
	}
	eng.accs.EXPECT().GetPartyTokenAccount(voter).Times(1).Return(&account, nil)
	err := eng.AddVote(types.Vote{
		PartyID:    voter,
		Value:      types.Vote_VALUE_YES, // does not matter
		ProposalID: proposal.ID,
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalNotFound.Error())
}

func testVotingPassedProposal(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// GetTotalTokens called for every SubmitProposal(1), AddVote(2)
	eng.accs.EXPECT().GetTotalTokens().Times(3).Return(uint64(3))

	proposal := makeGoodProposal(t, eng)

	voter1 := "voter1"
	account1 := types.Account{
		Id:      "account1",
		Owner:   voter1,
		Balance: 2,
		Asset:   collateral.TokenAsset,
	}
	eng.accs.EXPECT().GetPartyTokenAccount(voter1).Times(2).Return(&account1, nil)
	eng.vbuf.EXPECT().Add(gomock.Any()).Times(1).Do(func(vote types.Vote) {
		assert.Equal(t, proposal.ID, vote.ProposalID)
		assert.Equal(t, voter1, vote.PartyID)
	})
	err := eng.AddVote(types.Vote{
		PartyID:    voter1,
		Value:      types.Vote_VALUE_YES, // matters!
		ProposalID: proposal.ID,
	})
	assert.NoError(t, err)

	eng.accs.EXPECT().GetTotalTokens().Times(1).Return(uint64(3))
	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Hour)
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_PASSED, p.State)
	})
	accepted := eng.OnChainTimeUpdate(afterEnactment)
	assert.NotEmpty(t, accepted)

	voter2 := "voter"
	account2 := types.Account{
		Id:      "account2",
		Owner:   voter2,
		Balance: 1,
		Asset:   collateral.TokenAsset,
	}
	eng.accs.EXPECT().GetPartyTokenAccount(voter2).Times(1).Return(&account2, nil)
	err = eng.AddVote(types.Vote{
		PartyID:    voter2,
		Value:      types.Vote_VALUE_NO, // does not matter
		ProposalID: proposal.ID,
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalNotFound.Error())
}

func TestTimeUpdate(t *testing.T) {
	t.Run("Accepted proposal on time update", testProposalAccepted)
}
func testProposalAccepted(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()
	partyID := "party1"
	now := time.Now()
	acc := types.Account{
		Id:      "acc-1",
		Owner:   partyID,
		Balance: 1000,
		Asset:   collateral.TokenAsset,
	}
	partyID2 := "party2"
	acc2 := types.Account{
		Id:      "acc-2",
		Owner:   partyID2,
		Balance: 100,
		Asset:   collateral.TokenAsset,
	}
	closeTime := now.Add(100 * time.Hour)
	totalTokens := acc.Balance + acc2.Balance
	prop := types.Proposal{
		ID:        "prop-1",
		Reference: "test-prop-1",
		PartyID:   partyID,
		State:     types.Proposal_STATE_OPEN,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:   closeTime.Unix(),
			EnactmentTimestamp: closeTime.Unix(),
		},
	}
	calls := 0
	states := []types.Proposal_State{
		types.Proposal_STATE_OPEN,
		types.Proposal_STATE_PASSED,
	}
	eng.buf.EXPECT().Add(gomock.Any()).Times(2).Do(func(p types.Proposal) {
		assert.Equal(t, states[calls], p.State)
		calls++
	})

	// GetTotalTokens called for every SubmitProposal(1), AddVote(2) and OnChainTimeUpdate(1)
	eng.accs.EXPECT().GetTotalTokens().Times(4).Return(totalTokens)
	eng.accs.EXPECT().GetPartyTokenAccount(partyID).Times(1).Return(&acc, nil) // only stake holders can propose
	err := eng.SubmitProposal(prop)
	assert.NoError(t, err)
	vote := types.Vote{
		PartyID:    partyID,
		Value:      types.Vote_VALUE_YES,
		ProposalID: prop.ID,
	}
	eng.vbuf.EXPECT().Add(gomock.Any()).Times(2)
	eng.accs.EXPECT().GetPartyTokenAccount(partyID).Times(1).Return(&acc, nil) // only stake holders can vote
	assert.NoError(t, eng.AddVote(vote))

	vote.PartyID = partyID2
	vote.Value = types.Vote_VALUE_NO
	eng.accs.EXPECT().GetPartyTokenAccount(partyID2).Times(1).Return(&acc2, nil) // only stake holders can vote
	assert.NoError(t, eng.AddVote(vote))

	eng.accs.EXPECT().GetPartyTokenAccount(partyID).Times(1).Return(&acc, nil)   // to count votes
	eng.accs.EXPECT().GetPartyTokenAccount(partyID2).Times(1).Return(&acc2, nil) // to count votes
	// simulate block time update triggering end of proposal vote
	accepted := eng.OnChainTimeUpdate(closeTime.Add(time.Hour))
	assert.NotEmpty(t, accepted)
}

func getTestEngine(t *testing.T) *tstEngine {
	ctrl := gomock.NewController(t)
	cfg := governance.NewDefaultConfig()
	accs := mocks.NewMockAccounts(ctrl)
	buf := mocks.NewMockBuffer(ctrl)
	vbuf := mocks.NewMockVoteBuf(ctrl)
	eng := governance.NewEngine(logging.NewTestLogger(), cfg, governance.DefaultNetworkParameters(), accs, buf, vbuf, time.Now())
	return &tstEngine{
		Engine: eng,
		ctrl:   ctrl,
		accs:   accs,
		buf:    buf,
		vbuf:   vbuf,
	}
}

func newValidMarketTerms() *types.ProposalTerms_NewMarket {
	return &types.ProposalTerms_NewMarket{
		NewMarket: &types.NewMarket{
			Changes: &types.Market{
				Id:            "a-unit-test-market",
				DecimalPlaces: 5,
				Name:          "a-unit-test-market-name",
				TradingMode: &types.Market_Continuous{
					Continuous: &types.ContinuousTrading{
						TickSize: 0,
					},
				},
				TradableInstrument: &types.TradableInstrument{
					Instrument: &types.Instrument{
						Id:        "Crypto/GBPVUSD/Futures/Jun20",
						Code:      "CRYPTO:GBPVUSD/JUN20",
						Name:      "June 2020 GBP vs VUSD future",
						BaseName:  "GBP",
						QuoteName: "VUSD",
						Metadata: &types.InstrumentMetadata{
							Tags: []string{"asset_class:fx/crypto", "product:futures"},
						},
						InitialMarkPrice: 123321,
						Product: &types.Instrument_Future{
							Future: &types.Future{
								Maturity: "2030-06-30T22:59:59Z",
								Asset:    "VUSD",
								Oracle: &types.Future_EthereumEvent{
									EthereumEvent: &types.EthereumEvent{
										ContractID: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
										Event:      "price_changed",
									},
								},
							},
						},
					},
					MarginCalculator: &types.MarginCalculator{
						ScalingFactors: &types.ScalingFactors{
							InitialMargin:     1.2,
							CollateralRelease: 1.4,
							SearchLevel:       1.1,
						},
					},
					RiskModel: &types.TradableInstrument_LogNormalRiskModel{
						LogNormalRiskModel: &types.LogNormalRiskModel{
							RiskAversionParameter: 0.01,
							Tau:                   0.00011407711613050422,
							Params: &types.LogNormalModelParams{
								Mu:    0,
								R:     0.016,
								Sigma: 0.09,
							},
						},
					},
				},
			},
		},
	}
}
