package ethcall

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/types"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/ethereum/go-ethereum"
)

// Still TODO
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
	log         *logging.Logger
	cfg         Config
	client      EthReaderCaller
	dataSources map[string]*DataSource
	forwarder   Forwarder
	prevBlock   types.Blockish
}

func NewEngine(log *logging.Logger, cfg Config, client EthReaderCaller, forwarder Forwarder) (*Engine, error) {
	return &Engine{
		log:         log,
		cfg:         cfg,
		client:      client,
		forwarder:   forwarder,
		dataSources: make(map[string]*DataSource),
	}, nil
}

func (e *Engine) GetDataSource(id string) (*DataSource, bool) {
	if source, ok := e.dataSources[id]; ok {
		return source, true
	}

	return nil, false
}

func (e *Engine) OnSpecActivated(ctx context.Context, spec types.OracleSpec) error {
	switch d := spec.ExternalDataSourceSpec.Spec.Data.SourceType.(type) {
	case *types.EthCallSpec:
		if e.dataSources[d.HashHex()] != nil {
			return fmt.Errorf("duplicate spec: %s", d.HashHex())
		}

		dataSource, err := NewDataSource(d)
		if err != nil {
			return fmt.Errorf("failed to create data source: %w", err)
		}

		e.dataSources[d.HashHex()] = dataSource
	}

	return nil
}

func (e *Engine) OnSpecDeactivated(ctx context.Context, spec types.OracleSpec) {
	switch d := spec.ExternalDataSourceSpec.Spec.Data.SourceType.(type) {
	case *types.EthCallSpec:
		delete(e.dataSources, d.HashHex())
	}
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

	for specID, datasource := range e.dataSources {
		if datasource.Trigger(e.prevBlock, block) {
			res, err := datasource.Call.Call(ctx, e.client, block.Number())
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

func makeChainEvent(res Result, specID string, block types.Blockish) *commandspb.ChainEvent {
	ce := commandspb.ChainEvent{
		TxId:  "", // NA
		Nonce: 0,  // NA
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
