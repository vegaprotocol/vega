package eth

import (
	"context"
	"math/big"
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
	return c.clt.ChainID(ctx)
}
func (c *ethClientWrapper) NetworkID(ctx context.Context) (*big.Int, error) {
	metrics.EthereumRPCCallCounterInc("network_id")
	return c.clt.NetworkID(ctx)
}

func (c *ethClientWrapper) BlockByHash(ctx context.Context, hash ethcommon.Hash) (*ethtypes.Block, error) {
	metrics.EthereumRPCCallCounterInc("block_by_hash")
	return c.clt.BlockByHash(ctx, hash)
}

func (c *ethClientWrapper) HeaderByNumber(ctx context.Context, number *big.Int) (*ethtypes.Header, error) {

	// first check the cache
	if header, ok := c.headerByNumberCache.Get(number.String()); ok {
		return ethtypes.CopyHeader(header), nil
	}

	// cache miss, so let's inc the counter, and call the rpc.
	metrics.EthereumRPCCallCounterInc("header_by_number")
	header, err := c.clt.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, err
	}

	c.headerByNumberCache.Add(number.String(), ethtypes.CopyHeader(header))

	return header, nil
}

func (c *ethClientWrapper) BlockByNumber(ctx context.Context, number *big.Int) (*ethtypes.Block, error) {
	metrics.EthereumRPCCallCounterInc("block_by_number")
	return c.clt.BlockByNumber(ctx, number)
}

func (c *ethClientWrapper) HeaderByHash(ctx context.Context, hash ethcommon.Hash) (*ethtypes.Header, error) {
	metrics.EthereumRPCCallCounterInc("header_by_hash")
	return c.clt.HeaderByHash(ctx, hash)
}

func (c *ethClientWrapper) SubscribeNewHead(ctx context.Context, ch chan<- *ethtypes.Header) (ethereum.Subscription, error) {
	return c.clt.SubscribeNewHead(ctx, ch)
}

func (c *ethClientWrapper) TransactionCount(ctx context.Context, blockHash ethcommon.Hash) (uint, error) {
	metrics.EthereumRPCCallCounterInc("transaction_count")
	return c.clt.TransactionCount(ctx, blockHash)
}

func (c *ethClientWrapper) TransactionInBlock(ctx context.Context, blockHash ethcommon.Hash, index uint) (*ethtypes.Transaction, error) {
	metrics.EthereumRPCCallCounterInc("transaction_in_block")
	return c.clt.TransactionInBlock(ctx, blockHash, index)
}

func (c *ethClientWrapper) CodeAt(ctx context.Context, contract ethcommon.Address, blockNumber *big.Int) ([]byte, error) {
	metrics.EthereumRPCCallCounterInc("code_at")
	return c.clt.CodeAt(ctx, contract, blockNumber)
}

func (c *ethClientWrapper) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	metrics.EthereumRPCCallCounterInc("call_contract")
	return c.clt.CallContract(ctx, call, blockNumber)
}

func (c *ethClientWrapper) EstimateGas(ctx context.Context, call ethereum.CallMsg) (gas uint64, err error) {
	metrics.EthereumRPCCallCounterInc("estimate_gas")
	return c.clt.EstimateGas(ctx, call)
}

func (c *ethClientWrapper) PendingCodeAt(ctx context.Context, account ethcommon.Address) ([]byte, error) {
	metrics.EthereumRPCCallCounterInc("pending_code_at")
	return c.clt.PendingCodeAt(ctx, account)
}

func (c *ethClientWrapper) PendingNonceAt(ctx context.Context, account ethcommon.Address) (uint64, error) {
	metrics.EthereumRPCCallCounterInc("pending_nonce_at")
	return c.clt.PendingNonceAt(ctx, account)
}

func (c *ethClientWrapper) SendTransaction(ctx context.Context, tx *ethtypes.Transaction) error {
	metrics.EthereumRPCCallCounterInc("send_transaction")
	return c.clt.SendTransaction(ctx, tx)
}

func (c *ethClientWrapper) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	metrics.EthereumRPCCallCounterInc("suggest_gas_price")
	return c.clt.SuggestGasPrice(ctx)
}

func (c *ethClientWrapper) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	metrics.EthereumRPCCallCounterInc("suggest_gas_tip_cap")
	return c.clt.SuggestGasTipCap(ctx)
}

func (c *ethClientWrapper) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]ethtypes.Log, error) {
	metrics.EthereumRPCCallCounterInc("filter_logs")
	return c.clt.FilterLogs(ctx, query)
}

func (c *ethClientWrapper) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- ethtypes.Log) (ethereum.Subscription, error) {
	metrics.EthereumRPCCallCounterInc("subscribe_filter_logs")
	return c.clt.SubscribeFilterLogs(ctx, query, ch)
}
