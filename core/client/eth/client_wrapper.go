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
	"math/big"
	"regexp"
	"time"

	"code.vegaprotocol.io/vega/core/metrics"

	"github.com/ethereum/go-ethereum"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/hashicorp/golang-lru/v2/expirable"
)

// ETHClient ...
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/eth_client_mock.go -package mocks code.vegaprotocol.io/vega/core/client/eth ETHClient
type ETHClient interface { //revive:disable:exported
	// bind.ContractBackend
	// ethereum.ChainReader

	// client
	ChainID(context.Context) (*big.Int, error)
	NetworkID(context.Context) (*big.Int, error)

	// ethereum.ChainReader
	BlockByHash(ctx context.Context, hash ethcommon.Hash) (*ethtypes.Block, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*ethtypes.Header, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*ethtypes.Block, error)
	HeaderByHash(ctx context.Context, hash ethcommon.Hash) (*ethtypes.Header, error)
	SubscribeNewHead(ctx context.Context, ch chan<- *ethtypes.Header) (ethereum.Subscription, error)
	TransactionCount(ctx context.Context, blockHash ethcommon.Hash) (uint, error)
	TransactionInBlock(ctx context.Context, blockHash ethcommon.Hash, index uint) (*ethtypes.Transaction, error)

	// bind.ContractCaller
	CodeAt(ctx context.Context, contract ethcommon.Address, blockNumber *big.Int) ([]byte, error)
	CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)

	// bind.ContractTransactor
	EstimateGas(ctx context.Context, call ethereum.CallMsg) (gas uint64, err error)
	PendingCodeAt(ctx context.Context, account ethcommon.Address) ([]byte, error)
	PendingNonceAt(ctx context.Context, account ethcommon.Address) (uint64, error)
	SendTransaction(ctx context.Context, tx *ethtypes.Transaction) error
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
	SuggestGasTipCap(ctx context.Context) (*big.Int, error)

	// bind.ContractFilterer
	FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]ethtypes.Log, error)
	SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- ethtypes.Log) (ethereum.Subscription, error)
}

type ethClientWrapper struct {
	clt ETHClient

	headerByNumberCache *expirable.LRU[string, *ethtypes.Header]
}

func newEthClientWrapper(clt ETHClient) *ethClientWrapper {
	return &ethClientWrapper{
		clt: clt,
		// arbitrary size of 100 blocks, kept for at most 10 minutes,
		// let see later how to make this less hardcoded
		headerByNumberCache: expirable.NewLRU[string, *ethtypes.Header](100, nil, 10*time.Minute),
	}
}

func (c *ethClientWrapper) ChainID(ctx context.Context) (*big.Int, error) {
	metrics.EthereumRPCCallCounterInc("chain_id")
	id, err := c.clt.ChainID(ctx)
	if err != nil {
		return nil, ErrorWithStrippedSecrets{err: err}
	}
	return id, nil
}

func (c *ethClientWrapper) NetworkID(ctx context.Context) (*big.Int, error) {
	metrics.EthereumRPCCallCounterInc("network_id")
	id, err := c.clt.NetworkID(ctx)
	if err != nil {
		return nil, ErrorWithStrippedSecrets{err: err}
	}
	return id, nil
}

func (c *ethClientWrapper) BlockByHash(ctx context.Context, hash ethcommon.Hash) (*ethtypes.Block, error) {
	metrics.EthereumRPCCallCounterInc("block_by_hash")
	byHash, err := c.clt.BlockByHash(ctx, hash)
	if err != nil {
		return nil, ErrorWithStrippedSecrets{err: err}
	}
	return byHash, nil
}

func (c *ethClientWrapper) HeaderByNumber(ctx context.Context, number *big.Int) (*ethtypes.Header, error) {
	if number != nil {
		// first check the cache
		if header, ok := c.headerByNumberCache.Get(number.String()); ok {
			return ethtypes.CopyHeader(header), nil
		}
	}

	// cache miss, so let's inc the counter, and call the rpc.
	metrics.EthereumRPCCallCounterInc("header_by_number")
	header, err := c.clt.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, ErrorWithStrippedSecrets{err: err}
	}

	c.headerByNumberCache.Add(header.Number.String(), ethtypes.CopyHeader(header))

	return header, nil
}

func (c *ethClientWrapper) BlockByNumber(ctx context.Context, number *big.Int) (*ethtypes.Block, error) {
	metrics.EthereumRPCCallCounterInc("block_by_number")
	byNumber, err := c.clt.BlockByNumber(ctx, number)
	if err != nil {
		return nil, ErrorWithStrippedSecrets{err: err}
	}
	return byNumber, nil
}

func (c *ethClientWrapper) HeaderByHash(ctx context.Context, hash ethcommon.Hash) (*ethtypes.Header, error) {
	metrics.EthereumRPCCallCounterInc("header_by_hash")
	byHash, err := c.clt.HeaderByHash(ctx, hash)
	if err != nil {
		return nil, ErrorWithStrippedSecrets{err: err}
	}
	return byHash, nil
}

