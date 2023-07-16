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
	"sync"

	protoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"

	"code.vegaprotocol.io/vega/core/blockchain/abci"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
)

// Simple spam policy supports encforcing of max allowed commands and min required tokens + banning of parties when their reject rate in the block
// exceeds x%.
type SimpleSpamPolicy struct {
	log                *logging.Logger
	accounts           StakingAccounts
	policyName         string
	maxAllowedCommands uint64
	minTokensRequired  *num.Uint

	minTokensParamName  string
	maxAllowedParamName string

	partyToCount          map[string]uint64 // commands that are already on blockchain
	blockPartyToCount     map[string]uint64 // commands in the current block
	currentEpochSeq       uint64            // current epoch sequence
	lock                  sync.RWMutex      // global lock to sync calls from multiple tendermint threads
	insufficientTokensErr error
	tooManyCommands       error
}

// NewSimpleSpamPolicy instantiates the simple spam policy.
func NewSimpleSpamPolicy(policyName string, minTokensParamName string, maxAllowedParamName string, log *logging.Logger, accounts StakingAccounts) *SimpleSpamPolicy {
	return &SimpleSpamPolicy{
		log:                   log,
		accounts:              accounts,
		policyName:            policyName,
		partyToCount:          map[string]uint64{},
		blockPartyToCount:     map[string]uint64{},
		lock:                  sync.RWMutex{},
		minTokensParamName:    minTokensParamName,
		maxAllowedParamName:   maxAllowedParamName,
		minTokensRequired:     num.UintZero(),
		maxAllowedCommands:    1, // default is allow one per epoch
		insufficientTokensErr: errors.New("party has insufficient associated governance tokens in their staking account to submit " + policyName + " request"),
		tooManyCommands:       errors.New("party has already submitted the maximum number of " + policyName + " requests per epoch"),
	}
}

func (ssp *SimpleSpamPolicy) Serialise() ([]byte, error) {
	partyToCount := []*types.PartyCount{}
	for party, count := range ssp.partyToCount {
		partyToCount = append(partyToCount, &types.PartyCount{
			Party: party,
			Count: count,
		})
	}

	sort.SliceStable(partyToCount, func(i, j int) bool { return partyToCount[i].Party < partyToCount[j].Party })

	payload := types.Payload{
		Data: &types.PayloadSimpleSpamPolicy{
			SimpleSpamPolicy: &types.SimpleSpamPolicy{
				PolicyName:      ssp.policyName,
				PartyToCount:    partyToCount,
				CurrentEpochSeq: ssp.currentEpochSeq,
			},
		},
	}

	return proto.Marshal(payload.IntoProto())
}

func (ssp *SimpleSpamPolicy) Deserialise(p *types.Payload) error {
	pl := p.Data.(*types.PayloadSimpleSpamPolicy).SimpleSpamPolicy

	ssp.partyToCount = map[string]uint64{}
	for _, ptc := range pl.PartyToCount {
		ssp.partyToCount[ptc.Party] = ptc.Count
	}
	ssp.currentEpochSeq = pl.CurrentEpochSeq

	return nil
}

// UpdateUintParam is called to update Uint net params for the policy
// Specifically the min tokens required for executing the command for which the policy is attached.
func (ssp *SimpleSpamPolicy) UpdateUintParam(name string, value *num.Uint) error {
	if name == ssp.minTokensParamName {
		ssp.minTokensRequired = value.Clone()
	} else {
		return errors.New("unknown parameter for simple spam policy")
	}
	return nil
}

// UpdateIntParam is called to update int net params for the policy
// Specifically the number of commands a party can submit in an epoch.
func (ssp *SimpleSpamPolicy) UpdateIntParam(name string, value int64) error {
	if name == ssp.maxAllowedParamName {
		ssp.maxAllowedCommands = uint64(value)
	} else {
		return errors.New("unknown parameter for simple spam policy")
	}
	return nil
}

