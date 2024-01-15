// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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

	isValidator bool

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
	callEngines EthL2CallEngines,
	isValidator bool,
) (sv *L2Verifiers) {
	return &L2Verifiers{
		log:               log,
		witness:           witness,
		ts:                ts,
		broker:            broker,
		oracleBroadcaster: oracleBroadcaster,
		clients:           clients,
		verifiers:         map[string]*Verifier{},
		ethL2CallEngine:   callEngines,
		isValidator:       isValidator,
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
		v.verifiers[c.ChainID] = v.instanciate(c.ChainID)
	}

	return nil
}

func (v *L2Verifiers) instanciate(chainID string) *Verifier {
	var confs *eth.EthereumConfirmations
	if v.isValidator {
		var ok bool
		_, confs, ok = v.clients.Get(chainID)
		if !ok {
			v.log.Panic("ethereum client not configured for L2",
				logging.String("chain-id", chainID),
			)
		}
	}

	ethCallEngine, err := v.ethL2CallEngine.GetOrInstantiate(chainID)
	if err != nil {
		v.log.Panic("could not get call engine for L2", logging.String("chain-id", chainID))
	}

	return New(v.log, v.witness, v.ts, v.broker, v.oracleBroadcaster, ethCallEngine, confs)
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
