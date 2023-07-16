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

package pow

import (
	"context"
	"encoding/hex"
	"errors"
	"sync"

	protoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"

	"code.vegaprotocol.io/vega/core/blockchain/abci"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
)

const (
	ringSize = 500
)

type ValidationEntry struct {
	ValResult  ValidationResult
	Difficulty *uint
	Tx         abci.Tx
}

type ValidationResult int64

const (
	ValidationResultVerificationPowError ValidationResult = iota
	ValidationResultValidatorCommand
	ValidationResultTooManyTx
	ValidationResultSuccess
)

// params defines the modifiable set of parameters to be applied at the from block and valid for transactions generated for the untilBlock.
type params struct {
	spamPoWNumberOfPastBlocks   uint64
	spamPoWDifficulty           uint
	spamPoWHashFunction         string
	spamPoWNumberOfTxPerBlock   uint64
	spamPoWIncreasingDifficulty bool
	fromBlock                   uint64
	untilBlock                  *uint64
}

// isActive for a given block height returns true if:
// 1. there is no expiration for the param set (i.e. untilBlock is nil) or
// 2. the block is within the lookback from the until block.
func (p *params) isActive(blockHeight uint64) bool {
	return p.untilBlock == nil || *p.untilBlock+p.spamPoWNumberOfPastBlocks > blockHeight
}

// represents the number of transactions seen from a party and the total observed difficulty
// of transactions generated with a given block height.
type partyStateForBlock struct {
	observedDifficulty uint
	seenCount          uint
}

type state struct {
	blockToPartyState map[uint64]map[string]*partyStateForBlock
}
type Engine struct {
	activeParams []*params // active sets of parameters
	activeStates []*state  // active states corresponding to the sets of parameters

	currentBlock uint64              // the current block height
	blockHeight  [ringSize]uint64    // block heights in scope ring buffer - this has a fixed size which is equal to the maximum value of the network parameter
	blockHash    [ringSize]string    // block hashes in scope ring buffer - this has a fixed size which is equal to the maximum value of the network parameter
	seenTx       map[string]struct{} // seen transactions in scope set
	heightToTx   map[uint64][]string // height to slice of seen transaction in scope ring buffer
	seenTid      map[string]struct{} // seen tid in scope set
	heightToTid  map[uint64][]string // height to slice of seen tid in scope ring buffer

	// mempoolSeenTid map[string]struct{} // tids seen already in this node's mempool, cleared at the end of the block

	// snapshot key
	hashKeys []string
	log      *logging.Logger
	lock     sync.RWMutex
}

// New instantiates the proof of work engine.
func New(log *logging.Logger, config Config) *Engine {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())
	e := &Engine{
		log:          log,
		hashKeys:     []string{(&types.PayloadProofOfWork{}).Key()},
		activeParams: []*params{},
		activeStates: []*state{},
		seenTx:       map[string]struct{}{},
		heightToTx:   map[uint64][]string{},
		seenTid:      map[string]struct{}{},
		// mempoolSeenTid: map[string]struct{}{},
		heightToTid: map[uint64][]string{},
	}
	e.log.Info("PoW spam protection started")
	return e
}

// OnBeginBlock updates the block height and block hash and clears any out of scope parameters set and states.
// It also records all of the block's transactions.
func (e *Engine) BeginBlock(blockHeight uint64, blockHash string, txs []abci.Tx) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.currentBlock = blockHeight
	idx := blockHeight % ringSize
	e.blockHeight[idx] = blockHeight
	e.blockHash[idx] = blockHash
	e.updatePowState(txs)
}

