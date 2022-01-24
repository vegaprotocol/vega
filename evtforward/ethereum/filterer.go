package ethereum

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	vgproto "code.vegaprotocol.io/protos/vega"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/assets/erc20/bridge"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/staking"
	"code.vegaprotocol.io/vega/types"

	"github.com/cenkalti/backoff"
	eth "github.com/ethereum/go-ethereum"
	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	ethbind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcmn "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

const logFiltererLogger = "log-filterer"

type OnEventFound func(*commandspb.ChainEvent)

type Client interface {
	ethbind.ContractFilterer

	CurrentHeight(context.Context) (uint64, error)
	HeaderByNumber(context.Context, *big.Int) (*ethtypes.Header, error)
}

// LogFilterer wraps the Ethereum event filterers to return Vega events.
//
// WARNING: Because of the library implementation, we can only return the block
// number from the last matched event. There is no way to get the last processed
// block using the `Filter*` methods. For example: if the last processed block
// is the block 20 but the last matched event is the block 10, we are going to
// return the block number 10.
// When we loop over the filterer, we re-inject the last matched block number,
// and start processing the blocks starting from 10 and not 20, despite having
// already processed them in the previous call.
type LogFilterer struct {
	log *logging.Logger

	client Client

	collateralBridgeFilterer *bridge.BridgeFilterer
	collateralBridge         types.EthereumContract

	stakingBridgeFilterer *staking.StakingFilterer
	stakingBridge         types.EthereumContract
	collateralBridgeABI   ethabi.ABI
}

func NewLogFilterer(log *logging.Logger, ethClient Client, collateralBridge types.EthereumContract, stakingBridge types.EthereumContract) (*LogFilterer, error) {
	l := log.Named(logFiltererLogger)

	collateralBridgeFilterer, err := bridge.NewBridgeFilterer(collateralBridge.Address(), ethClient)
	if err != nil {
		return nil, fmt.Errorf("couldn't create log filterer for ERC20 brigde: %w", err)
	}

	collateralBridgeABI, err := ethabi.JSON(strings.NewReader(bridge.BridgeABI))
	if err != nil {
		return nil, fmt.Errorf("couldn't load collateral bridge ABI: %w", err)
	}

	stakingBridgeFilterer, err := staking.NewStakingFilterer(stakingBridge.Address(), ethClient)
	if err != nil {
		return nil, fmt.Errorf("couldn't create log filterer for ERC20 brigde: %w", err)
	}

	return &LogFilterer{
		log:                      l,
		client:                   ethClient,
		collateralBridgeABI:      collateralBridgeABI,
		collateralBridgeFilterer: collateralBridgeFilterer,
		collateralBridge:         collateralBridge,
		stakingBridgeFilterer:    stakingBridgeFilterer,
		stakingBridge:            stakingBridge,
	}, nil
}

func (f *LogFilterer) CurrentHeight(ctx context.Context) uint64 {
	currentHeight := new(uint64)

	infiniteRetry(func() error {
		height, err := f.client.CurrentHeight(ctx)
		if err != nil {
			return fmt.Errorf("couldn't get the current height of Ethereum blockchain: %e", err)
		}

		if f.log.IsDebug() {
			f.log.Debug(fmt.Sprintf("Current height of Ethereum blockchain is %d", height))
		}

		*currentHeight = height

		return nil
	})

	return *currentHeight
}

// FilterCollateralEvents retrieves the events from the collateral bridge on
// Ethereum starting at startAt, transform them into ChainEvent, and pass it to
// the OnEventFound callback.
// It returns the block number from the last block that has been entirely
// processed. If the filtering fails in the middle of the block, the previous
// block number is returned.
func (f *LogFilterer) FilterCollateralEvents(ctx context.Context, startAt uint64, cb OnEventFound) uint64 {
	logs, sub := f.subscribeToFilterLogs(ctx, startAt)
	defer func() {
		close(logs)
		sub.Unsubscribe()
	}()

	var event *types.ChainEvent
	var previousBlockChecked uint64
	var blockNumber uint64
	currentBlockNumber := startAt
	for {
		select {
		case log := <-logs:
			event, blockNumber = f.toChainEvent(log)
		case err := <-sub.Err():
			if err != nil {
				f.log.Error("Subscription to Ethereum log filterer encountered an issue, will retry from previous block number", logging.Error(err))
				return previousBlockChecked
			}
			// No more log to process, exiting the filtering, it's safe to
			// return the current block number.
			return currentBlockNumber
		}

		// In case of error, we don't want to return the block number of the
		// block being processed as we may still have events to process for that
		// block. So, we have to keep the number of the last block that has been
		// entirely processed.
		if previousBlockChecked != currentBlockNumber && currentBlockNumber != blockNumber {
			previousBlockChecked = currentBlockNumber
			currentBlockNumber = blockNumber
		}

		cb(event)
	}
}

