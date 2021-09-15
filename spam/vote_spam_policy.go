package spam

import (
	"errors"
	"sync"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
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

type VoteSpamPolicy struct {
	log             *logging.Logger
	numVotes        uint64
	minVotingTokens *num.Uint

	minTokensParamName  string
	maxAllowedParamName string

	minVotingTokensFactor   *num.Uint                                        // a factor applied on the min voting tokens
	effectiveMinTokens      *num.Uint                                        // minVotingFactor * minVotingTokens
	partyToVote             map[string]map[string]uint64                     // those are votes that are already on blockchain
	blockPartyToVote        map[string]map[string]uint64                     // votes in the current block
	bannedParties           map[string]uint64                                // parties banned until epoch seq
	tokenBalance            map[string]*num.Uint                             // the balance of the party in governance tokens at the beginning of the epoch
	recentBlocksRejectStats [numberOfBlocksForIncreaseCheck]*blockRejectInfo // recent blocks post rejection stats
	blockPostRejects        *blockRejectInfo                                 // this blocks post reject stats
	partyBlockRejects       map[string]*blockRejectInfo                      // total vs rejection in the current block
	currentBlockIndex       int                                              // the index of the current block in the circular buffer <recentBlocksRejectStats>
	lastIncreaseBlock       uint64                                           // the last block we've increased the number of <minVotingTokens>
	currentEpochSeq         uint64                                           // the sequence id of the current epoch
	lock                    sync.RWMutex                                     // global lock to sync calls from multiple tendermint threads
}

//NewVoteSpamPolicy instantiates vote spam policy
func NewVoteSpamPolicy(minTokensParamName string, maxAllowedParamName string, log *logging.Logger) *VoteSpamPolicy {
	return &VoteSpamPolicy{
		log:                   log,
		minVotingTokensFactor: num.NewUint(1),

		partyToVote:         map[string]map[string]uint64{},
		blockPartyToVote:    map[string]map[string]uint64{},
		bannedParties:       map[string]uint64{},
		tokenBalance:        map[string]*num.Uint{},
		blockPostRejects:    &blockRejectInfo{total: 0, rejected: 0},
		partyBlockRejects:   map[string]*blockRejectInfo{},
		currentBlockIndex:   0,
		lastIncreaseBlock:   0,
		lock:                sync.RWMutex{},
		minTokensParamName:  minTokensParamName,
		maxAllowedParamName: maxAllowedParamName,
	}
}

//UpdateUintParam is called to update Uint net params for the policy
//Specifically the min tokens required for voting
func (vsp *VoteSpamPolicy) UpdateUintParam(name string, value *num.Uint) error {
	if name == vsp.minTokensParamName {
		vsp.minVotingTokens = value.Clone()
		//NB: this means that if during the epoch the min tokens changes externally
		// and we already have a factor on it, the factor will be applied on the new value for the duration of the epoch
		vsp.effectiveMinTokens = num.Zero().Mul(vsp.minVotingTokens, vsp.minVotingTokensFactor)
	} else {
		return errors.New("unknown parameter for vote spam policy")
	}
	return nil
}

//UpdateIntParam is called to update iint net params for the policy
//Specifically the number of votes to a proposal a party can submit in an epoch
func (vsp *VoteSpamPolicy) UpdateIntParam(name string, value int64) error {
	if name == vsp.maxAllowedParamName {
		vsp.numVotes = uint64(value)
	} else {
		return errors.New("unknown parameter for vote spam policy")
	}
	return nil
}

//Reset is called at the beginning of an epoch to reset the settings for the epoch
func (vsp *VoteSpamPolicy) Reset(epoch types.Epoch, tokenBalances map[string]*num.Uint) {
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
	for i := 0; i < numberOfBlocksForIncreaseCheck; i++ {
		vsp.recentBlocksRejectStats[i] = nil
	}

	// clear banned if necessary
	for party, epochSeq := range vsp.bannedParties {
		if epochSeq < epoch.Seq {
			delete(vsp.bannedParties, party)
		}
	}
	// update token balances
	vsp.tokenBalance = make(map[string]*num.Uint, len(tokenBalances))
	for party, balance := range tokenBalances {
		vsp.tokenBalance[party] = balance.Clone()
	}

	vsp.blockPostRejects = &blockRejectInfo{
		total:    0,
		rejected: 0,
	}
}

//EndOfBlock is called at the end of the block to allow updating of the state for the next block
func (vsp *VoteSpamPolicy) EndOfBlock(blockHeight uint64) {
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

	// ban parties with more than <banFactor> rejection rate in the block
	for p, bStats := range vsp.partyBlockRejects {
		if float64(bStats.rejected)/float64(bStats.total) >= banFactor {
			vsp.bannedParties[p] = vsp.currentEpochSeq + numberOfEpochsBan
		}
	}

	// add the block rejects to the last 10 blocks
	vsp.recentBlocksRejectStats[vsp.currentBlockIndex] = vsp.blockPostRejects
	vsp.currentBlockIndex++
	vsp.currentBlockIndex %= numberOfBlocksForIncreaseCheck

	// check if we need to increase the limits, i.e. if we're below the max and we've not increased in the last n blocks
	if (vsp.lastIncreaseBlock == 0 || blockHeight > vsp.lastIncreaseBlock+uint64(numberOfBlocksForIncreaseCheck)) && num.Zero().Mul(vsp.minVotingTokens, vsp.minVotingTokensFactor).LT(maxMinVotingTokens) {
		average := vsp.calcRejectAverage()
		if average > rejectRatioForIncrease {
			vsp.lastIncreaseBlock = blockHeight
			vsp.minVotingTokensFactor = num.Zero().Mul(vsp.minVotingTokensFactor, increaseFactor)
			vsp.effectiveMinTokens = num.Zero().Mul(vsp.minVotingTokensFactor, vsp.minVotingTokens)
		}
	}
}

// calculate the mean rejection rate in the last <numberOfBlocksForIncreaseCheck>
func (vsp *VoteSpamPolicy) calcRejectAverage() float64 {
	var total uint64 = 0
	var rejected uint64 = 0
	for i := 0; i < numberOfBlocksForIncreaseCheck; i++ {
		if vsp.recentBlocksRejectStats[i] != nil {
			total += vsp.recentBlocksRejectStats[i].total
			rejected += vsp.recentBlocksRejectStats[i].rejected
		}
	}
	return float64(rejected) / float64(total)
}

//PostBlockAccept checks if votes that made it to the block should be rejected based on the number of votes preceding the block + votes seen in the block
//NB: this is called as part of the processing of the block
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
	var epochVotes uint64 = 0
	if partyVotes, ok := vsp.partyToVote[party]; ok {
		if voteCount, ok := partyVotes[vote.ProposalId]; ok {
			epochVotes = voteCount
		}
	}

	// get number of votes so far in current block
	var blockVotes uint64 = 0
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
		vsp.log.Error("Spam post: party has already voted for proposal the max amount of votes", logging.String("party", party), logging.String("proposal", vote.ProposalId))

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

//PreBlockAccept checks if the vote should be rejected as spam or not based on the number of votes in current epoch's preceding blocks and the number of tokens
//held by the party.
//NB: this is done at mempool before adding to block
func (vsp *VoteSpamPolicy) PreBlockAccept(tx abci.Tx) (bool, error) {
	party := tx.Party()

	vsp.lock.RLock()
	defer vsp.lock.RUnlock()

	_, ok := vsp.bannedParties[party]
	if ok {
		vsp.log.Error("Spam pre: party is banned from voting", logging.String("party", party))
		return false, ErrPartyIsBannedFromVoting
	}

	// check if the party has enough balance to submit votes
	if balance, ok := vsp.tokenBalance[party]; !ok || balance.LT(vsp.effectiveMinTokens) {
		vsp.log.Error("Spam pre: party has insufficient balance for voting", logging.String("balance", num.UintToString(balance)))
		return false, ErrInsufficientTokensForVoting
	}

	vote := &commandspb.VoteSubmission{}

	if err := tx.Unmarshal(vote); err != nil {
		return false, err
	}

	// Check we have not exceeded our vote limit for this given proposal in this epoch
	if partyVotes, ok := vsp.partyToVote[party]; ok {
		if voteCount, ok := partyVotes[vote.ProposalId]; ok && voteCount >= vsp.numVotes {
			vsp.log.Error("Spam pre: party has already voted for proposal the max amount of votes", logging.String("party", party), logging.String("proposal", vote.ProposalId))
			return false, ErrTooManyVotes
		}
	}

	return true, nil
}