func (c *ethClientWrapper) SubscribeNewHead(ctx context.Context, ch chan<- *ethtypes.Header) (ethereum.Subscription, error) {
	head, err := c.clt.SubscribeNewHead(ctx, ch)
	if err != nil {
		return nil, ErrorWithStrippedSecrets{err: err}
	}
	return head, nil
}

func (c *ethClientWrapper) TransactionCount(ctx context.Context, blockHash ethcommon.Hash) (uint, error) {
	metrics.EthereumRPCCallCounterInc("transaction_count")
	count, err := c.clt.TransactionCount(ctx, blockHash)
	if err != nil {
		return 0, ErrorWithStrippedSecrets{err: err}
	}
	return count, nil
}

func (c *ethClientWrapper) TransactionInBlock(ctx context.Context, blockHash ethcommon.Hash, index uint) (*ethtypes.Transaction, error) {
	metrics.EthereumRPCCallCounterInc("transaction_in_block")
	block, err := c.clt.TransactionInBlock(ctx, blockHash, index)
	if err != nil {
		return nil, ErrorWithStrippedSecrets{err: err}
	}
	return block, nil
}

func (c *ethClientWrapper) CodeAt(ctx context.Context, contract ethcommon.Address, blockNumber *big.Int) ([]byte, error) {
	metrics.EthereumRPCCallCounterInc("code_at")
	at, err := c.clt.CodeAt(ctx, contract, blockNumber)
	if err != nil {
		return nil, ErrorWithStrippedSecrets{err: err}
	}
	return at, nil
}

func (c *ethClientWrapper) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	metrics.EthereumRPCCallCounterInc("call_contract")
	contract, err := c.clt.CallContract(ctx, call, blockNumber)
	if err != nil {
		return nil, ErrorWithStrippedSecrets{err: err}
	}
	return contract, nil
}

func (c *ethClientWrapper) EstimateGas(ctx context.Context, call ethereum.CallMsg) (gas uint64, err error) {
	metrics.EthereumRPCCallCounterInc("estimate_gas")
	estimateGas, err := c.clt.EstimateGas(ctx, call)
	if err != nil {
		return 0, ErrorWithStrippedSecrets{err: err}
	}
	return estimateGas, nil
}

func (c *ethClientWrapper) PendingCodeAt(ctx context.Context, account ethcommon.Address) ([]byte, error) {
	metrics.EthereumRPCCallCounterInc("pending_code_at")
	at, err := c.clt.PendingCodeAt(ctx, account)
	if err != nil {
		return nil, ErrorWithStrippedSecrets{err: err}
	}
	return at, nil
}

func (c *ethClientWrapper) PendingNonceAt(ctx context.Context, account ethcommon.Address) (uint64, error) {
	metrics.EthereumRPCCallCounterInc("pending_nonce_at")
	at, err := c.clt.PendingNonceAt(ctx, account)
	if err != nil {
		return 0, ErrorWithStrippedSecrets{err: err}
	}
	return at, nil
}

func (c *ethClientWrapper) SendTransaction(ctx context.Context, tx *ethtypes.Transaction) error {
	metrics.EthereumRPCCallCounterInc("send_transaction")
	err := c.clt.SendTransaction(ctx, tx)
	return ErrorWithStrippedSecrets{err: err}
}

func (c *ethClientWrapper) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	metrics.EthereumRPCCallCounterInc("suggest_gas_price")
	price, err := c.clt.SuggestGasPrice(ctx)
	if err != nil {
		return nil, ErrorWithStrippedSecrets{err: err}
	}
	return price, nil
}

func (c *ethClientWrapper) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	metrics.EthereumRPCCallCounterInc("suggest_gas_tip_cap")
	tipCap, err := c.clt.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, ErrorWithStrippedSecrets{err: err}
	}
	return tipCap, nil
}

func (c *ethClientWrapper) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]ethtypes.Log, error) {
	metrics.EthereumRPCCallCounterInc("filter_logs")
	logs, err := c.clt.FilterLogs(ctx, query)
	if err != nil {
		return nil, ErrorWithStrippedSecrets{err: err}
	}
	return logs, nil
}

func (c *ethClientWrapper) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- ethtypes.Log) (ethereum.Subscription, error) {
	metrics.EthereumRPCCallCounterInc("subscribe_filter_logs")
	logs, err := c.clt.SubscribeFilterLogs(ctx, query, ch)
	if err != nil {
		return nil, ErrorWithStrippedSecrets{err: err}
	}
	return logs, nil
}

// ErrorWithStrippedSecrets is an extremely naïve implementation of an error that
// will strip the API token from a URL.
type ErrorWithStrippedSecrets struct {
	err error
}

func (e ErrorWithStrippedSecrets) Error() string {
	return regexp.
		MustCompile(`(?i)(apitoken|token|apikey|key)=(.+)"`).
		ReplaceAllString(e.err.Error(), "$1=xxx\"")
}
