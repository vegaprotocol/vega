package spam

import (
	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

const numProposals uint64 = 3

var minTokensForProposal, _ = num.UintFromString("100000000000000000000000", 10)

type ProposalSpamPolicy struct {
	numProposals         uint64
	minTokensForProposal *num.Uint

	partyToProposalCount      map[string]uint64           // proposals that are already on blockchain
	blockPartyToProposalCount map[string]uint64           // proposals in the current block
	tokenBalance              map[string]*num.Uint        // the balance of the party in governance tokens at the beginning of the epoch
	bannedParties             map[string]uint64           // parties banned until epoch seq
	partyBlockRejects         map[string]*blockRejectInfo // total vs rejection in the current block
	currentEpochSeq           uint64                      // current epoch sequence
}

//NewProposalSpamPolicy instantiates the proposal spam policy
func NewProposalSpamPolicy() *ProposalSpamPolicy {
	return &ProposalSpamPolicy{
		numProposals:              numProposals,
		minTokensForProposal:      minTokensForProposal,
		partyToProposalCount:      map[string]uint64{},
		blockPartyToProposalCount: map[string]uint64{},
		tokenBalance:              map[string]*num.Uint{},
		bannedParties:             map[string]uint64{},
		partyBlockRejects:         map[string]*blockRejectInfo{},
	}
}

//Reset is called when the epoch begins to reset policy state
func (psp *ProposalSpamPolicy) Reset(epoch types.Epoch, tokenBalances map[string]*num.Uint) {
	psp.currentEpochSeq = epoch.Seq

	// reset proposal counts
	psp.partyToProposalCount = map[string]uint64{}

	// update token balances
	psp.tokenBalance = make(map[string]*num.Uint, len(tokenBalances))
	for party, balance := range tokenBalances {
		psp.tokenBalance[party] = balance
	}

	// clear banned if necessary
	for party, epochSeq := range psp.bannedParties {
		if epochSeq < epoch.Seq {
			delete(psp.bannedParties, party)
		}
	}
}

//EndOfBlock is called at the end of the processing of the block to carry over state and trigger bans if necessary
func (psp *ProposalSpamPolicy) EndOfBlock(blockHeight uint64) {
	// add the block's proposal counters to the epoch's
	for party, count := range psp.blockPartyToProposalCount {
		if _, ok := psp.partyToProposalCount[party]; !ok {
			psp.partyToProposalCount[party] = 0
		}
		psp.partyToProposalCount[party] += count
	}

	// ban parties with more than <banFactor> rejection rate in the block
	for p, bStats := range psp.partyBlockRejects {
		if float64(bStats.rejected)/float64(bStats.total) >= banFactor {
			psp.bannedParties[p] = psp.currentEpochSeq + numberOfEpochsBan
		}
	}
}

//PostBlockAccept is called to verify a transaction from the block before passed to the application layer
func (psp *ProposalSpamPolicy) PostBlockAccept(tx abci.Tx) (bool, error) {
	party := tx.Party()

	// get number of proposals preceding the block in this epoch
	var epochProposals uint64 = 0
	if count, ok := psp.partyToProposalCount[party]; ok {
		epochProposals = count
	}

	// get number of votes so far in current block
	var blockProposals uint64 = 0
	if count, ok := psp.blockPartyToProposalCount[party]; ok {
		blockProposals += count
	}

	// if too many votes in total - reject and update counters
	if epochProposals+blockProposals >= psp.numProposals {
		// update vote stats for the epoch
		if partyRejectStats, ok := psp.partyBlockRejects[party]; ok {
			partyRejectStats.add(true)
		} else {
			psp.partyBlockRejects[party] = &blockRejectInfo{total: 1, rejected: 1}
		}
		return false, ErrTooManyProposals
	}

	// update vote counters for party/proposal votes
	if _, ok := psp.blockPartyToProposalCount[party]; !ok {
		psp.blockPartyToProposalCount[party] = 0
	}
	psp.blockPartyToProposalCount[party]++

	// update party and block stats
	if partyRejectStats, ok := psp.partyBlockRejects[party]; ok {
		partyRejectStats.add(false)
	} else {
		psp.partyBlockRejects[party] = &blockRejectInfo{total: 1, rejected: 0}
	}
	return true, nil

}

//PreBlockAccept checks if the proposal violates spam rules based on the information we had about the number of existing proposals preceding the current block
func (psp *ProposalSpamPolicy) PreBlockAccept(tx abci.Tx) (bool, error) {
	party := tx.Party()

	// check if the party is banned
	_, ok := psp.bannedParties[party]
	if ok {
		return false, ErrPartyIsBannedFromProposal
	}

	// check if the party has enough balance to submit proposals
	if balance, ok := psp.tokenBalance[party]; !ok || balance.LT(psp.minTokensForProposal) {
		return false, ErrInsufficientTokensForProposal
	}

	// Check we have not exceeded our proposal limit for this given party in this epoch
	if proposalCount, ok := psp.partyToProposalCount[party]; ok && proposalCount >= psp.numProposals {
		return false, ErrTooManyProposals
	}

	return true, nil
}