// CheckTx is called by checkTx in the abci and verifies the proof of work, it doesn't update any state.
func (e *Engine) CheckTx(tx abci.Tx) error {
	// if e.log.IsDebug() {
	e.lock.RLock()
	e.log.Debug("checktx got tx", logging.String("command", tx.Command().String()), logging.Uint64("height", tx.BlockHeight()), logging.String("tid", tx.GetPoWTID()), logging.Uint64("current-block", e.currentBlock))
	e.lock.RUnlock()
	// }
	// if !tx.Command().IsValidatorCommand() {
	// 	e.lock.Lock()
	// if _, ok := e.mempoolSeenTid[tx.GetPoWTID()]; ok {
	// 	e.log.Error("tid already seen", logging.String("tid", tx.GetPoWTID()), logging.String("party", tx.Party()))
	// 	e.lock.Unlock()
	// 	return errors.New("proof of work tid already seen")
	// }
	// e.mempoolSeenTid[tx.GetPoWTID()] = struct{}{}
	// 	e.lock.Unlock()
	// }

	_, err := e.verify(tx)
	if err != nil {
		e.log.Debug("checktx error", logging.String("command", tx.Command().String()), logging.Uint64("height", tx.BlockHeight()), logging.String("tid", tx.GetPoWTID()), logging.Uint64("current-block", e.currentBlock))
	}
	return err
}

// EndPrepareProposal is a callback called at the end of prepareBlock to revert to the state
// before prepare block.
func (e *Engine) EndPrepareProposal(txs []ValidationEntry) {
	e.log.Debug("EndPrepareBlock called with", logging.Int("txs", len(txs)))
	e.rollback(txs)
}

// updatePowState updates the pow state given the block transaction and cleans up out of scope states and param sets.
func (e *Engine) updatePowState(txs []abci.Tx) {
	for _, tx := range txs {
		d, _ := e.verifyWithLock(tx)
		dUint := uint(d)
		txHash := hex.EncodeToString(tx.Hash())
		txBlock := tx.BlockHeight()
		stateInd := 0
		for i, p := range e.activeParams {
			if txBlock >= p.fromBlock && (p.untilBlock == nil || *p.untilBlock >= txBlock) {
				stateInd = i
				break
			}
		}
		state := e.activeStates[stateInd]
		e.seenTx[txHash] = struct{}{}
		e.heightToTx[tx.BlockHeight()] = append(e.heightToTx[tx.BlockHeight()], txHash)
		if tx.Command().IsValidatorCommand() {
			continue
		}

		e.heightToTid[tx.BlockHeight()] = append(e.heightToTid[tx.BlockHeight()], tx.GetPoWTID())
		e.seenTid[tx.GetPoWTID()] = struct{}{}
		if _, ok := state.blockToPartyState[txBlock]; !ok {
			state.blockToPartyState[txBlock] = map[string]*partyStateForBlock{tx.Party(): {observedDifficulty: dUint, seenCount: uint(1)}}
			continue
		}
		if _, ok := state.blockToPartyState[txBlock][tx.Party()]; !ok {
			state.blockToPartyState[txBlock][tx.Party()] = &partyStateForBlock{observedDifficulty: dUint, seenCount: uint(1)}
			continue
		}
		partyState := state.blockToPartyState[txBlock][tx.Party()]
		partyState.observedDifficulty += dUint
		partyState.seenCount++
	}

	// update out of scope states/params
	toDelete := []int{}
	// iterate over parameters set and clear then out if ther's not relevant anymore.
	for i, p := range e.activeParams {
		// is active means if we're still accepting transactions from it i.e. if the untilBlock + spamPoWNumberOfPastBlocks <= blockHeight
		if !p.isActive(e.currentBlock) {
			toDelete = append(toDelete, i)
			continue
		}
	}

	for i, p := range e.activeParams {
		outOfScopeBlock := int64(e.currentBlock) + 1 - int64(p.spamPoWNumberOfPastBlocks)
		if outOfScopeBlock < 0 {
			continue
		}
		uOutOfScopeBlock := uint64(outOfScopeBlock)
		b, ok := e.heightToTx[uOutOfScopeBlock]
		if !ok {
			continue
		}
		for _, v := range b {
			delete(e.seenTx, v)
		}
		for _, v := range e.heightToTid[uOutOfScopeBlock] {
			delete(e.seenTid, v)
		}
		delete(e.heightToTx, uOutOfScopeBlock)
		delete(e.heightToTid, uOutOfScopeBlock)
		delete(e.activeStates[i].blockToPartyState, uOutOfScopeBlock)
	}

	// delete all out of scope configurations and states
	for i := len(toDelete) - 1; i >= 0; i-- {
		e.activeParams = append(e.activeParams[:toDelete[i]], e.activeParams[toDelete[i]+1:]...)
		e.activeStates = append(e.activeStates[:toDelete[i]], e.activeStates[toDelete[i]+1:]...)
	}
}

