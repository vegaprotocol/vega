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
	"strconv"
	"strings"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/blockchain/abci"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

type blockRejectInfo struct {
	rejected uint64
	total    uint64
}

func (b *blockRejectInfo) add(rejected bool) {
	b.total++
	if rejected {
		b.rejected++
	}
}

var maxMinVotingTokens, _ = num.UintFromString("1600000000000000000000", 10)
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

	minVotingTokensFactor   *num.Uint                                        // a factor applied on the min voting tokens
	effectiveMinTokens      *num.Uint                                        // minVotingFactor * minVotingTokens
	partyToVote             map[string]map[string]uint64                     // those are votes that are already on blockchain
	blockPartyToVote        map[string]map[string]uint64                     // votes in the current block
	bannedParties           map[string]int64                                 // parties banned -> ban end time
	recentBlocksRejectStats [numberOfBlocksForIncreaseCheck]*blockRejectInfo // recent blocks post rejection stats
	blockPostRejects        *blockRejectInfo                                 // this blocks post reject stats
	partyBlockRejects       map[string]*blockRejectInfo                      // total vs rejection in the current block
	currentBlockIndex       uint64                                           // the index of the current block in the circular buffer <recentBlocksRejectStats>
	lastIncreaseBlock       uint64                                           // the last block we've increased the number of <minVotingTokens>
	currentEpochSeq         uint64                                           // the sequence id of the current epoch
	lock                    sync.RWMutex                                     // global lock to sync calls from multiple tendermint threads
}

