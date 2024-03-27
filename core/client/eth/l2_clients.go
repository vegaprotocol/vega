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

package eth

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"

	"github.com/ethereum/go-ethereum/ethclient"
)

type L2Client struct {
	ETHClient
}

type L2Clients struct {
	ctx context.Context
	log *logging.Logger
	// map of chainID -> Client
	clients       map[string]*L2Client
	confirmations map[string]*EthereumConfirmations
}

func NewL2Clients(
	ctx context.Context,
	log *logging.Logger,
	cfg Config,
) (*L2Clients, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	clients := map[string]*L2Client{}
	confirmations := map[string]*EthereumConfirmations{}

	for _, v := range cfg.EVMChainConfigs {
		log.Info("starting L2 client",
			logging.String("chain-id", v.ChainID),
			logging.String("endpoint", v.RPCEndpoint),
		)
		if len(v.ChainID) <= 0 || len(v.RPCEndpoint) <= 0 {
			return nil, errors.New("l2 rpc endpoint configured with empty strings")
		}
		clt, err := DialL2(ctx, v.RPCEndpoint)
		if err != nil {
			return nil, err
		}

		chainID, err := clt.ChainID(ctx)
		if err != nil {
			return nil, fmt.Errorf("couldn't get chain id: %w", err)
		}

		if chainID.String() != v.ChainID {
			return nil, fmt.Errorf("client retrieve different chain id: %v vs %v", chainID.String(), v.ChainID)
		}

		clients[v.ChainID] = clt
		confirmations[v.ChainID] = NewEthereumConfirmations(cfg, clt, nil)
	}

	return &L2Clients{
		ctx:           ctx,
		log:           log,
		clients:       clients,
		confirmations: confirmations,
	}, nil
}

func (e *L2Clients) UpdateConfirmations(ethCfg *types.EthereumL2Configs) {
	for _, v := range ethCfg.Configs {
		confs, ok := e.confirmations[v.ChainID]
		if !ok {
			e.log.Panic("ethereum client for L2 is not configured", logging.String("name", v.Name), logging.String("chain-id", v.ChainID))
		}

		confs.UpdateConfirmations(v.Confirmations)
	}
}

// ReloadConf updates the internal configuration of the execution
// engine and its dependencies.
func (e *L2Clients) ReloadConf(cfg Config) {
	e.log.Debug("reloading configuration")

	if e.log.GetLevel() != cfg.Level.Get() {
		e.log.Info("updating log level",
			logging.String("old", e.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		e.log.SetLevel(cfg.Level.Get())
	}

	e.log.Info("updating L2 clients")
	for _, v := range cfg.EVMChainConfigs {
		if _, ok := e.clients[v.ChainID]; ok {
			e.log.Warn("L2 client already setted up, please stop and restart node to update existing configuration",
				logging.String("chain-id", v.ChainID),
				logging.String("endpoint", v.RPCEndpoint),
			)
			continue
		}

		e.log.Info("starting L2 client",
			logging.String("chain-id", v.ChainID),
			logging.String("endpoint", v.RPCEndpoint),
		)
		if len(v.ChainID) <= 0 || len(v.RPCEndpoint) <= 0 {
			e.log.Warn("invalid L2 client configuration",
				logging.String("chain-id", v.ChainID),
				logging.String("endpoint", v.RPCEndpoint),
			)
			continue
		}
		clt, err := DialL2(e.ctx, v.RPCEndpoint)
		if err != nil {
			e.log.Warn("couldn't start L2 client",
				logging.String("chain-id", v.ChainID),
				logging.String("endpoint", v.RPCEndpoint),
			)
			continue
		}

		chainID, err := clt.ChainID(e.ctx)
		if err != nil {
			e.log.Warn("couldn't get chain id", logging.Error(err))
			continue
		}

		if chainID.String() != v.ChainID {
			e.log.Warn("client retrieved different chain id",
				logging.String("chain-id", chainID.String()),
				logging.String("expected", v.ChainID),
			)
			continue
		}

		e.clients[v.ChainID] = clt
		e.confirmations[v.ChainID] = NewEthereumConfirmations(cfg, clt, nil)
	}
}

func DialL2(ctx context.Context, endpoint string) (*L2Client, error) {
	ethClient, err := ethclient.DialContext(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("couldn't instantiate Ethereum client: %w", err)
	}

	return &L2Client{ETHClient: newEthClientWrapper(ethClient)}, nil
}

func (c *L2Clients) Get(chainID string) (*L2Client, *EthereumConfirmations, bool) {
	clt, ok1 := c.clients[chainID]
	confs, ok2 := c.confirmations[chainID]
	return clt, confs, ok1 && ok2
}