// OnFinalize is called when the finalizeBlock is completed to clenup the mempool cache.
func (e *Engine) OnFinalize() {
	// e.lock.Lock()
	// e.mempoolSeenTid = map[string]struct{}{}
	// e.lock.Unlock()
	// e.log.Info("mempool seen cleared")
	e.log.Debug("OnFinalize")
}

// rollback is called without the lock. For each input validation entry depending on its status it reverts any changes made to the interim block state.
func (e *Engine) rollback(txs []ValidationEntry) {
	for _, ve := range txs {
		e.log.Debug("rollback", logging.String("party", ve.Tx.Party()), logging.String("tx-hash", hex.EncodeToString(ve.Tx.Hash())), logging.Int64("ve-result", int64(ve.ValResult)))
		// pow error does not change state, we can skip
		if ve.ValResult == ValidationResultVerificationPowError {
			continue
		}
		txHash := hex.EncodeToString(ve.Tx.Hash())
		// remove the transaction from seenTx - need to acquire lock!
		e.lock.Lock()
		delete(e.seenTx, txHash)
		e.lock.Unlock()

		// if it's a validator command, we're done
		if ve.ValResult == ValidationResultValidatorCommand {
			continue
		}

		// otherwise need to remove the seenTid from the block state - need to acquire lock!
		e.lock.Lock()
		delete(e.seenTid, ve.Tx.GetPoWTID())
		e.lock.Unlock()

		// if the validation result is too many transactions or the difficulty is nil, nothing to revert
		if ve.ValResult == ValidationResultTooManyTx || ve.Difficulty == nil {
			continue
		}
		stateInd := 0
		txBlock := ve.Tx.BlockHeight()
		for i, p := range e.activeParams {
			if txBlock >= p.fromBlock && (p.untilBlock == nil || *p.untilBlock >= txBlock) {
				stateInd = i
				break
			}
		}
		state := e.activeStates[stateInd]
		if _, ok := state.blockToPartyState[txBlock]; !ok {
			e.log.Error("cannot find state of the block - that should be impossible")
		} else if _, ok := state.blockToPartyState[txBlock][ve.Tx.Party()]; !ok {
			e.log.Error("cannot find the party in the block state - that should be impossible")
		}

		partyState := state.blockToPartyState[txBlock][ve.Tx.Party()]
		e.log.Debug("found party state for party", logging.Bool("found", partyState != nil), logging.String("party", ve.Tx.Party()))
		partyState.seenCount--
		partyState.observedDifficulty -= *ve.Difficulty
		if partyState.seenCount == 0 {
			e.log.Debug("seen count for party is zero, removing party from block state", logging.String("party", ve.Tx.Party()))
			delete(state.blockToPartyState[txBlock], ve.Tx.Party())
		}
		if len(state.blockToPartyState[txBlock]) == 0 {
			e.log.Debug("no more transactions for block, removing block height", logging.Uint64("height", txBlock))
			delete(state.blockToPartyState, txBlock)
		}
	}
}

func (e *Engine) ProcessProposal(txs []abci.Tx) bool {
	ves := []ValidationEntry{}
	success := true
	for _, tx := range txs {
		vr, d := e.CheckBlockTx(tx)
		ves = append(ves, ValidationEntry{Tx: tx, Difficulty: d, ValResult: vr})
		if vr == ValidationResultVerificationPowError || vr == ValidationResultTooManyTx {
			success = false
			break
		}
	}
	e.rollback(ves)
	return success
}