func (f *LogFilterer) subscribeToFilterLogs(ctx context.Context, startAt uint64) (chan ethtypes.Log, eth.Subscription) {
	query := eth.FilterQuery{
		FromBlock: new(big.Int).SetUint64(startAt),
		Addresses: []ethcmn.Address{
			f.collateralBridge.Address(),
		},
		Topics: [][]ethcmn.Hash{{
			f.collateralBridgeABI.Events["Asset_Deposited"].ID,
			f.collateralBridgeABI.Events["Asset_Withdrawn"].ID,
			f.collateralBridgeABI.Events["Asset_Listed"].ID,
			f.collateralBridgeABI.Events["Asset_Removed"].ID,
		}},
	}

	// This has been taken from the go-ethereum library. I don't know why it has
	// a size of 128.
	logs := make(chan ethtypes.Log, 128)

	var sub eth.Subscription
	infiniteRetry(func() error {
		s, err := f.client.SubscribeFilterLogs(ctx, query, logs)
		if err != nil {
			f.log.Error("Couldn't subscribe to the Ethereum log filterer", logging.Error(err))
			return fmt.Errorf("couldn't subscribe to the Ethereum log filterer: %w", err)
		}
		sub = s
		return nil
	})

	return logs, sub
}

// toChainEvent transform a log to a ChainEvent. It must succeed, otherwise
// it raises a fatal error. At this point, if we can't parse the log, it means:
// - a new event type as been added to the query without being adding support in
//   this method,
// - or, the log doesn't have a backward or forward compatible format.
// Either way, this is a programming error.
func (f *LogFilterer) toChainEvent(log ethtypes.Log) (*types.ChainEvent, uint64) {
	switch log.Topics[0] {
	case f.collateralBridgeABI.Events["Asset_Deposited"].ID:
		event, err := f.collateralBridgeFilterer.ParseAssetDeposited(log)
		if err != nil {
			f.log.Fatal("Couldn't parse AssetDeposited event", logging.Error(err))
			return nil, 0
		}
		f.debugAssetDeposited(event)
		return toERC20Deposit(event), event.Raw.BlockNumber
	case f.collateralBridgeABI.Events["Asset_Withdrawn"].ID:
		event, err := f.collateralBridgeFilterer.ParseAssetWithdrawn(log)
		if err != nil {
			f.log.Fatal("Couldn't parse AssetWithdrawn event", logging.Error(err))
			return nil, 0
		}
		f.debugAssetWithdrawn(event)
		return toERC20Withdraw(event), event.Raw.BlockNumber
	case f.collateralBridgeABI.Events["Asset_Listed"].ID:
		event, err := f.collateralBridgeFilterer.ParseAssetListed(log)
		if err != nil {
			f.log.Fatal("Couldn't parse AssetListed event", logging.Error(err))
			return nil, 0
		}
		f.debugAssetListed(event)
		return toERC20AssetList(event), event.Raw.BlockNumber
	case f.collateralBridgeABI.Events["Asset_Removed"].ID:
		event, err := f.collateralBridgeFilterer.ParseAssetRemoved(log)
		if err != nil {
			f.log.Fatal("Couldn't parse AssetRemoved event", logging.Error(err))
			return nil, 0
		}
		f.debugAssetRemoved(event)
		return toERC20AssetDelist(event), event.Raw.BlockNumber
	default:
		f.log.Fatal("Unsupported Ethereum log event", logging.String("event-id", log.Topics[0].String()))
		return nil, 0
	}
}