// Reset is called when the epoch begins to reset policy state.
func (ssp *SimpleSpamPolicy) Reset(epoch types.Epoch) {
	ssp.lock.Lock()
	defer ssp.lock.Unlock()
	ssp.currentEpochSeq = epoch.Seq

	// reset counts
	ssp.partyToCount = map[string]uint64{}
	ssp.blockPartyToCount = map[string]uint64{}
}

func (ssp *SimpleSpamPolicy) UpdateTx(tx abci.Tx) {
	ssp.lock.Lock()
	defer ssp.lock.Unlock()
	if _, ok := ssp.partyToCount[tx.Party()]; !ok {
		ssp.partyToCount[tx.Party()] = 0
	}
	ssp.partyToCount[tx.Party()]++
}

// PostBlockAccept is called to verify a transaction from the block before passed to the application layer.
func (ssp *SimpleSpamPolicy) CheckBlockTx(tx abci.Tx) error {
	party := tx.Party()

	ssp.lock.Lock()
	defer ssp.lock.Unlock()

	// get number of commands preceding the block in this epoch
	var epochCommands uint64
	if count, ok := ssp.partyToCount[party]; ok {
		epochCommands = count
	}

	// get number of votes so far in current block
	var blockCommands uint64
	if count, ok := ssp.blockPartyToCount[party]; ok {
		blockCommands += count
	}

	// if too many votes in total - reject and update counters
	if epochCommands+blockCommands >= ssp.maxAllowedCommands {
		return ssp.tooManyCommands
	}

	// update block counters
	if _, ok := ssp.blockPartyToCount[party]; !ok {
		ssp.blockPartyToCount[party] = 0
	}
	ssp.blockPartyToCount[party]++

	return nil
}

func (ssp *SimpleSpamPolicy) RollbackProposal() {
	ssp.blockPartyToCount = map[string]uint64{}
}

// PreBlockAccept checks if the commands violates spam rules based on the information we had about the number of existing commands preceding the current block.
func (ssp *SimpleSpamPolicy) PreBlockAccept(tx abci.Tx) error {
	party := tx.Party()

	ssp.lock.RLock()
	defer ssp.lock.RUnlock()

	// check if the party has enough balance to submit commands
	balance, err := ssp.accounts.GetAvailableBalance(party)
	if !ssp.minTokensRequired.IsZero() && (err != nil || balance.LT(ssp.minTokensRequired)) {
		if ssp.log.GetLevel() <= logging.DebugLevel {
			ssp.log.Debug("Spam pre: party has insufficient balance for "+ssp.policyName, logging.String("txHash", hex.EncodeToString(tx.Hash())), logging.String("party", party), logging.String("balance", num.UintToString(balance)))
		}
		return ssp.insufficientTokensErr
	}

	// Check we have not exceeded our command limit for this given party in this epoch
	if commandCount, ok := ssp.partyToCount[party]; ok && commandCount >= ssp.maxAllowedCommands {
		if ssp.log.GetLevel() <= logging.DebugLevel {
			ssp.log.Debug("Spam pre: party has already submitted the max amount of commands for "+ssp.policyName, logging.String("txHash", hex.EncodeToString(tx.Hash())), logging.String("party", party), logging.Uint64("count", commandCount), logging.Uint64("maxAllowed", ssp.maxAllowedCommands))
		}
		return ssp.tooManyCommands
	}

	return nil
}

func (ssp *SimpleSpamPolicy) GetSpamStats(party string) *protoapi.SpamStatistic {
	ssp.lock.RLock()
	defer ssp.lock.RUnlock()
	return &protoapi.SpamStatistic{
		CountForEpoch:     ssp.partyToCount[party],
		MaxForEpoch:       ssp.maxAllowedCommands,
		MinTokensRequired: ssp.minTokensRequired.String(),
	}
}

func (ssp *SimpleSpamPolicy) GetVoteSpamStats(_ string) *protoapi.VoteSpamStatistics {
	return nil
}