// CheckBlockTx verifies if a transaction can be included a prepared/verified block.
func (e *Engine) CheckBlockTx(tx abci.Tx) (ValidationResult, *uint) {
	if e.log.IsDebug() {
		e.lock.RLock()
		e.log.Debug("CheckBlockTx got tx", logging.String("command", tx.Command().String()), logging.Uint64("height", tx.BlockHeight()), logging.String("tid", tx.GetPoWTID()), logging.Uint64("current-block", e.currentBlock))
		e.lock.RUnlock()
	}

	d, err := e.verify(tx)
	dUint := uint(d)
	if err != nil {
		e.log.Error("pow error", logging.Error(err))
		return ValidationResultVerificationPowError, nil
	}

	e.lock.Lock()
	defer e.lock.Unlock()

	// keep the transaction hash
	txHash := hex.EncodeToString(tx.Hash())
	txBlock := tx.BlockHeight()
	stateInd := 0
	for i, p := range e.activeParams {
		if txBlock >= p.fromBlock && (p.untilBlock == nil || *p.untilBlock >= txBlock) {
			stateInd = i
			break
		}
	}
	state := e.activeStates[stateInd]
	params := e.activeParams[stateInd]

	e.seenTx[txHash] = struct{}{}

	if tx.Command().IsValidatorCommand() {
		return ValidationResultValidatorCommand, nil
	}

	// if version supports pow, save the pow result and the tid
	e.seenTid[tx.GetPoWTID()] = struct{}{}

	// if it's the first transaction we're seeing from any party for this block height, initialise the state
	if _, ok := state.blockToPartyState[txBlock]; !ok {
		state.blockToPartyState[txBlock] = map[string]*partyStateForBlock{tx.Party(): {observedDifficulty: dUint, seenCount: uint(1)}}
		if e.log.IsDebug() {
			e.log.Debug("transaction accepted", logging.String("tid", tx.GetPoWTID()))
		}
		e.log.Debug("updated party block state", logging.Uint64("height", txBlock), logging.String("party", tx.Party()), logging.String("tx-hash", txHash))
		return ValidationResultSuccess, &dUint
	}

	// if it's the first transaction for the party for this block height
	if _, ok := state.blockToPartyState[txBlock][tx.Party()]; !ok {
		state.blockToPartyState[txBlock][tx.Party()] = &partyStateForBlock{observedDifficulty: dUint, seenCount: uint(1)}
		e.log.Debug("updated party block state", logging.Uint64("height", txBlock), logging.String("party", tx.Party()), logging.String("tx-hash", txHash))
		return ValidationResultSuccess, &dUint
	}

	// it's not the first transaction for the party for the given block height
	// if we've seen less than the allowed number of transactions per block, take a note and let it pass
	partyState := state.blockToPartyState[txBlock][tx.Party()]
	if partyState.seenCount < uint(params.spamPoWNumberOfTxPerBlock) {
		partyState.observedDifficulty += dUint
		partyState.seenCount++

		if e.log.IsDebug() {
			e.log.Debug("transaction accepted", logging.String("tid", tx.GetPoWTID()))
		}
		e.log.Debug("updated party block state", logging.Uint64("height", txBlock), logging.String("party", tx.Party()), logging.String("tx-hash", txHash))
		return ValidationResultSuccess, &dUint
	}

	// if we've seen already enough transactions and `spamPoWIncreasingDifficulty` is not enabled then fail the transaction
	if !params.spamPoWIncreasingDifficulty {
		return ValidationResultTooManyTx, nil
	}

	// calculate the expected difficulty - allow spamPoWNumberOfTxPerBlock for every level of increased difficulty
	totalExpectedDifficulty, _ := calculateExpectedDifficulty(params.spamPoWDifficulty, uint(params.spamPoWNumberOfTxPerBlock), partyState.seenCount+1)

	// if the observed difficulty sum is less than the expected difficulty, reject the tx
	if partyState.observedDifficulty+dUint < totalExpectedDifficulty {
		return ValidationResultTooManyTx, nil
	}

	partyState.observedDifficulty += dUint
	partyState.seenCount++
	e.log.Debug("updated party block state", logging.Uint64("height", txBlock), logging.String("party", tx.Party()), logging.String("tx-hash", txHash))
	return ValidationResultSuccess, &dUint
}

