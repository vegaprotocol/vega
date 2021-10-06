package spam

import (
	"errors"
	"sync"

	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

// Simple spam policy supports encforcing of max allowed commands and min required tokens + banning of parties when their reject rate in the block
// exceeds x%.
type SimpleSpamPolicy struct {
	log                *logging.Logger
	policyName         string
	maxAllowedCommands uint64
	minTokensRequired  *num.Uint

	minTokensParamName  string
	maxAllowedParamName string

	partyToCount          map[string]uint64           // commands that are already on blockchain
	blockPartyToCount     map[string]uint64           // commands in the current block
	tokenBalance          map[string]*num.Uint        // the balance of the party in governance tokens at the beginning of the epoch
	bannedParties         map[string]uint64           // parties banned until epoch seq
	partyBlockRejects     map[string]*blockRejectInfo // total vs rejection in the current block
	currentEpochSeq       uint64                      // current epoch sequence
	lock                  sync.RWMutex                // global lock to sync calls from multiple tendermint threads
	banErr                error
	insufficientTokensErr error
	tooManyCommands       error
}

//NewSimpleSpamPolicy instantiates the simple spam policy
func NewSimpleSpamPolicy(policyName string, minTokensParamName string, maxAllowedParamName string, log *logging.Logger) *SimpleSpamPolicy {
	return &SimpleSpamPolicy{
		log:                   log,
		policyName:            policyName,
		partyToCount:          map[string]uint64{},
		blockPartyToCount:     map[string]uint64{},
		tokenBalance:          map[string]*num.Uint{},
		bannedParties:         map[string]uint64{},
		partyBlockRejects:     map[string]*blockRejectInfo{},
		lock:                  sync.RWMutex{},
		minTokensParamName:    minTokensParamName,
		maxAllowedParamName:   maxAllowedParamName,
		banErr:                errors.New("party is banned from submitting " + policyName + " in the current epoch"),
		insufficientTokensErr: errors.New("party has insufficient tokens to submit " + policyName + " request in this epoch"),
		tooManyCommands:       errors.New("party has already proposed the maximum number of " + policyName + " requests per epoch"),
	}
}

//UpdateUintParam is called to update Uint net params for the policy
//Specifically the min tokens required for executing the command for which the policy is attached
func (ssp *SimpleSpamPolicy) UpdateUintParam(name string, value *num.Uint) error {
	if name == ssp.minTokensParamName {
		ssp.minTokensRequired = value.Clone()
	} else {
		return errors.New("unknown parameter for simple spam policy")
	}
	return nil
}

//UpdateIntParam is called to update int net params for the policy
//Specifically the number of commands a party can submit in an epoch
func (ssp *SimpleSpamPolicy) UpdateIntParam(name string, value int64) error {
	if name == ssp.maxAllowedParamName {
		ssp.maxAllowedCommands = uint64(value)
	} else {
		return errors.New("unknown parameter for simple spam policy")
	}
	return nil
}

//Reset is called when the epoch begins to reset policy state
func (ssp *SimpleSpamPolicy) Reset(epoch types.Epoch, tokenBalances map[string]*num.Uint) {
	ssp.lock.Lock()
	defer ssp.lock.Unlock()
	ssp.currentEpochSeq = epoch.Seq

	// reset counts
	ssp.partyToCount = map[string]uint64{}

	// update token balances
	ssp.tokenBalance = make(map[string]*num.Uint, len(tokenBalances))
	for party, balance := range tokenBalances {
		ssp.tokenBalance[party] = balance
	}

	// clear banned if necessary
	for party, epochSeq := range ssp.bannedParties {
		if epochSeq < epoch.Seq {
			delete(ssp.bannedParties, party)
		}
	}
}

//EndOfBlock is called at the end of the processing of the block to carry over state and trigger bans if necessary
func (ssp *SimpleSpamPolicy) EndOfBlock(blockHeight uint64) {
	ssp.lock.Lock()
	defer ssp.lock.Unlock()
	// add the block's counters to the epoch's
	for party, count := range ssp.blockPartyToCount {
		if _, ok := ssp.partyToCount[party]; !ok {
			ssp.partyToCount[party] = 0
		}
		ssp.partyToCount[party] += count
	}

	ssp.blockPartyToCount = map[string]uint64{}

	// ban parties with more than <banFactor> rejection rate in the block
	for p, bStats := range ssp.partyBlockRejects {
		if float64(bStats.rejected)/float64(bStats.total) >= banFactor {
			ssp.bannedParties[p] = ssp.currentEpochSeq + numberOfEpochsBan
		}
	}
}

//PostBlockAccept is called to verify a transaction from the block before passed to the application layer
func (ssp *SimpleSpamPolicy) PostBlockAccept(tx abci.Tx) (bool, error) {
	party := tx.Party()

	ssp.lock.Lock()
	defer ssp.lock.Unlock()

	// get number of commands preceding the block in this epoch
	var epochCommands uint64 = 0
	if count, ok := ssp.partyToCount[party]; ok {
		epochCommands = count
	}

	// get number of votes so far in current block
	var blockCommands uint64 = 0
	if count, ok := ssp.blockPartyToCount[party]; ok {
		blockCommands += count
	}

	// if too many votes in total - reject and update counters
	if epochCommands+blockCommands >= ssp.maxAllowedCommands {
		// update vote stats for the epoch
		if partyRejectStats, ok := ssp.partyBlockRejects[party]; ok {
			partyRejectStats.add(true)
		} else {
			ssp.partyBlockRejects[party] = &blockRejectInfo{total: 1, rejected: 1}
		}
		ssp.log.Error("Spam post: party has already submitted the max amount of commands for "+ssp.policyName, logging.String("party", party))
		return false, ssp.tooManyCommands
	}

	// update block counters
	if _, ok := ssp.blockPartyToCount[party]; !ok {
		ssp.blockPartyToCount[party] = 0
	}
	ssp.blockPartyToCount[party]++

	// update party and block stats
	if partyRejectStats, ok := ssp.partyBlockRejects[party]; ok {
		partyRejectStats.add(false)
	} else {
		ssp.partyBlockRejects[party] = &blockRejectInfo{total: 1, rejected: 0}
	}
	return true, nil

}

//PreBlockAccept checks if the commands violates spam rules based on the information we had about the number of existing commands preceding the current block
func (ssp *SimpleSpamPolicy) PreBlockAccept(tx abci.Tx) (bool, error) {
	party := tx.Party()

	ssp.lock.RLock()
	defer ssp.lock.RUnlock()

	// check if the party is banned
	_, ok := ssp.bannedParties[party]
	if ok {
		ssp.log.Error("Spam pre: party is banned from "+ssp.policyName, logging.String("party", party))
		return false, ssp.banErr
	}

	// check if the party has enough balance to submit commands
	if balance, ok := ssp.tokenBalance[party]; !ok || balance.LT(ssp.minTokensRequired) {
		ssp.log.Error("Spam pre: party has insufficient balance for "+ssp.policyName, logging.String("party", party), logging.String("balance", num.UintToString(balance)))
		return false, ssp.insufficientTokensErr
	}

	// Check we have not exceeded our command limit for this given party in this epoch
	if commandCount, ok := ssp.partyToCount[party]; ok && commandCount >= ssp.maxAllowedCommands {
		ssp.log.Error("Spam pre: party has already submitted the max amount of commands for "+ssp.policyName, logging.String("party", party), logging.Uint64("count", commandCount), logging.Uint64("maxAllowed", ssp.maxAllowedCommands))
		return false, ssp.tooManyCommands
	}

	return true, nil
}
