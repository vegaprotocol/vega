package pow

import (
	"context"
	"errors"
	"math"
	"math/big"
	"sort"
	"sync"

	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/shared/libs/crypto"
	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

const banPeriod = 4

type EpochEngine interface {
	NotifyOnEpoch(f func(context.Context, types.Epoch), r func(context.Context, types.Epoch))
}

type Engine struct {
	blockHeight []uint64            // block heights in scope ring buffer
	blockHash   []string            // block hashes in scope ring buffer
	seenTx      map[string]struct{} // seen transactions in scope set
	heightToTx  map[uint64][]string // height to slice of seen transaction in scope ring buffer

	seenTid     map[string]struct{} // seen tid in scope set
	heightToTid map[uint64][]string // height to slice of seen tid in scope ring buffer

	bannedParties   map[string]uint64    // banned party to last epoch of ban
	blockPartyToPoW map[string][]big.Int // proof of work for party transactions for the current block

	currentBlock uint64 // the current block height
	currentEpoch uint64 // the current epoch sequence

	// spam proof of work configuration
	spamPoWNumberOfPastBlocks   uint64
	spamPoWDifficulty           uint
	spamPoWHashFunction         string
	spamPoWNumberOfTxPerBlock   uint64
	spamPoWIncreasingDifficulty bool

	// difficulty masks for quicker verification for banning at the end of block.
	difficultyMasks [256]big.Int

	// snapshot key
	hashKeys []string
	log      *logging.Logger
	lock     sync.RWMutex
}

// New instantiates the proof of work engine.
func New(log *logging.Logger, config Config, epochEngine EpochEngine) *Engine {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())
	e := &Engine{
		seenTx:          map[string]struct{}{},
		seenTid:         map[string]struct{}{},
		bannedParties:   map[string]uint64{},
		blockPartyToPoW: map[string][]big.Int{},
		heightToTx:      map[uint64][]string{},
		heightToTid:     map[uint64][]string{},
		log:             log,
		hashKeys:        []string{(&types.PayloadProofOfWork{}).Key()},
	}
	epochEngine.NotifyOnEpoch(e.OnEpochEvent, e.OnEpochRestore)

	for i := uint(1); i < 257; i++ {
		target := big.NewInt(1)
		e.difficultyMasks[i-1] = *target.Lsh(target, 256-i)
	}

	e.log.Info("PoW spam protection started")
	return e
}

// OnEpochRestore is called when we restore the epoch from snapshot.
func (e *Engine) OnEpochRestore(ctx context.Context, epoch types.Epoch) {
	e.currentEpoch = epoch.Seq
}

// OnEpochEvent is called on epoch events. It only cares about new epoch events.
func (e *Engine) OnEpochEvent(ctx context.Context, epoch types.Epoch) {
	if epoch.Action != vega.EpochAction_EPOCH_ACTION_START {
		return
	}
	e.currentEpoch = epoch.Seq

	e.lock.Lock()
	defer e.lock.Unlock()

	// check if there are banned parties who can be released
	for k, v := range e.bannedParties {
		if epoch.Seq > v {
			delete(e.bannedParties, k)
			e.log.Info("released proof of work spam ban from", logging.String("party", k))
		}
	}
}

// OnBeginBlock updates the block height and block hash and clears any out of scope block height transactions.
func (e *Engine) BeginBlock(blockHeight uint64, blockHash string) {
	e.lock.Lock()
	defer e.lock.Unlock()

	// save the block height and hash for the new block
	idx := blockHeight % e.spamPoWNumberOfPastBlocks
	e.blockHeight[idx] = blockHeight
	e.blockHash[idx] = blockHash

	// if need to clear stale blocks, delete seen transactions from stale block heights
	if blockHeight > e.spamPoWNumberOfPastBlocks {
		for _, v := range e.heightToTx[blockHeight-e.spamPoWNumberOfPastBlocks] {
			delete(e.seenTx, v)
		}
		for _, v := range e.heightToTid[blockHeight-e.spamPoWNumberOfPastBlocks] {
			delete(e.seenTid, v)
		}
		delete(e.heightToTx, blockHeight-e.spamPoWNumberOfPastBlocks)
		delete(e.heightToTid, blockHeight-e.spamPoWNumberOfPastBlocks)
	}
	e.currentBlock = blockHeight
}