// calculateExpectedDifficulty calculates the expected total difficulty given the default difficulty, the max batch size and the number of seen transactions
// such that for each difficulty we allow batch size transactions.
// e.g.  spamPoWDifficulty = 5
//
//			 spamPoWNumberOfTxPerBlock = 10
//	      seenTx = 33
//
// expected difficulty = 10 * 5 + 10 * 6 + 10 * 7 + 3 * 8 = 204.
func calculateExpectedDifficulty(spamPoWDifficulty uint, spamPoWNumberOfTxPerBlock uint, seenTx uint) (uint, uint) {
	if seenTx <= spamPoWNumberOfTxPerBlock {
		if seenTx == spamPoWNumberOfTxPerBlock {
			return seenTx * spamPoWDifficulty, spamPoWDifficulty + 1
		}

		return seenTx * spamPoWDifficulty, spamPoWDifficulty
	}
	total := uint(0)
	diff := spamPoWDifficulty
	d := seenTx
	for {
		if d > spamPoWNumberOfTxPerBlock {
			total += diff * spamPoWNumberOfTxPerBlock
			d -= spamPoWNumberOfTxPerBlock
		} else {
			total += diff * d
			break
		}
		diff++
	}

	if seenTx%spamPoWNumberOfTxPerBlock == 0 {
		diff++
	}

	return total, diff
}

func (e *Engine) findParamsForBlockHeight(height uint64) int {
	paramIndex := -1
	for i, p := range e.activeParams {
		if height >= p.fromBlock && (p.untilBlock == nil || *p.untilBlock >= height) {
			paramIndex = i
		}
	}
	return paramIndex
}

func (e *Engine) verifyWithLock(tx abci.Tx) (byte, error) {
	var h byte

	// check if the transaction was seen in scope
	txHash := hex.EncodeToString(tx.Hash())
	if _, ok := e.seenTx[txHash]; ok {
		e.log.Error("replay attack: txHash already used", logging.String("tx-hash", txHash), logging.String("tid", tx.GetPoWTID()), logging.String("party", tx.Party()), logging.String("command", tx.Command().String()))
		return h, errors.New("transaction hash already used")
	}

	// validator commands skip PoW verification
	if tx.Command().IsValidatorCommand() {
		return h, nil
	}

	// check if the block height is in scope and is known

	// we need to find the parameters that is relevant to the block for which the pow was generated
	// as assume that the number of elements in the slice is small so no point in bothering with binary search
	paramIndex := e.findParamsForBlockHeight(tx.BlockHeight())
	if paramIndex < 0 {
		return h, errors.New("transaction too old")
	}

	params := e.activeParams[paramIndex]
	idx := tx.BlockHeight() % ringSize
	// if the block height doesn't match out expectation or is older than what's allowed by the parameters used for the transaction then reject
	if e.blockHeight[idx] != tx.BlockHeight() || tx.BlockHeight()+params.spamPoWNumberOfPastBlocks <= e.currentBlock {
		if e.log.IsDebug() {
			e.log.Debug("unknown block height", logging.Uint64("current-block-height", e.currentBlock), logging.String("tx-hash", txHash), logging.String("tid", tx.GetPoWTID()), logging.Uint64("tx-block-height", tx.BlockHeight()), logging.Uint64("index", idx), logging.String("command", tx.Command().String()), logging.String("party", tx.Party()))
		}
		return h, errors.New("unknown block height for tx:" + txHash + ", command:" + tx.Command().String() + ", party:" + tx.Party())
	}

	// check if the tid was seen in scope
	if _, ok := e.seenTid[tx.GetPoWTID()]; ok {
		if e.log.IsDebug() {
			e.log.Debug("tid already used", logging.String("tid", tx.GetPoWTID()), logging.String("party", tx.Party()))
		}
		return h, errors.New("proof of work tid already used")
	}

	// verify the proof of work
	hash := e.blockHash[idx]
	success, diff := crypto.Verify(hash, tx.GetPoWTID(), tx.GetPoWNonce(), params.spamPoWHashFunction, params.spamPoWDifficulty)
	if !success {
		if e.log.IsDebug() {
			e.log.Debug("failed to verify proof of work", logging.String("tid", tx.GetPoWTID()), logging.String("party", tx.Party()))
		}
		return diff, errors.New("failed to verify proof of work")
	}
	if e.log.IsDebug() {
		e.log.Debug("transaction passed verify", logging.String("tid", tx.GetPoWTID()), logging.String("party", tx.Party()))
	}
	return diff, nil
}