func (f *LogFilterer) debugAssetWithdrawn(event *bridge.BridgeAssetWithdrawn) {
	if f.log.IsDebug() {
		f.log.Debug("Found AssetWithdrawn event",
			logging.String("bridge-address", f.collateralBridge.HexAddress()),
			logging.String("user-ethereum-address", event.UserAddress.Hex()),
			logging.String("asset-id", event.AssetSource.Hex()),
		)
	}
}

func toERC20Withdraw(event *bridge.BridgeAssetWithdrawn) *commandspb.ChainEvent {
	return &commandspb.ChainEvent{
		TxId: event.Raw.TxHash.Hex(),
		Event: &commandspb.ChainEvent_Erc20{
			Erc20: &vgproto.ERC20Event{
				Index: uint64(event.Raw.TxIndex),
				Block: event.Raw.BlockNumber,
				Action: &vgproto.ERC20Event_Withdrawal{
					Withdrawal: &vgproto.ERC20Withdrawal{
						VegaAssetId:           event.AssetSource.Hex(),
						TargetEthereumAddress: event.UserAddress.Hex(),
						ReferenceNonce:        event.Nonce.String(),
					},
				},
			},
		},
	}
}

func (f *LogFilterer) debugAssetDeposited(event *bridge.BridgeAssetDeposited) {
	if f.log.IsDebug() {
		f.log.Debug("Found AssetDeposited event",
			logging.String("bridge-address", f.collateralBridge.HexAddress()),
			logging.String("user-ethereum-address", event.UserAddress.Hex()),
			logging.String("user-vega-address", hex.EncodeToString(event.VegaPublicKey[:])),
			logging.String("asset-id", event.AssetSource.Hex()),
		)
	}
}

func toERC20Deposit(event *bridge.BridgeAssetDeposited) *commandspb.ChainEvent {
	return &commandspb.ChainEvent{
		TxId: event.Raw.TxHash.Hex(),
		Event: &commandspb.ChainEvent_Erc20{
			Erc20: &vgproto.ERC20Event{
				Index: uint64(event.Raw.TxIndex),
				Block: event.Raw.BlockNumber,
				Action: &vgproto.ERC20Event_Deposit{
					Deposit: &vgproto.ERC20Deposit{
						VegaAssetId:           event.AssetSource.Hex(),
						SourceEthereumAddress: event.UserAddress.Hex(),
						TargetPartyId:         hex.EncodeToString(event.VegaPublicKey[:]),
						Amount:                event.Amount.String(),
					},
				},
			},
		},
	}
}

func (f *LogFilterer) debugAssetListed(event *bridge.BridgeAssetListed) {
	if f.log.IsDebug() {
		f.log.Debug("Found AssetListed event",
			logging.String("bridge-address", f.collateralBridge.HexAddress()),
			logging.String("asset-id", event.AssetSource.Hex()),
		)
	}
}

func toERC20AssetList(event *bridge.BridgeAssetListed) *commandspb.ChainEvent {
	return &commandspb.ChainEvent{
		TxId: event.Raw.TxHash.Hex(),
		Event: &commandspb.ChainEvent_Erc20{
			Erc20: &vgproto.ERC20Event{
				Index: uint64(event.Raw.TxIndex),
				Block: event.Raw.BlockNumber,
				Action: &vgproto.ERC20Event_AssetList{
					AssetList: &vgproto.ERC20AssetList{
						VegaAssetId: event.AssetSource.Hex(),
					},
				},
			},
		},
	}
}

func (f *LogFilterer) debugAssetRemoved(event *bridge.BridgeAssetRemoved) {
	if f.log.IsDebug() {
		f.log.Debug("Found AssetRemoved event",
			logging.String("bridge-address", f.collateralBridge.HexAddress()),
			logging.String("asset-id", event.AssetSource.Hex()),
		)
	}
}

func toERC20AssetDelist(event *bridge.BridgeAssetRemoved) *commandspb.ChainEvent {
	return &commandspb.ChainEvent{
		TxId: event.Raw.TxHash.Hex(),
		Event: &commandspb.ChainEvent_Erc20{
			Erc20: &vgproto.ERC20Event{
				Index: uint64(event.Raw.TxIndex),
				Block: event.Raw.BlockNumber,
				Action: &vgproto.ERC20Event_AssetDelist{
					AssetDelist: &vgproto.ERC20AssetDelist{
						VegaAssetId: event.AssetSource.Hex(),
					},
				},
			},
		},
	}
}