// EndOfBlock processes transactions at the end of the block to check for violations of the number of transactions allowed per block and ban offenders.
func (e *Engine) EndOfBlock() {
	e.lock.Lock()
	defer e.lock.Unlock()

	// iterate over the parties and their transactions in the block and verify that they didn't abuse the block
	for k, v := range e.blockPartyToPoW {
		// if the number of transactions for the party is less or equal than what's allowed, no violation
		if uint64(len(v)) <= e.spamPoWNumberOfTxPerBlock {
			continue
		}

		// if the number of transaction is more than what's allowed and increasing difficulty is off or the number of transaction in the block
		// is greater than the maximum number of transactions - then the party should be banned.
		if !e.spamPoWIncreasingDifficulty || uint64(len(v))-e.spamPoWNumberOfTxPerBlock > uint64(256-e.spamPoWDifficulty) {
			e.log.Info("banning party for sending too many transactions in block", logging.String("party", k))
			e.bannedParties[k] = e.currentEpoch + banPeriod
			continue
		}

		// sort the proof of work from the smallest to the largest
		sort.SliceStable(v, func(i, j int) bool {
			return v[i].Cmp(&v[j]) < 0
		})

		// we need to check that all transactions beyond the `spamPoWNumberOfTxPerBlock` have increasing difficulty
		numberToCompare := uint64(len(v)) - e.spamPoWNumberOfTxPerBlock
		incDiff := v[:numberToCompare]

		// we're looking at the difficulty from the hardest to the easiet and checking that there is a pow that satisfied this difficulty
		// by comparing against the corresponding hash.
		// NB there is always a chance that they got lucky and when asking for difficulty of 20 they incidentally got 22 and by that they can get away
		// without actually increasing the difficulty of the calculation
		for ind, pow := range incDiff {
			maskIndex := uint64(e.spamPoWDifficulty) + uint64(len(incDiff)) - uint64(ind)
			mask := e.difficultyMasks[maskIndex-1]
			// if they don't satisfy the required difficulty for last level - i then ban them
			if pow.Cmp(&mask) != -1 {
				e.log.Info("banning party for sending too many transactions in block with insufficient difficulty", logging.String("party", k))
				e.bannedParties[k] = e.currentEpoch + banPeriod
				break
			}
		}
	}
	// clear the pow map in preparation for the next block.
	e.blockPartyToPoW = map[string][]big.Int{}
}

// CheckTx is called by checkTx in the abci and verifies the proof of work, it doesn't update any state.
func (e *Engine) CheckTx(tx abci.Tx) error {
	// we don't require proof of work for validator command
	if tx.Command().IsValidatorCommand() {
		return nil
	}

	_, err := e.verify(tx)
	return err
}

// DeliverTx is called by deliverTx in the abci and verifies the proof of work, takes a not of the transaction id and counts the number of transactions of the party in the block.
func (e *Engine) DeliverTx(tx abci.Tx) error {
	// we don't require proof of work for validator command
	if tx.Command().IsValidatorCommand() {
		return nil
	}

	h, err := e.verify(tx)
	if err != nil {
		return err
	}

	e.lock.Lock()
	defer e.lock.Unlock()

	// keep the transaction ID
	e.seenTx[string(tx.Hash())] = struct{}{}
	e.heightToTx[tx.BlockHeight()] = append(e.heightToTx[tx.BlockHeight()], string(tx.Hash()))

	// if version supports pow, save the pow result and the tid
	if tx.GetVersion() > 1 {
		e.heightToTid[tx.BlockHeight()] = append(e.heightToTid[tx.BlockHeight()], tx.GetPoWTID())
		e.seenTid[tx.GetPoWTID()] = struct{}{}
		e.blockPartyToPoW[tx.Party()] = append(e.blockPartyToPoW[tx.Party()], h)
	}
	return nil
}

// verify the proof of work
// 1. check that the party is not banned
// 2. check that the block height is already known to the engine - this is rejected if its too old or not yet seen as we need to know the block hash
// 3. check that we've not seen this transaction ID before (in the previous `spamPoWNumberOfPastBlocks` blocks)
// 4. check that the proof of work can be verified with the required difficulty.
func (e *Engine) verify(tx abci.Tx) (big.Int, error) {
	e.lock.RLock()
	defer e.lock.RUnlock()
	var h big.Int

	// check if the party is banned for the epoch
	if _, ok := e.bannedParties[tx.Party()]; ok {
		return h, errors.New("party is banned from sending transactions")
	}

	// check if the block height is in scope and is known
	idx := tx.BlockHeight() % e.spamPoWNumberOfPastBlocks
	if e.blockHeight[idx] != tx.BlockHeight() {
		return h, errors.New("unknown block height")
	}

	// check if the transaction was seen in scope
	if _, ok := e.seenTx[string(tx.Hash())]; ok {
		return h, errors.New("transaction ID already used")
	}

	if tx.GetVersion() < 2 {
		return h, nil
	}

	// check if the tid was seen in scope
	if _, ok := e.seenTid[tx.GetPoWTID()]; ok {
		return h, errors.New("transaction ID already used")
	}

	// verify the proof of work
	hash := e.blockHash[idx]
	success, h := crypto.Verify(hash, tx.GetPoWTID(), tx.GetPoWNonce(), e.spamPoWHashFunction, e.spamPoWDifficulty)
	if !success {
		return h, errors.New("failed to verify proof of work")
	}
	return h, nil
}

