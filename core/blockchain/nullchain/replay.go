package nullchain

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/core/blockchain"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/proto/tendermint/types"
)

var ErrReplayFileIsRequired = errors.New("replay-file is required when replay/record is enabled")

type blockData struct {
	Height  int64    `json:"height"`
	Time    int64    `json:"time"`
	Txs     [][]byte `json:"txns"`
	AppHash []byte   `json:"appHash"`
}

type Replayer struct {
	log     *logging.Logger
	app     ApplicationService
	rFile   *os.File
	current *blockData
	stop    chan struct{}
}

func NewNullChainReplayer(app ApplicationService, cfg blockchain.ReplayConfig, log *logging.Logger) (*Replayer, error) {
	if cfg.ReplayFile == "" {
		return nil, ErrReplayFileIsRequired
	}

	flags := os.O_RDWR | os.O_CREATE
	if !cfg.Replay {
		// not replaying so make sure the file is empty before we start recording
		flags |= os.O_TRUNC
	}
	f, err := os.OpenFile(cfg.ReplayFile, flags, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to open replay file %s: %w", cfg.ReplayFile, err)
	}

	return &Replayer{
		app:   app,
		rFile: f,
		log:   log,
		stop:  make(chan struct{}, 1),
	}, nil
}

func (r *Replayer) InitChain(req abci.RequestInitChain) (resp abci.ResponseInitChain) {
	return r.app.InitChain(req)
}

func (r *Replayer) Stop() error {
	r.stop <- struct{}{}
	close(r.stop)
	return r.rFile.Close()
}

// startBlock saves in memory all the transactions in the block, we do not write until saveBlock us called
// with a potential appHash.
func (r *Replayer) startBlock(height, now int64, txs []*abci.RequestDeliverTx) {
	r.current = &blockData{
		Height: height,
		Time:   now,
	}
	for _, tx := range txs {
		r.current.Txs = append(r.current.Txs, tx.Tx)
	}
}

// saveBlock writes to the replay file the details of the current block adding the appHash to it.
// If a panic occurred appHash may be empty.
func (r *Replayer) saveBlock(appHash []byte) {
	r.current.AppHash = appHash
	if err := r.write(); err != nil {
		r.log.Panic("unable to write block to file", logging.Int64("block-height", r.current.Height))
	}
	r.current = nil
}

// replayChain sends all the recorded per-block transactions into the protocol returning the block-height and block-time it reached
// appHeight is the block-height the application will process next, any blocks less than this will not be replayed.
func (r *Replayer) replayChain(appHeight int64, chainID string) (int64, time.Time, error) {
	// open the replay file and read line by line
	s := bufio.NewScanner(r.rFile)
	s.Split(bufio.ScanLines)

	var replayedHeight int64
	var replayedTime time.Time
	for s.Scan() {
		select {
		case <-r.stop:
			r.log.Info("core is shutting down, nullchain replaying stopped", logging.Int64("block-height", replayedHeight))
			return replayedHeight, replayedTime, nil
		default:
		}

		var data blockData
		if err := json.Unmarshal(s.Bytes(), &data); err != nil {
			return replayedHeight, replayedTime, err
		}

		replayedHeight = data.Height
		replayedTime = time.Unix(0, data.Time)

		if data.Height < appHeight {
			// skip because we've loaded from a snapshot at a block higher than this
			continue
		}

		r.log.Info("replaying block", logging.Int64("height", data.Height), logging.Int("ntxns", len(data.Txs)))
		r.app.BeginBlock(
			abci.RequestBeginBlock{
				Header: types.Header{
					Time:    time.Unix(0, data.Time),
					Height:  data.Height,
					ChainID: chainID,
				},
				Hash: vgcrypto.Hash([]byte(strconv.FormatInt(data.Height+data.Time, 10))),
			},
		)

		// deliever all the txns in that block
		for _, tx := range data.Txs {
			r.app.DeliverTx(abci.RequestDeliverTx{Tx: tx})
		}

		r.app.EndBlock(
			abci.RequestEndBlock{
				Height: data.Height,
			},
		)
		resp := r.app.Commit()

		if len(data.AppHash) == 0 {
			// we've replayed a block which when recorded must have panicked so we do not have a apphash
			// somehow we've made it through this time, maybe someone is testing a fix so we skip the hash check and log it as strange
			r.log.Error("app-hash missing from block data -- a block with a panic is working now?")
			continue
		}

		if !bytes.Equal(data.AppHash, resp.Data) {
			return replayedHeight, replayedTime, fmt.Errorf("appHash mismatch on replay, expected %s got %s", hex.EncodeToString(data.AppHash), hex.EncodeToString(resp.Data))
		}
	}

	if replayedHeight < appHeight-1 {
		return replayedHeight, replayedTime, fmt.Errorf("replay data missing, replay store up to height %d, but app-height is %d", replayedHeight, appHeight)
	}

	return replayedHeight, replayedTime, nil
}

func (r *Replayer) write() error {
	b, err := json.Marshal(r.current)
	if err != nil {
		return fmt.Errorf("unable to record block %d: %w", r.current.Height, err)
	}

	// write each marshalled json block on a new line, its crude, but lets worry about perf if perf becomes a problem.
	r.rFile.Write(b)
	r.rFile.Write([]byte("\n"))
	return nil
}