// StakeDepositedEvents retrieves the StakeDeposited events on Ethereum starting
// from startAt, transform them into StakeDeposited, and pass it to the
// OnEventFound callback.
// It returns block number from the last event matched.
func (f *LogFilterer) StakeDepositedEvents(ctx context.Context, startAt uint64, cb OnEventFound) uint64 {
	var iter *staking.StakingStakeDepositedIterator

	infiniteRetry(func() error {
		i, err := f.stakingBridgeFilterer.FilterStakeDeposited(
			&ethbind.FilterOpts{
				Start:   startAt,
				Context: ctx,
			},
			[]ethcmn.Address{},
			[][32]byte{},
		)
		if err != nil {
			f.log.Error("Couldn't retrieve StakeDeposited event from staking bridge", logging.Error(err))
			return fmt.Errorf("couldn't retrieve StakeDeposited event from staking bridge: %w", err)
		}
		iter = i
		return nil
	})

	defer func() {
		if err := iter.Close(); err != nil {
			f.log.Error("Couldn't close StakeDeposited iterator, meaning subscription to Ethereum might still be alive", logging.Error(err))
		}
	}()

	blockTimesFetcher := NewBlockTimeFetcher(nil, f.client)
	lastBlockChecked := uint64(0)
	for iter.Next() {
		f.debugStakeDeposited(iter.Event)

		blockTime := blockTimesFetcher.TimeForBlock(ctx, iter.Event.Raw.BlockNumber)
		cb(toStakeDeposited(iter.Event, blockTime))
		lastBlockChecked = iter.Event.Raw.BlockNumber
	}

	return lastBlockChecked
}

func (f *LogFilterer) debugStakeDeposited(event *staking.StakingStakeDeposited) {
	if f.log.IsDebug() {
		f.log.Debug("Found StakeDeposited event",
			logging.String("bridge-address", f.stakingBridge.HexAddress()),
			logging.String("user-ethereum-address", event.User.Hex()),
			logging.String("user-vega-address", hex.EncodeToString(event.VegaPublicKey[:])),
			logging.String("amount", event.Amount.String()),
		)
	}
}

func toStakeDeposited(event *staking.StakingStakeDeposited, blockTime uint64) *commandspb.ChainEvent {
	return &commandspb.ChainEvent{
		TxId: event.Raw.TxHash.Hex(),
		Event: &commandspb.ChainEvent_StakingEvent{
			StakingEvent: &vgproto.StakingEvent{
				Index: uint64(event.Raw.TxIndex),
				Block: event.Raw.BlockNumber,
				Action: &vgproto.StakingEvent_StakeDeposited{
					StakeDeposited: &vgproto.StakeDeposited{
						EthereumAddress: event.User.Hex(),
						VegaPublicKey:   hex.EncodeToString(event.VegaPublicKey[:]),
						Amount:          event.Amount.String(),
						BlockTime:       int64(blockTime),
					},
				},
			},
		},
	}
}

// StakeRemovedEvents retrieves the StakeRemoved events on Ethereum starting
// from startAt, transform them into StakeRemoved, and pass it to the
// OnEventFound callback.
// It returns block number from the last event matched.
func (f *LogFilterer) StakeRemovedEvents(ctx context.Context, startAt uint64, cb OnEventFound) uint64 {
	var iter *staking.StakingStakeRemovedIterator

	infiniteRetry(func() error {
		i, err := f.stakingBridgeFilterer.FilterStakeRemoved(
			&ethbind.FilterOpts{
				Start:   startAt,
				Context: ctx,
			},
			[]ethcmn.Address{},
			[][32]byte{},
		)
		if err != nil {
			f.log.Error("Couldn't retrieve StakeRemoved event from staking bridge", logging.Error(err))
			return fmt.Errorf("couldn't retrieve StakeRemoved event from staking bridge: %w", err)
		}
		iter = i
		return nil
	})

	defer func() {
		if err := iter.Close(); err != nil {
			f.log.Error("Couldn't close StakeRemoved iterator, meaning subscription to Ethereum might still be alive", logging.Error(err))
		}
	}()

	blockTimesFetcher := NewBlockTimeFetcher(f.log, f.client)
	lastBlockChecked := uint64(0)
	for iter.Next() {
		f.debugStakeRemoved(iter.Event)

		blockTime := blockTimesFetcher.TimeForBlock(ctx, iter.Event.Raw.BlockNumber)
		cb(toStakeRemoved(iter.Event, blockTime))
		lastBlockChecked = iter.Event.Raw.BlockNumber
	}

	return lastBlockChecked
}