// UpdateSpamPoWNumberOfPastBlocks updates the network parameter of number of past blocks to look at. This requires extending or shrinking the size of the cache.
func (e *Engine) UpdateSpamPoWNumberOfPastBlocks(_ context.Context, spamPoWNumberOfPastBlocks *num.Uint) error {
	e.log.Info("updating spamPoWNumberOfPastBlocks", logging.Uint64("old-value", e.spamPoWNumberOfPastBlocks), logging.Uint64("new-value", spamPoWNumberOfPastBlocks.Uint64()))

	e.lock.Lock()
	defer e.lock.Unlock()
	// need to remap recent blocks
	newLen := spamPoWNumberOfPastBlocks.Uint64()
	oldLen := e.spamPoWNumberOfPastBlocks
	blockHeights := make([]uint64, newLen)
	blockHashes := make([]string, newLen)
	if e.spamPoWNumberOfPastBlocks > 0 {
		lenToCopy := uint64(math.Min(float64(newLen), float64(len(e.blockHeight))))

		for i := uint64(0); i < lenToCopy; i++ {
			blockHeights[e.blockHeight[(e.currentBlock-i)%oldLen]%newLen] = e.blockHeight[(e.currentBlock-i)%oldLen]
			blockHashes[e.blockHeight[(e.currentBlock-i)%oldLen]%newLen] = e.blockHash[(e.currentBlock-i)%oldLen]
		}

		// clear transactions if necessary
		if spamPoWNumberOfPastBlocks.Uint64() < e.spamPoWNumberOfPastBlocks {
			for i := e.currentBlock - spamPoWNumberOfPastBlocks.Uint64(); i > e.currentBlock-e.spamPoWNumberOfPastBlocks; i-- {
				for _, v := range e.heightToTx[i] {
					delete(e.seenTx, v)
				}
				delete(e.heightToTx, i)
			}

			for i := e.currentBlock - spamPoWNumberOfPastBlocks.Uint64(); i > e.currentBlock-e.spamPoWNumberOfPastBlocks; i-- {
				for _, v := range e.heightToTid[i] {
					delete(e.seenTid, v)
				}
				delete(e.heightToTid, i)
			}
		}
	}

	e.blockHash = blockHashes
	e.blockHeight = blockHeights
	e.spamPoWNumberOfPastBlocks = spamPoWNumberOfPastBlocks.Uint64()
	return nil
}

// UpdateSpamPoWDifficulty updates the network parameter for difficulty.
func (e *Engine) UpdateSpamPoWDifficulty(_ context.Context, spamPoWDifficulty *num.Uint) error {
	e.log.Info("updating spamPoWDifficulty", logging.Uint("old-value", e.spamPoWDifficulty), logging.Uint64("new-value", spamPoWDifficulty.Uint64()))
	e.lock.Lock()
	defer e.lock.Unlock()
	e.spamPoWDifficulty = uint(spamPoWDifficulty.Uint64())
	return nil
}

// UpdateSpamPoWHashFunction updates the network parameter for hash function.
func (e *Engine) UpdateSpamPoWHashFunction(_ context.Context, spamPoWHashFunction string) error {
	e.log.Info("updating spamPoWHashFunction", logging.String("old-value", e.spamPoWHashFunction), logging.String("new-value", spamPoWHashFunction))
	e.lock.Lock()
	defer e.lock.Unlock()
	e.spamPoWHashFunction = spamPoWHashFunction
	return nil
}

// UpdateSpamPoWNumberOfTxPerBlock updates the number of transactions allowed for a party per block before increased difficulty kicks in if enabled.
func (e *Engine) UpdateSpamPoWNumberOfTxPerBlock(_ context.Context, spamPoWNumberOfTxPerBlock *num.Uint) error {
	e.log.Info("updating spamPoWNumberOfTxPerBlock", logging.Uint64("old-value", e.spamPoWNumberOfTxPerBlock), logging.Uint64("new-value", spamPoWNumberOfTxPerBlock.Uint64()))

	e.lock.Lock()
	defer e.lock.Unlock()
	e.spamPoWNumberOfTxPerBlock = spamPoWNumberOfTxPerBlock.Uint64()
	return nil
}

// UpdateSpamPoWIncreasingDifficulty enables/disabled increased difficulty.
func (e *Engine) UpdateSpamPoWIncreasingDifficulty(_ context.Context, spamPoWIncreasingDifficulty *num.Uint) error {
	e.log.Info("updating spamPoWIncreasingDifficulty", logging.Bool("old-value", e.spamPoWIncreasingDifficulty), logging.Bool("new-value", !spamPoWIncreasingDifficulty.IsZero()))
	e.lock.Lock()
	defer e.lock.Unlock()
	e.spamPoWIncreasingDifficulty = !spamPoWIncreasingDifficulty.IsZero()
	return nil
}

func (e *Engine) SpamPoWNumberOfPastBlocks() uint32 { return uint32(e.spamPoWNumberOfPastBlocks) }
func (e *Engine) SpamPoWDifficulty() uint32         { return uint32(e.spamPoWDifficulty) }
func (e *Engine) SpamPoWHashFunction() string       { return e.spamPoWHashFunction }
func (e *Engine) SpamPoWNumberOfTxPerBlock() uint32 { return uint32(e.spamPoWNumberOfTxPerBlock) }
func (e *Engine) SpamPoWIncreasingDifficulty() bool { return e.spamPoWIncreasingDifficulty }
