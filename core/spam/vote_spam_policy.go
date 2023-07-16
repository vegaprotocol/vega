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

package spam

import (
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"sync"

	protoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"

	"code.vegaprotocol.io/vega/core/blockchain/abci"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

var (
	// ErrInsufficientTokensForVoting is returned when the party has insufficient tokens for voting.
	ErrInsufficientTokensForVoting = errors.New("party has insufficient associated governance tokens in their staking account to submit votes")
	// ErrTooManyVotes is returned when the party has voted already the maximum allowed votes per proposal per epoch.
	ErrTooManyVotes = errors.New("party has already voted the maximum number of times per proposal per epoch")
)

type VoteSpamPolicy struct {
	log             *logging.Logger
	numVotes        uint64
	minVotingTokens *num.Uint
	accounts        StakingAccounts

	minTokensParamName  string
	maxAllowedParamName string
	partyToVote         map[string]map[string]uint64 // those are votes that are already on blockchain
	blockPartyToVote    map[string]map[string]uint64 // votes in the current block
	currentEpochSeq     uint64                       // the sequence id of the current epoch
	lock                sync.RWMutex                 // global lock to sync calls from multiple tendermint threads
}

// NewVoteSpamPolicy instantiates vote spam policy.
func NewVoteSpamPolicy(minTokensParamName string, maxAllowedParamName string, log *logging.Logger, accounts StakingAccounts) *VoteSpamPolicy {
	return &VoteSpamPolicy{
		log:                 log,
		accounts:            accounts,
		partyToVote:         map[string]map[string]uint64{},
		blockPartyToVote:    map[string]map[string]uint64{},
		lock:                sync.RWMutex{},
		minTokensParamName:  minTokensParamName,
		maxAllowedParamName: maxAllowedParamName,
	}
}

func (vsp *VoteSpamPolicy) Serialise() ([]byte, error) {
	partyProposalVoteCount := []*types.PartyProposalVoteCount{}
	for party, proposalToCount := range vsp.partyToVote {
		for proposal, count := range proposalToCount {
			partyProposalVoteCount = append(partyProposalVoteCount, &types.PartyProposalVoteCount{
				Party:    party,
				Proposal: proposal,
				Count:    count,
			})
		}
	}

	sort.SliceStable(partyProposalVoteCount, func(i, j int) bool {
		switch strings.Compare(partyProposalVoteCount[i].Party, partyProposalVoteCount[j].Party) {
		case -1:
			return true
		case 1:
			return false
		}
		return partyProposalVoteCount[i].Proposal < partyProposalVoteCount[j].Proposal
	})

	payload := types.Payload{
		Data: &types.PayloadVoteSpamPolicy{
			VoteSpamPolicy: &types.VoteSpamPolicy{
				PartyProposalVoteCount: partyProposalVoteCount,
				CurrentEpochSeq:        vsp.currentEpochSeq,
			},
		},
	}

	return proto.Marshal(payload.IntoProto())
}

func (vsp *VoteSpamPolicy) Deserialise(p *types.Payload) error {
	pl := p.Data.(*types.PayloadVoteSpamPolicy).VoteSpamPolicy
	vsp.partyToVote = map[string]map[string]uint64{}
	for _, ptv := range pl.PartyProposalVoteCount {
		if _, ok := vsp.partyToVote[ptv.Party]; !ok {
			vsp.partyToVote[ptv.Party] = map[string]uint64{}
		}
		vsp.partyToVote[ptv.Party][ptv.Proposal] = ptv.Count
	}
	vsp.currentEpochSeq = pl.CurrentEpochSeq
	return nil
}

// UpdateUintParam is called to update Uint net params for the policy
// Specifically the min tokens required for voting.
func (vsp *VoteSpamPolicy) UpdateUintParam(name string, value *num.Uint) error {
	if name == vsp.minTokensParamName {
		vsp.minVotingTokens = value.Clone()
	} else {
		return errors.New("unknown parameter for vote spam policy")
	}
	return nil
}

// UpdateIntParam is called to update iint net params for the policy
// Specifically the number of votes to a proposal a party can submit in an epoch.
func (vsp *VoteSpamPolicy) UpdateIntParam(name string, value int64) error {
	if name == vsp.maxAllowedParamName {
		vsp.numVotes = uint64(value)
	} else {
		return errors.New("unknown parameter for vote spam policy")
	}
	return nil
}

// Reset is called at the beginning of an epoch to reset the settings for the epoch.
func (vsp *VoteSpamPolicy) Reset(epoch types.Epoch) {
	vsp.lock.Lock()
	defer vsp.lock.Unlock()
	// reset the token count factor to 1
	vsp.currentEpochSeq = epoch.Seq

	// reset vote counts
	vsp.partyToVote = map[string]map[string]uint64{}

	// reset current block vote counts
	vsp.blockPartyToVote = map[string]map[string]uint64{}
}

func (vsp *VoteSpamPolicy) UpdateTx(tx abci.Tx) {
	vsp.lock.Lock()
	defer vsp.lock.Unlock()
	if _, ok := vsp.partyToVote[tx.Party()]; !ok {
		vsp.partyToVote[tx.Party()] = map[string]uint64{}
	}
	vote := &commandspb.VoteSubmission{}
	tx.Unmarshal(vote)
	if _, ok := vsp.partyToVote[tx.Party()][vote.ProposalId]; !ok {
		vsp.partyToVote[tx.Party()][vote.ProposalId] = 0
	}
	vsp.partyToVote[tx.Party()][vote.ProposalId] = vsp.partyToVote[tx.Party()][vote.ProposalId] + 1
}

func (vsp *VoteSpamPolicy) RollbackProposal() {
	vsp.blockPartyToVote = map[string]map[string]uint64{}
}

func (vsp *VoteSpamPolicy) CheckBlockTx(tx abci.Tx) error {
	party := tx.Party()

	vsp.lock.Lock()
	defer vsp.lock.Unlock()

	vote := &commandspb.VoteSubmission{}
	if err := tx.Unmarshal(vote); err != nil {
		return err
	}

	// get number of votes preceding the block in this epoch
	var epochVotes uint64
	if partyVotes, ok := vsp.partyToVote[party]; ok {
		if voteCount, ok := partyVotes[vote.ProposalId]; ok {
			epochVotes = voteCount
		}
	}

	// get number of votes so far in current block
	var blockVotes uint64
	if proposals, ok := vsp.blockPartyToVote[party]; ok {
		if votes, ok := proposals[vote.ProposalId]; ok {
			blockVotes += votes
		}
	}

	// if too many votes in total - reject and update counters
	if epochVotes+blockVotes >= vsp.numVotes {
		return ErrTooManyVotes
	}

	// update vote counters for party/proposal votes
	if _, ok := vsp.blockPartyToVote[party]; !ok {
		vsp.blockPartyToVote[party] = map[string]uint64{}
	}
	if votes, ok := vsp.blockPartyToVote[party][vote.ProposalId]; !ok {
		vsp.blockPartyToVote[party][vote.ProposalId] = 1
	} else {
		vsp.blockPartyToVote[party][vote.ProposalId] = votes + 1
	}
	return nil
}

// PreBlockAccept checks if the vote should be rejected as spam or not based on the number of votes in current epoch's preceding blocks and the number of tokens
// held by the party.
// NB: this is done at mempool before adding to block.
func (vsp *VoteSpamPolicy) PreBlockAccept(tx abci.Tx) error {
	party := tx.Party()

	vsp.lock.RLock()
	defer vsp.lock.RUnlock()

	// check if the party has enough balance to submit votes
	balance, err := vsp.accounts.GetAvailableBalance(party)
	if err != nil || balance.LT(vsp.minVotingTokens) {
		if vsp.log.GetLevel() <= logging.DebugLevel {
			vsp.log.Debug("Spam pre: party has insufficient balance for voting", logging.String("txHash", hex.EncodeToString(tx.Hash())), logging.String("party", party), logging.String("balance", num.UintToString(balance)))
		}
		return ErrInsufficientTokensForVoting
	}

	vote := &commandspb.VoteSubmission{}

	if err := tx.Unmarshal(vote); err != nil {
		return err
	}

	// Check we have not exceeded our vote limit for this given proposal in this epoch
	if partyVotes, ok := vsp.partyToVote[party]; ok {
		if voteCount, ok := partyVotes[vote.ProposalId]; ok && voteCount >= vsp.numVotes {
			if vsp.log.GetLevel() <= logging.DebugLevel {
				vsp.log.Debug("Spam pre: party has already voted for proposal the max amount of votes", logging.String("txHash", hex.EncodeToString(tx.Hash())), logging.String("party", party), logging.String("proposal", vote.ProposalId), logging.Uint64("voteCount", voteCount), logging.Uint64("maxAllowed", vsp.numVotes))
			}
			return ErrTooManyVotes
		}
	}

	return nil
}

func (vsp *VoteSpamPolicy) GetSpamStats(_ string) *protoapi.SpamStatistic {
	return nil
}

func (vsp *VoteSpamPolicy) GetVoteSpamStats(partyID string) *protoapi.VoteSpamStatistics {
	vsp.lock.RLock()
	defer vsp.lock.RUnlock()

	partyStats := vsp.partyToVote[partyID]

	stats := make([]*protoapi.VoteSpamStatistic, 0, len(partyStats))

	for proposal, votes := range partyStats {
		stats = append(stats, &protoapi.VoteSpamStatistic{
			Proposal:          proposal,
			CountForEpoch:     votes,
			MinTokensRequired: vsp.minVotingTokens.String(),
		})
	}
	return &protoapi.VoteSpamStatistics{
		Statistics:  stats,
		MaxForEpoch: vsp.numVotes,
	}
}