// verify the proof of work
// 1. check that the block height is already known to the engine - this is rejected if it's too old or not yet seen as we need to know the block hash
// 2. check that we've not seen this transaction ID before (in the previous `spamPoWNumberOfPastBlocks` blocks)
// 3. check that the proof of work can be verified with the required difficulty.
func (e *Engine) verify(tx abci.Tx) (byte, error) {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.verifyWithLock(tx)
}

func (e *Engine) updateParam(netParamName, netParamValue string, p *params) {
	switch netParamName {
	case "spamPoWNumberOfPastBlocks":
		spamPoWNumberOfPastBlock, _ := num.UintFromString(netParamValue, 10)
		p.spamPoWNumberOfPastBlocks = spamPoWNumberOfPastBlock.Uint64()
	case "spamPoWDifficulty":
		spamPoWDifficulty, _ := num.UintFromString(netParamValue, 10)
		p.spamPoWDifficulty = uint(spamPoWDifficulty.Uint64())
	case "spamPoWHashFunction":
		p.spamPoWHashFunction = netParamValue
	case "spamPoWNumberOfTxPerBlock":
		spamPoWNumberOfTxPerBlock, _ := num.UintFromString(netParamValue, 10)
		p.spamPoWNumberOfTxPerBlock = spamPoWNumberOfTxPerBlock.Uint64()
	case "spamPoWIncreasingDifficulty":
		p.spamPoWIncreasingDifficulty = netParamValue != "0"
	}
}

func (e *Engine) updateWithLock(netParamName, netParamValue string) {
	// if there are no settings yet
	if len(e.activeParams) == 0 {
		p := &params{
			fromBlock:  e.currentBlock,
			untilBlock: nil,
		}
		e.activeParams = append(e.activeParams, p)
		newState := &state{
			blockToPartyState: map[uint64]map[string]*partyStateForBlock{},
		}
		e.activeStates = append(e.activeStates, newState)
		e.updateParam(netParamName, netParamValue, p)
		return
	}
	lastActive := e.activeParams[len(e.activeParams)-1]
	if lastActive.fromBlock == e.currentBlock+1 || (len(e.activeParams) == 1 && e.activeParams[0].fromBlock == e.currentBlock) {
		e.updateParam(netParamName, netParamValue, lastActive)
		return
	}
	lastActive.untilBlock = new(uint64)
	*lastActive.untilBlock = e.currentBlock
	newParams := &params{
		fromBlock:                   e.currentBlock + 1,
		untilBlock:                  nil,
		spamPoWIncreasingDifficulty: lastActive.spamPoWIncreasingDifficulty,
		spamPoWNumberOfPastBlocks:   lastActive.spamPoWNumberOfPastBlocks,
		spamPoWDifficulty:           lastActive.spamPoWDifficulty,
		spamPoWHashFunction:         lastActive.spamPoWHashFunction,
		spamPoWNumberOfTxPerBlock:   lastActive.spamPoWNumberOfTxPerBlock,
	}
	e.updateParam(netParamName, netParamValue, newParams)
	e.activeParams = append(e.activeParams, newParams)

	newState := &state{
		blockToPartyState: map[uint64]map[string]*partyStateForBlock{},
	}
	e.activeStates = append(e.activeStates, newState)
}

