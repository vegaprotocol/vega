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

package protocol

// circling around import cycle here...

import (
	"context"

	"code.vegaprotocol.io/vega/core/client/eth"
	"code.vegaprotocol.io/vega/core/datasource/external/ethcall"
	"code.vegaprotocol.io/vega/core/datasource/external/ethverifier"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/logging"
)

type SpecActivationListener func(listener spec.SpecActivationsListener)

type L2EthCallEngines struct {
	log         *logging.Logger
	cfg         ethcall.Config
	isValidator bool
	clients     *eth.L2Clients
	forwarder   ethcall.Forwarder

	// chain id -> engine
	engines                map[string]*ethcall.Engine
	specActivationListener SpecActivationListener
}

func NewL2EthCallEngines(log *logging.Logger, cfg ethcall.Config, isValidator bool, clients *eth.L2Clients, forwarder ethcall.Forwarder, specActivationListener SpecActivationListener) *L2EthCallEngines {
	return &L2EthCallEngines{
		log:                    log,
		cfg:                    cfg,
		isValidator:            isValidator,
		clients:                clients,
		forwarder:              forwarder,
		engines:                map[string]*ethcall.Engine{},
		specActivationListener: specActivationListener,
	}
}

func (v *L2EthCallEngines) GetOrInstantiate(chainID string) (ethverifier.EthCallEngine, error) {
	if e, ok := v.engines[chainID]; ok {
		return e, nil
	}

	v.log.Panic("should be instantiated by now really?")
	return nil, nil
}

func (v *L2EthCallEngines) OnEthereumL2ConfigsUpdated(
	ctx context.Context, ethCfg *types.EthereumL2Configs,
) error {
	// new L2 configured, instatiate the verifier for it.
	for _, c := range ethCfg.Configs {
		// if already exists, do nothing
		if _, ok := v.engines[c.ChainID]; ok {
			continue
		}

		var clt *eth.L2Client
		if v.isValidator {
			var ok bool
			clt, _, ok = v.clients.Get(c.ChainID)
			if !ok {
				v.log.Panic("ethereum client not configured for L2",
					logging.String("chain-id", c.ChainID),
					logging.String("network-id", c.NetworkID),
				)
			}
		}

		e := ethcall.NewEngine(v.log, v.cfg, v.isValidator, clt, v.forwarder)
		e.EnsureChainID(ctx, c.ChainID, v.isValidator)
		v.engines[c.ChainID] = e

		// if we are restoring from a snapshot we want to delay starting the engine
		// until we know what block height to use. If we aren't restoring from a snapshot
		// we are either loading from genesis, or the engine has been added dynamically and
		// so we want to kick it off
		if !vgcontext.InProgressSnapshotRestore(ctx) {
			e.Start()
		}

		// setup activation listener
		v.specActivationListener(v.engines[c.ChainID])
	}

	return nil
}
