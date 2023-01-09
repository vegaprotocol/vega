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
	"sync"
	"time"

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

	partyToCount          map[string]uint64           // commands that are already on blockchain
	blockPartyToCount     map[string]uint64           // commands in the current block
	bannedParties         map[string]int64            // parties banned -> ban end time
	partyBlockRejects     map[string]*blockRejectInfo // total vs rejection in the current block
	currentEpochSeq       uint64                      // current epoch sequence
	lock                  sync.RWMutex                // global lock to sync calls from multiple tendermint threads
	banErr                func(until time.Time) error
	insufficientTokensErr error
	tooManyCommands       error
}

// NewSimpleSpamPolicy instantiates the simple spam policy.
func NewSimpleSpamPolicy(policyName string, minTokensParamName string, maxAllowedParamName string, log *logging.Logger, accounts StakingAccounts) *SimpleSpamPolicy {
	return &SimpleSpamPolicy{
		log:                 log,
		accounts:            accounts,
		policyName:          policyName,
		partyToCount:        map[string]uint64{},
		blockPartyToCount:   map[string]uint64{},
		bannedParties:       map[string]int64{},
		partyBlockRejects:   map[string]*blockRejectInfo{},
		lock:                sync.RWMutex{},
		minTokensParamName:  minTokensParamName,
		maxAllowedParamName: maxAllowedParamName,
		minTokensRequired:   num.UintZero(),
		maxAllowedCommands:  1, // default is allow one per epoch
		banErr: func(until time.Time) error {
			return errors.New("party is banned from submitting " + policyName + " until the earlier between " + until.String() + " and the beginning of the next epoch")
		},
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

	bannedParties := make([]*types.BannedParty, 0, len(ssp.bannedParties))
	for party, until := range ssp.bannedParties {
		bannedParties = append(bannedParties, &types.BannedParty{
			Party: party,
			Until: until,
		})
	}

	sort.SliceStable(bannedParties, func(i, j int) bool { return bannedParties[i].Party < bannedParties[j].Party })

	payload := types.Payload{
		Data: &types.PayloadSimpleSpamPolicy{
			SimpleSpamPolicy: &types.SimpleSpamPolicy{
				PolicyName:      ssp.policyName,
				PartyToCount:    partyToCount,
				BannedParty:     bannedParties,
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
	ssp.bannedParties = make(map[string]int64, len(pl.BannedParty))
	for _, bp := range pl.BannedParty {
		ssp.bannedParties[bp.Party] = bp.Until
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

	// clear banned on new epoch
	ssp.bannedParties = map[string]int64{}

	ssp.blockPartyToCount = map[string]uint64{}
	ssp.partyBlockRejects = map[string]*blockRejectInfo{}
}

// EndOfBlock is called at the end of the processing of the block to carry over state and trigger bans if necessary.
func (ssp *SimpleSpamPolicy) EndOfBlock(blockHeight uint64, now time.Time, banDuration time.Duration) {
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

	// release bans
	nowNano := now.UnixNano()
	for k, v := range ssp.bannedParties {
		if nowNano >= v {
			delete(ssp.bannedParties, k)
		}
	}

	endBanTime := now.Add(banDuration).UnixNano()

	// ban parties with more than <banFactor> rejection rate in the block
	for p, bStats := range ssp.partyBlockRejects {
		if num.DecimalFromInt64(int64(bStats.rejected)).Div(num.DecimalFromInt64(int64(bStats.total))).GreaterThanOrEqual(banFactor) {
			ssp.bannedParties[p] = endBanTime
		}
	}
	ssp.partyBlockRejects = map[string]*blockRejectInfo{}
}

// PostBlockAccept is called to verify a transaction from the block before passed to the application layer.
func (ssp *SimpleSpamPolicy) PostBlockAccept(tx abci.Tx) (bool, error) {
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
		// update vote stats for the epoch
		if partyRejectStats, ok := ssp.partyBlockRejects[party]; ok {
			partyRejectStats.add(true)
		} else {
			ssp.partyBlockRejects[party] = &blockRejectInfo{total: 1, rejected: 1}
		}
		if ssp.log.GetLevel() <= logging.DebugLevel {
			ssp.log.Debug("Spam post: party has already submitted the max amount of commands for "+ssp.policyName, logging.String("txHash", hex.EncodeToString(tx.Hash())), logging.String("party", party))
		}
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

// PreBlockAccept checks if the commands violates spam rules based on the information we had about the number of existing commands preceding the current block.
func (ssp *SimpleSpamPolicy) PreBlockAccept(tx abci.Tx) (bool, error) {
	party := tx.Party()

	ssp.lock.RLock()
	defer ssp.lock.RUnlock()

	// check if the party is banned
	until, ok := ssp.bannedParties[party]
	if ok {
		if ssp.log.GetLevel() <= logging.DebugLevel {
			ssp.log.Debug("Spam pre: party is banned from "+ssp.policyName, logging.String("txHash", hex.EncodeToString(tx.Hash())), logging.String("party", party))
		}
		return false, ssp.banErr(time.Unix(0, until).UTC())
	}

	// check if the party has enough balance to submit commands
	balance, err := ssp.accounts.GetAvailableBalance(party)
	if !ssp.minTokensRequired.IsZero() && (err != nil || balance.LT(ssp.minTokensRequired)) {
		if ssp.log.GetLevel() <= logging.DebugLevel {
			ssp.log.Debug("Spam pre: party has insufficient balance for "+ssp.policyName, logging.String("txHash", hex.EncodeToString(tx.Hash())), logging.String("party", party), logging.String("balance", num.UintToString(balance)))
		}
		return false, ssp.insufficientTokensErr
	}

	// Check we have not exceeded our command limit for this given party in this epoch
	if commandCount, ok := ssp.partyToCount[party]; ok && commandCount >= ssp.maxAllowedCommands {
		if ssp.log.GetLevel() <= logging.DebugLevel {
			ssp.log.Debug("Spam pre: party has already submitted the max amount of commands for "+ssp.policyName, logging.String("txHash", hex.EncodeToString(tx.Hash())), logging.String("party", party), logging.Uint64("count", commandCount), logging.Uint64("maxAllowed", ssp.maxAllowedCommands))
		}
		return false, ssp.tooManyCommands
	}

	return true, nil
}

func (ssp *SimpleSpamPolicy) GetStats(party string) []Statistic {
	ssp.lock.RLock()
	defer ssp.lock.RUnlock()

	return []Statistic{
		{
			Total:       strconv.FormatUint(ssp.partyToCount[party], formatBase),
			Limit:       strconv.FormatUint(ssp.maxAllowedCommands, formatBase),
			BannedUntil: ssp.bannedParties[party],
		},
	}
}