// UpdateSpamPoWNumberOfPastBlocks updates the network parameter of number of past blocks to look at. This requires extending or shrinking the size of the cache.
func (e *Engine) UpdateSpamPoWNumberOfPastBlocks(_ context.Context, spamPoWNumberOfPastBlocks *num.Uint) error {
	e.log.Info("updating spamPoWNumberOfPastBlocks", logging.Uint64("new-value", spamPoWNumberOfPastBlocks.Uint64()))
	e.lock.Lock()
	defer e.lock.Unlock()
	e.updateWithLock("spamPoWNumberOfPastBlocks", spamPoWNumberOfPastBlocks.String())
	return nil
}

// UpdateSpamPoWDifficulty updates the network parameter for difficulty.
func (e *Engine) UpdateSpamPoWDifficulty(_ context.Context, spamPoWDifficulty *num.Uint) error {
	e.log.Info("updating spamPoWDifficulty", logging.Uint64("new-value", spamPoWDifficulty.Uint64()))
	e.lock.Lock()
	defer e.lock.Unlock()
	e.updateWithLock("spamPoWDifficulty", spamPoWDifficulty.String())
	return nil
}

// UpdateSpamPoWHashFunction updates the network parameter for hash function.
func (e *Engine) UpdateSpamPoWHashFunction(_ context.Context, spamPoWHashFunction string) error {
	e.log.Info("updating spamPoWHashFunction", logging.String("new-value", spamPoWHashFunction))
	e.lock.Lock()
	defer e.lock.Unlock()
	e.updateWithLock("spamPoWHashFunction", spamPoWHashFunction)
	return nil
}

// UpdateSpamPoWNumberOfTxPerBlock updates the number of transactions allowed for a party per block before increased difficulty kicks in if enabled.
func (e *Engine) UpdateSpamPoWNumberOfTxPerBlock(_ context.Context, spamPoWNumberOfTxPerBlock *num.Uint) error {
	e.log.Info("updating spamPoWNumberOfTxPerBlock", logging.Uint64("new-value", spamPoWNumberOfTxPerBlock.Uint64()))
	e.lock.Lock()
	defer e.lock.Unlock()
	e.updateWithLock("spamPoWNumberOfTxPerBlock", spamPoWNumberOfTxPerBlock.String())
	return nil
}

// UpdateSpamPoWIncreasingDifficulty enables/disabled increased difficulty.
func (e *Engine) UpdateSpamPoWIncreasingDifficulty(_ context.Context, spamPoWIncreasingDifficulty *num.Uint) error {
	e.log.Info("updating spamPoWIncreasingDifficulty", logging.Bool("new-value", !spamPoWIncreasingDifficulty.IsZero()))
	e.lock.Lock()
	defer e.lock.Unlock()
	e.updateWithLock("spamPoWIncreasingDifficulty", spamPoWIncreasingDifficulty.String())
	return nil
}

func (e *Engine) getActiveParams() *params {
	if len(e.activeParams) == 1 {
		return e.activeParams[0]
	}
	if e.activeParams[len(e.activeParams)-1].fromBlock > e.currentBlock {
		return e.activeParams[len(e.activeParams)-2]
	}
	return e.activeParams[len(e.activeParams)-1]
}

func (e *Engine) IsReady() bool {
	return len(e.activeParams) > 0
}

func (e *Engine) SpamPoWNumberOfPastBlocks() uint32 {
	return uint32(e.getActiveParams().spamPoWNumberOfPastBlocks)
}

func (e *Engine) SpamPoWDifficulty() uint32 {
	return uint32(e.getActiveParams().spamPoWDifficulty)
}

func (e *Engine) SpamPoWHashFunction() string {
	return e.getActiveParams().spamPoWHashFunction
}

func (e *Engine) SpamPoWNumberOfTxPerBlock() uint32 {
	return uint32(e.getActiveParams().spamPoWNumberOfTxPerBlock)
}