func (f *LogFilterer) debugStakeRemoved(event *staking.StakingStakeRemoved) {
	if f.log.IsDebug() {
		f.log.Debug("Found StakeRemoved event",
			logging.String("bridge-address", f.stakingBridge.HexAddress()),
			logging.String("user-ethereum-address", event.User.Hex()),
			logging.String("user-vega-address", hex.EncodeToString(event.VegaPublicKey[:])),
			logging.String("amount", event.Amount.String()),
		)
	}
}

func toStakeRemoved(event *staking.StakingStakeRemoved, blockTime uint64) *commandspb.ChainEvent {
	return &commandspb.ChainEvent{
		TxId: event.Raw.TxHash.Hex(),
		Event: &commandspb.ChainEvent_StakingEvent{
			StakingEvent: &vgproto.StakingEvent{
				Index: uint64(event.Raw.TxIndex),
				Block: event.Raw.BlockNumber,
				Action: &vgproto.StakingEvent_StakeRemoved{
					StakeRemoved: &vgproto.StakeRemoved{
						EthereumAddress: event.User.Hex(),
						VegaPublicKey:   hex.EncodeToString(event.VegaPublicKey[:]),
						Amount:          event.Amount.String(),
						BlockTime:       int64(blockTime),
					},
				},
			},
		},
	}
}

// blockTimeFetcher wraps the retrieval of the block time on Ethereum with a
// naive cache in front of it, so we can save calls to Ethereum when there
// several logs contained in the same block.
// I am using this method because I couldn't find this information on the
// event returned by the library.
type blockTimeFetcher struct {
	log    *logging.Logger
	client Client

	// cachedTimes keeps track of the time for a given block.
	// The key is the block number. The value is the time.
	cachedTimes map[uint64]uint64
}

func NewBlockTimeFetcher(log *logging.Logger, client Client) *blockTimeFetcher {
	return &blockTimeFetcher{
		log:         log,
		client:      client,
		cachedTimes: map[uint64]uint64{},
	}
}

// TimeForBlock retrieves the block time for a given block number. It returns
// the value from the cache if present, otherwise, it retrieves it through the
// Ethereum API.
func (f *blockTimeFetcher) TimeForBlock(ctx context.Context, blockNumber uint64) uint64 {
	blockTime, ok := f.cachedTimes[blockNumber]
	if !ok {
		blockTime = f.fetchTimeByBlock(ctx, blockNumber)
		f.cachedTimes[blockNumber] = blockTime
	}

	return blockTime
}

func (f *blockTimeFetcher) fetchTimeByBlock(ctx context.Context, blockNumber uint64) uint64 {
	var header *ethtypes.Header
	infiniteRetry(func() error {
		h, err := f.client.HeaderByNumber(ctx, new(big.Int).SetUint64(blockNumber))
		if err != nil {
			f.log.Error(fmt.Sprintf("Couldn't retrieve the block header with number \"%d\" on the staking bridge", blockNumber), logging.Error(err))
			return fmt.Errorf("couldn't retrieve the block header with number \"%d\" on the staking bridge: %w", blockNumber, err)
		}
		header = h
		return nil
	})
	return header.Time
}

// We are retrying infinitely, on purpose, as we don't want the Ethereum
// Forwarder to exit for any reason. Failure is not an option.
func infiniteRetry(fn backoff.Operation) {
	// No need to retrieve the error, as we are waiting indefinitely for a
	// success.
	_ = backoff.Retry(fn, backoff.NewConstantBackOff(durationBetweenTwoRetry))
}
