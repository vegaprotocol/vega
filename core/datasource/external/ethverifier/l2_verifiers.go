package ethverifier

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/client/eth"
	"code.vegaprotocol.io/vega/core/datasource/external/ethcall"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	"golang.org/x/exp/maps"
)

type L2Clients interface {
	Get(chainID string) (*eth.L2Client, *eth.EthereumConfirmations, bool)
}

type EthL2CallEngines interface {
	GetOrInstantiate(chainID string) (EthCallEngine, error)
}

type L2Verifiers struct {
	log               *logging.Logger
	witness           Witness
	ts                TimeService
	broker            Broker
	oracleBroadcaster OracleDataBroadcaster

	clients L2Clients

	// chain id -> Verifier
	verifiers map[string]*Verifier

	// eth L2 call engines
	ethL2CallEngine EthL2CallEngines
}

func NewL2Verifiers(
	log *logging.Logger,
	witness Witness,
	ts TimeService,
	broker Broker,
	oracleBroadcaster OracleDataBroadcaster,
	clients L2Clients,
) (sv *L2Verifiers) {
	return &L2Verifiers{
		log:               log,
		witness:           witness,
		ts:                ts,
		broker:            broker,
		oracleBroadcaster: oracleBroadcaster,
		clients:           clients,
		verifiers:         map[string]*Verifier{},
	}
}

func (v *L2Verifiers) OnEthereumL2ConfigsUpdated(
	ctx context.Context, ethCfg *types.EthereumL2Configs,
) error {
	// new L2 configured, instatiate the verifier for it.
	for _, c := range ethCfg.Configs {
		// if already exists, do nothing
		if _, ok := v.verifiers[c.ChainID]; ok {
			continue
		}

		clt, confs, ok := v.clients.Get(c.ChainID)
		if !ok {
			v.log.Panic("ethereum client not configured for L2", logging.String("chain-id", c.ChainID))
		}

		_ = clt

		ethCallEngine, err := v.ethL2CallEngine.GetOrInstantiate(c.ChainID)
		if err != nil {
			v.log.Panic("could not get call engine for L2", logging.String("chain-id", c.ChainID))
		}

		verifier := New(v.log, v.witness, v.ts, v.broker, v.oracleBroadcaster, ethCallEngine, confs)

		v.verifiers[c.ChainID] = verifier
	}

	return nil
}

func (v *L2Verifiers) OnTick(ctx context.Context, t time.Time) {
	ids := maps.Keys(v.verifiers)
	sort.Strings(ids)
	for _, id := range ids {
		v.verifiers[id].OnTick(ctx, t)
	}
}

func (v *L2Verifiers) ProcessEthereumContractCallResult(callEvent ethcall.ContractCallEvent) error {
	if callEvent.L2ChainID == nil {
		return errors.New("invalid non l2 event")
	}

	verifier, ok := v.verifiers[fmt.Sprintf("%v", *callEvent.L2ChainID)]
	if !ok {
		return errors.New("unsupported l2 chain")
	}

	return verifier.ProcessEthereumContractCallResult(callEvent)
}

func (v *L2Verifiers) FromProtoSnapshot() {}

func (v *L2Verifiers) ToProtoSnapshot() []byte { return nil }