func (e *Engine) SpamPoWIncreasingDifficulty() bool {
	return e.getActiveParams().spamPoWIncreasingDifficulty
}

func (e *Engine) BlockData() (uint64, string) {
	e.lock.RLock()
	defer e.lock.RUnlock()

	if len(e.activeParams) == 0 {
		return 0, ""
	}
	return e.currentBlock, e.blockHash[e.currentBlock%ringSize]
}

func getParamsForBlock(block uint64, activeParams []*params) *params {
	stateInd := 0
	for i, p := range activeParams {
		if block >= p.fromBlock && (p.untilBlock == nil || *p.untilBlock >= block) {
			stateInd = i
			break
		}
	}

	params := activeParams[stateInd]
	return params
}

func (e *Engine) GetSpamStatistics(partyID string) *protoapi.PoWStatistic {
	e.lock.RLock()
	defer e.lock.RUnlock()

	stats := make([]*protoapi.PoWBlockState, 0)

	currentBlockStatsExists := false

	for _, state := range e.activeStates {
		for block, blockToPartyState := range state.blockToPartyState {
			if block == e.currentBlock {
				currentBlockStatsExists = true
			}

			if partyState, ok := blockToPartyState[partyID]; ok {
				blockIndex := block % ringSize
				params := getParamsForBlock(block, e.activeParams)

				stats = append(stats, &protoapi.PoWBlockState{
					BlockHeight:      block,
					BlockHash:        e.blockHash[blockIndex],
					TransactionsSeen: uint64(partyState.seenCount),
					ExpectedDifficulty: getMinDifficultyForNextTx(params.spamPoWDifficulty,
						uint(params.spamPoWNumberOfTxPerBlock),
						partyState.seenCount,
						partyState.observedDifficulty,
						params.spamPoWIncreasingDifficulty,
					),
					IncreasingDifficulty: params.spamPoWIncreasingDifficulty,
					TxPerBlock:           params.spamPoWNumberOfTxPerBlock,
					HashFunction:         params.spamPoWHashFunction,
					Difficulty:           uint64(params.spamPoWDifficulty),
				})
			}
		}
	}

	// If we don't have any spam stats for the current block, add it
	if !currentBlockStatsExists {
		params := getParamsForBlock(e.currentBlock, e.activeParams)
		expected := uint64(params.spamPoWDifficulty)
		stats = append(stats, &protoapi.PoWBlockState{
			BlockHeight:          e.currentBlock,
			BlockHash:            e.blockHash[e.currentBlock%ringSize],
			TransactionsSeen:     0,
			ExpectedDifficulty:   &expected,
			HashFunction:         params.spamPoWHashFunction,
			IncreasingDifficulty: params.spamPoWIncreasingDifficulty,
			TxPerBlock:           params.spamPoWNumberOfTxPerBlock,
			Difficulty:           uint64(params.spamPoWDifficulty),
		})
	}

	return &protoapi.PoWStatistic{
		BlockStates:        stats,
		NumberOfPastBlocks: e.getActiveParams().spamPoWNumberOfPastBlocks,
	}
}

func getMinDifficultyForNextTx(baseDifficulty, txPerBlock, seenTx, observedDifficulty uint, increaseDifficulty bool) *uint64 {
	if !increaseDifficulty {
		if seenTx < txPerBlock {
			return ptr.From(uint64(baseDifficulty))
		}
		// they cannot submit any more against this block, do not return a next-difficulty
		return nil
	}

	// calculate the total expected difficulty based on the number of transactions seen
	totalDifficulty, powDiff := calculateExpectedDifficulty(baseDifficulty, txPerBlock, seenTx)
	// add the current PoW difficulty to the current expected difficulty to get the expected total difficulty for the next transaction
	totalDifficulty += powDiff
	nextExpectedDifficulty := totalDifficulty - observedDifficulty
	if nextExpectedDifficulty < baseDifficulty {
		nextExpectedDifficulty = baseDifficulty
	}

	minDifficultyForNextTx := uint64(nextExpectedDifficulty)

	return &minDifficultyForNextTx
}