// NewVoteSpamPolicy instantiates vote spam policy.
func NewVoteSpamPolicy(minTokensParamName string, maxAllowedParamName string, log *logging.Logger, accounts StakingAccounts) *VoteSpamPolicy {
	return &VoteSpamPolicy{
		log:                   log,
		accounts:              accounts,
		minVotingTokensFactor: num.NewUint(1),

		partyToVote:         map[string]map[string]uint64{},
		blockPartyToVote:    map[string]map[string]uint64{},
		bannedParties:       map[string]int64{},
		blockPostRejects:    &blockRejectInfo{total: 0, rejected: 0},
		partyBlockRejects:   map[string]*blockRejectInfo{},
		currentBlockIndex:   0,
		lastIncreaseBlock:   0,
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

	bannedParties := make([]*types.BannedParty, 0, len(vsp.bannedParties))
	for party, until := range vsp.bannedParties {
		bannedParties = append(bannedParties, &types.BannedParty{
			Party: party,
			Until: until,
		})
	}

	sort.SliceStable(bannedParties, func(i, j int) bool { return bannedParties[i].Party < bannedParties[j].Party })

	recentRejects := make([]*types.BlockRejectStats, 0, len(vsp.recentBlocksRejectStats))
	for _, brs := range vsp.recentBlocksRejectStats {
		if brs != nil {
			recentRejects = append(recentRejects, &types.BlockRejectStats{
				Total:    brs.total,
				Rejected: brs.rejected,
			})
		}
	}

	payload := types.Payload{
		Data: &types.PayloadVoteSpamPolicy{
			VoteSpamPolicy: &types.VoteSpamPolicy{
				PartyProposalVoteCount:  partyProposalVoteCount,
				BannedParty:             bannedParties,
				RecentBlocksRejectStats: recentRejects,
				CurrentBlockIndex:       vsp.currentBlockIndex,
				LastIncreaseBlock:       vsp.lastIncreaseBlock,
				CurrentEpochSeq:         vsp.currentEpochSeq,
				MinVotingTokensFactor:   vsp.minVotingTokensFactor,
			},
		},
	}

	return proto.Marshal(payload.IntoProto())
}

func (vsp *VoteSpamPolicy) Deserialise(p *types.Payload) error {
	pl := p.Data.(*types.PayloadVoteSpamPolicy).VoteSpamPolicy

	var i uint64
	for ; i < numberOfBlocksForIncreaseCheck; i++ {
		vsp.recentBlocksRejectStats[i] = nil
	}
	for j, bl := range pl.RecentBlocksRejectStats {
		vsp.recentBlocksRejectStats[j] = &blockRejectInfo{
			total:    bl.Total,
			rejected: bl.Rejected,
		}
	}

	vsp.partyToVote = map[string]map[string]uint64{}
	for _, ptv := range pl.PartyProposalVoteCount {
		if _, ok := vsp.partyToVote[ptv.Party]; !ok {
			vsp.partyToVote[ptv.Party] = map[string]uint64{}
		}
		vsp.partyToVote[ptv.Party][ptv.Proposal] = ptv.Count
	}
	vsp.bannedParties = make(map[string]int64, len(pl.BannedParty))
	for _, bp := range pl.BannedParty {
		vsp.bannedParties[bp.Party] = bp.Until
	}

	vsp.currentEpochSeq = pl.CurrentEpochSeq
	vsp.lastIncreaseBlock = pl.LastIncreaseBlock
	vsp.currentBlockIndex = pl.CurrentBlockIndex
	vsp.minVotingTokensFactor = pl.MinVotingTokensFactor
	vsp.effectiveMinTokens = num.UintZero().Mul(vsp.minVotingTokens, vsp.minVotingTokensFactor)

	return nil
}

// UpdateUintParam is called to update Uint net params for the policy
// Specifically the min tokens required for voting.
func (vsp *VoteSpamPolicy) UpdateUintParam(name string, value *num.Uint) error {
	if name == vsp.minTokensParamName {
		vsp.minVotingTokens = value.Clone()
		// NB: this means that if during the epoch the min tokens changes externally
		// and we already have a factor on it, the factor will be applied on the new value for the duration of the epoch
		vsp.effectiveMinTokens = num.UintZero().Mul(vsp.minVotingTokens, vsp.minVotingTokensFactor)
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
	vsp.minVotingTokensFactor = num.NewUint(1)
	vsp.effectiveMinTokens = vsp.minVotingTokens
	vsp.currentEpochSeq = epoch.Seq

	// set last increase to 0 so we'd check right away on the next block
	vsp.lastIncreaseBlock = 0

	// reset vote counts
	vsp.partyToVote = map[string]map[string]uint64{}

	// reset current block vote counts
	vsp.blockPartyToVote = map[string]map[string]uint64{}

	// reset block stats
	vsp.currentBlockIndex = 0
	var i uint64
	for ; i < numberOfBlocksForIncreaseCheck; i++ {
		vsp.recentBlocksRejectStats[i] = nil
	}

	// clear banned
	vsp.bannedParties = map[string]int64{}

	// reset block rejects - this is not essential here as it's cleared at the end of every block anyways
	// but just for consistency
	vsp.partyBlockRejects = map[string]*blockRejectInfo{}
	vsp.blockPostRejects = &blockRejectInfo{
		total:    0,
		rejected: 0,
	}
}

// EndOfBlock is called at the end of the block to allow updating of the state for the next block.
func (vsp *VoteSpamPolicy) EndOfBlock(blockHeight uint64, now time.Time, banDuration time.Duration) {
	vsp.lock.Lock()
	defer vsp.lock.Unlock()
	// add the block's vote counters to the epoch's
	for p, v := range vsp.blockPartyToVote {
		if _, ok := vsp.partyToVote[p]; !ok {
			vsp.partyToVote[p] = map[string]uint64{}
		}

		for proposalID, votes := range v {
			if _, ok := vsp.partyToVote[p][proposalID]; !ok {
				vsp.partyToVote[p][proposalID] = 0
			}
			vsp.partyToVote[p][proposalID] = vsp.partyToVote[p][proposalID] + votes
		}
	}

	vsp.blockPartyToVote = map[string]map[string]uint64{}

	// release bans
	nowNano := now.UnixNano()
	for k, v := range vsp.bannedParties {
		if nowNano >= v {
			delete(vsp.bannedParties, k)
		}
	}

	endBanTime := now.Add(banDuration).UnixNano()

	// ban parties with more than <banFactor> rejection rate in the block
	for p, bStats := range vsp.partyBlockRejects {
		if num.DecimalFromInt64(int64(bStats.rejected)).Div(num.DecimalFromInt64(int64(bStats.total))).GreaterThanOrEqual(banFactor) {
			vsp.bannedParties[p] = endBanTime
		}
	}
	vsp.partyBlockRejects = map[string]*blockRejectInfo{}

	// add the block rejects to the last 10 blocks
	vsp.recentBlocksRejectStats[vsp.currentBlockIndex] = &blockRejectInfo{
		rejected: vsp.blockPostRejects.rejected,
		total:    vsp.blockPostRejects.total,
	}
	vsp.currentBlockIndex++
	vsp.currentBlockIndex %= numberOfBlocksForIncreaseCheck
	vsp.blockPostRejects = &blockRejectInfo{
		rejected: 0,
		total:    0,
	}

	// check if we need to increase the limits, i.e. if we're below the max and we've not increased in the last n blocks
	if (vsp.lastIncreaseBlock == 0 || blockHeight > vsp.lastIncreaseBlock+numberOfBlocksForIncreaseCheck) && num.UintZero().Mul(vsp.minVotingTokens, vsp.minVotingTokensFactor).LT(maxMinVotingTokens) {
		average := vsp.calcRejectAverage()
		if average > rejectRatioForIncrease {
			vsp.lastIncreaseBlock = blockHeight
			vsp.minVotingTokensFactor = num.UintZero().Mul(vsp.minVotingTokensFactor, increaseFactor)
			vsp.effectiveMinTokens = num.UintZero().Mul(vsp.minVotingTokensFactor, vsp.minVotingTokens)
		}
	}
}

// calculate the mean rejection rate in the last <numberOfBlocksForIncreaseCheck>.
func (vsp *VoteSpamPolicy) calcRejectAverage() float64 {
	var total uint64
	var rejected uint64
	var i uint64
	for ; i < numberOfBlocksForIncreaseCheck; i++ {
		if vsp.recentBlocksRejectStats[i] != nil {
			total += vsp.recentBlocksRejectStats[i].total
			rejected += vsp.recentBlocksRejectStats[i].rejected
		}
	}
	return float64(rejected) / float64(total)
}

// PostBlockAccept checks if votes that made it to the block should be rejected based on the number of votes preceding the block + votes seen in the block
// NB: this is called as part of the processing of the block.
func (vsp *VoteSpamPolicy) PostBlockAccept(tx abci.Tx) (bool, error) {
	party := tx.Party()

	vsp.lock.Lock()
	defer vsp.lock.Unlock()

	vote := &commandspb.VoteSubmission{}
	if err := tx.Unmarshal(vote); err != nil {
		vsp.blockPostRejects.add(true)
		return false, err
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
		// update party/proposal vote stats for the epoch
		vsp.blockPostRejects.add(true)
		// update vote stats for the epoch
		if partyRejectStats, ok := vsp.partyBlockRejects[party]; ok {
			partyRejectStats.add(true)
		} else {
			vsp.partyBlockRejects[party] = &blockRejectInfo{total: 1, rejected: 1}
		}
		if vsp.log.GetLevel() <= logging.DebugLevel {
			vsp.log.Debug("Spam post: party has already voted for proposal the max amount of votes", logging.String("txHash", hex.EncodeToString(tx.Hash())), logging.String("party", party), logging.String("proposal", vote.ProposalId), logging.Uint64("voteCount", epochVotes+blockVotes), logging.Uint64("maxAllowed", vsp.numVotes))
		}

		return false, ErrTooManyVotes
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

	// update party and block stats
	if partyRejectStats, ok := vsp.partyBlockRejects[party]; ok {
		partyRejectStats.add(false)
	} else {
		vsp.partyBlockRejects[party] = &blockRejectInfo{total: 1, rejected: 0}
	}
	vsp.blockPostRejects.add(false)
	return true, nil
}

// PreBlockAccept checks if the vote should be rejected as spam or not based on the number of votes in current epoch's preceding blocks and the number of tokens
// held by the party.
// NB: this is done at mempool before adding to block.
func (vsp *VoteSpamPolicy) PreBlockAccept(tx abci.Tx) (bool, error) {
	party := tx.Party()

	vsp.lock.RLock()
	defer vsp.lock.RUnlock()

	until, ok := vsp.bannedParties[party]
	if ok {
		if vsp.log.GetLevel() <= logging.DebugLevel {
			vsp.log.Debug("Spam pre: party is banned from voting", logging.String("txHash", hex.EncodeToString(tx.Hash())), logging.String("party", party))
		}
		return false, errors.New("party is banned from submitting votes until the earlier between " + time.Unix(0, until).UTC().String() + " and the beginning of the next epoch")
	}

	// check if the party has enough balance to submit votes
	balance, err := vsp.accounts.GetAvailableBalance(party)
	if err != nil || balance.LT(vsp.effectiveMinTokens) {
		if vsp.log.GetLevel() <= logging.DebugLevel {
			vsp.log.Debug("Spam pre: party has insufficient balance for voting", logging.String("txHash", hex.EncodeToString(tx.Hash())), logging.String("party", party), logging.String("balance", num.UintToString(balance)))
		}
		return false, ErrInsufficientTokensForVoting
	}

	vote := &commandspb.VoteSubmission{}

	if err := tx.Unmarshal(vote); err != nil {
		return false, err
	}

	// Check we have not exceeded our vote limit for this given proposal in this epoch
	if partyVotes, ok := vsp.partyToVote[party]; ok {
		if voteCount, ok := partyVotes[vote.ProposalId]; ok && voteCount >= vsp.numVotes {
			if vsp.log.GetLevel() <= logging.DebugLevel {
				vsp.log.Debug("Spam pre: party has already voted for proposal the max amount of votes", logging.String("txHash", hex.EncodeToString(tx.Hash())), logging.String("party", party), logging.String("proposal", vote.ProposalId), logging.Uint64("voteCount", voteCount), logging.Uint64("maxAllowed", vsp.numVotes))
			}
			return false, ErrTooManyVotes
		}
	}

	return true, nil
}

func (vsp *VoteSpamPolicy) GetStats(partyID string) Statistic {
	vsp.lock.RLock()
	defer vsp.lock.RUnlock()

	stats := Statistic{
		Limit: banFactor.String(),
	}

	bStats, ok := vsp.partyBlockRejects[partyID]
	if !ok {
		return stats
	}

	stats.Total = strconv.FormatUint(bStats.total, formatBase)
	stats.BlockCount = strconv.FormatUint(bStats.rejected, formatBase)
	stats.BlockedUntil = vsp.bannedParties[partyID]

	return stats
}
