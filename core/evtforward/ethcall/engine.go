package ethcall

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/ethereum/go-ethereum"
)

// Still TODO
//   - listen for new data sources and add specs
//   - on tick check every block since last tick not just current
//   - submit some sort of error event if call fails
//   - know when datasources stop being active and remove them
//     -- because e.g. market is dead, or amended to have different source
//     -- or because trigger will never fire again
//   - what to do about catching up e.g. if node is restarted
type EthReaderCaller interface {
	ethereum.ContractCaller
	ethereum.ChainReader
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/forwarder_mock.go -package mocks code.vegaprotocol.io/vega/core/evtforward/ethcall Forwarder
type Forwarder interface {
	ForwardFromSelf(*commandspb.ChainEvent)
}

type Engine struct {
	log       *logging.Logger
	cfg       Config
	client    EthReaderCaller
	specs     map[string]Spec
	forwarder Forwarder
	prevBlock blockish
}

func NewEngine(log *logging.Logger, cfg Config, client EthReaderCaller, forwarder Forwarder) (*Engine, error) {
	return &Engine{
		log:       log,
		cfg:       cfg,
		client:    client,
		forwarder: forwarder,
		specs:     make(map[string]Spec),
	}, nil
}

func (e *Engine) AddSpec(s Spec) (string, error) {
	id := s.HashHex()
	e.specs[id] = s
	return id, nil
}

func (e *Engine) GetSpec(id string) (Spec, bool) {
	spec, ok := e.specs[id]
	return spec, ok
}

func (e *Engine) OnTick(ctx context.Context, vegaTime time.Time) {
	block, err := e.client.BlockByNumber(ctx, nil)
	if err != nil {
		e.log.Errorf("failed to get current block header: %w", err)
		return
	}

	e.log.Info("tick",
		logging.Time("vegaTime", vegaTime),
		logging.BigInt("ethBlock", block.Number()),
		logging.Time("ethTime", time.Unix(int64(block.Time()), 0)))

	if e.prevBlock == nil {
		e.prevBlock = block
		return
	}

	for specID, spec := range e.specs {
		if spec.Trigger.Trigger(e.prevBlock, block) {
			res, err := spec.Call.Call(ctx, e.client, block.Number())
			if err != nil {
				e.log.Errorf("failed to call contract: %w", err)
				continue
			}
			event := makeChainEvent(res, specID, block)
			e.forwarder.ForwardFromSelf(event)
		}
	}
	e.prevBlock = block
}

func makeChainEvent(res Result, specID string, block blockish) *commandspb.ChainEvent {
	ce := commandspb.ChainEvent{
		TxId:  "", // todo? Are we in a transcation
		Nonce: 0,  // TODO
		Event: &commandspb.ChainEvent_ContractCall{
			ContractCall: &vega.EthContractCallEvent{
				SpecId:      specID,
				BlockHeight: block.NumberU64(),
				BlockTime:   block.Time(),
				Result:      res.Bytes,
			},
		},
	}

	return &ce
}
