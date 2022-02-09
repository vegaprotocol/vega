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

const (
	logFiltererLogger = "log-filterer"

	eventAssetListed    = "Asset_Listed"
	eventAssetRemoved   = "Asset_Removed"
	eventAssetDeposited = "Asset_Deposited"
	eventAssetWithdrawn = "Asset_Withdrawn"
	eventStakeDeposited = "Stake_Deposited"
	eventStakeRemoved   = "Stake_Removed"
)

// Assets ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/assets_mock.go -package mocks code.vegaprotocol.io/vega/evtforward/ethereum Assets
type Assets interface {
	GetVegaIDFromEthereumAddress(string) string
}

type OnEventFound func(*commandspb.ChainEvent)

type Client interface {
	ethbind.ContractFilterer

	CurrentHeight(context.Context) (uint64, error)
	HeaderByNumber(context.Context, *big.Int) (*ethtypes.Header, error)
}

// LogFilterer wraps the Ethereum event filterers to return Vega events.
type LogFilterer struct {
	log *logging.Logger

	client Client

	collateralBridgeABI      ethabi.ABI
	collateralBridgeFilterer *bridge.BridgeFilterer
	collateralBridge         types.EthereumContract

	stakingBridgeABI      ethabi.ABI
	stakingBridgeFilterer *staking.StakingFilterer
	stakingBridge         types.EthereumContract

	vestingBridgeFilterer *staking.StakingFilterer
	vestingBridge         types.EthereumContract

	assets Assets
}

func NewLogFilterer(
	log *logging.Logger,
	ethClient Client,
	collateralBridge types.EthereumContract,
	stakingBridge types.EthereumContract,
	vestingBridge types.EthereumContract,
	assets Assets,
) (*LogFilterer, error) {
	l := log.Named(logFiltererLogger)

	collateralBridgeFilterer, err := bridge.NewBridgeFilterer(collateralBridge.Address(), ethClient)
	if err != nil {
		return nil, fmt.Errorf("couldn't create log filterer for ERC20 brigde: %w", err)
	}

	collateralBridgeABI, err := ethabi.JSON(strings.NewReader(bridge.BridgeMetaData.ABI))
	if err != nil {
		return nil, fmt.Errorf("couldn't load collateral bridge ABI: %w", err)
	}

	stakingBridgeFilterer, err := staking.NewStakingFilterer(stakingBridge.Address(), ethClient)
	if err != nil {
		return nil, fmt.Errorf("couldn't create log filterer for staking brigde: %w", err)
	}

	stakingBridgeABI, err := ethabi.JSON(strings.NewReader(staking.StakingMetaData.ABI))
	if err != nil {
		return nil, fmt.Errorf("couldn't load staking bridge ABI: %w", err)
	}

	vestingBridgeFilterer, err := staking.NewStakingFilterer(vestingBridge.Address(), ethClient)
	if err != nil {
		return nil, fmt.Errorf("couldn't create log filterer for vesting brigde: %w", err)
	}

	return &LogFilterer{
		log:                      l,
		client:                   ethClient,
		collateralBridgeABI:      collateralBridgeABI,
		collateralBridgeFilterer: collateralBridgeFilterer,
		collateralBridge:         collateralBridge,
		stakingBridgeABI:         stakingBridgeABI,
		stakingBridgeFilterer:    stakingBridgeFilterer,
		stakingBridge:            stakingBridge,
		vestingBridgeFilterer:    vestingBridgeFilterer,
		vestingBridge:            vestingBridge,
		assets:                   assets,
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
			f.log.Debug("Current height of Ethereum blockchain has been retrieved",
				logging.Uint64("height", *currentHeight),
			)
		}

		*currentHeight = height

		return nil
	})

	return *currentHeight
}

// FilterCollateralEvents retrieves the events from the collateral bridge on
// Ethereum starting at startAt, transform them into ChainEvent, and pass it to
// the OnEventFound callback.
// The properties startAt and stopAt are inclusive.
func (f *LogFilterer) FilterCollateralEvents(ctx context.Context, startAt, stopAt uint64, cb OnEventFound) {
	query := f.newCollateralBridgeQuery(startAt, stopAt)
	logs := f.filterLogs(ctx, query)

	var event *types.ChainEvent
	for _, log := range logs {
		event = f.toCollateralChainEvent(log)
		cb(event)
	}
}

// FilterStakingEvents retrieves the events from the staking bridge on
// Ethereum starting at startAt, transform them into ChainEvent, and pass it to
// the OnEventFound callback.
// The properties startAt and stopAt are inclusive.
func (f *LogFilterer) FilterStakingEvents(ctx context.Context, startAt, stopAt uint64, cb OnEventFound) {
	query := f.newStakingBridgeQuery(startAt, stopAt)
	logs := f.filterLogs(ctx, query)

	var event *types.ChainEvent
	blockTimesFetcher := NewBlockTimeFetcher(f.log, f.client)
	for _, log := range logs {
		blockTime := blockTimesFetcher.TimeForBlock(ctx, log.BlockNumber)
		event = f.toStakingChainEvent(log, blockTime)
		cb(event)
	}
}

// FilterVestingEvents retrieves the events from the vesting bridge on
// Ethereum starting at startAt, transform them into ChainEvent, and pass it to
// the OnEventFound callback.
// The properties startAt and stopAt are inclusive.
func (f *LogFilterer) FilterVestingEvents(ctx context.Context, startAt, stopAt uint64, cb OnEventFound) {
	query := f.newVestingBridgeQuery(startAt, stopAt)
	logs := f.filterLogs(ctx, query)

	var event *types.ChainEvent
	blockTimesFetcher := NewBlockTimeFetcher(f.log, f.client)
	for _, log := range logs {
		blockTime := blockTimesFetcher.TimeForBlock(ctx, log.BlockNumber)
		event = f.toStakingChainEvent(log, blockTime)
		cb(event)
	}
}

func (f *LogFilterer) filterLogs(ctx context.Context, query eth.FilterQuery) []ethtypes.Log {
	var logs []ethtypes.Log

	infiniteRetry(func() error {
		l, err := f.client.FilterLogs(ctx, query)
		if err != nil {
			f.log.Error("Couldn't subscribe to the Ethereum log filterer", logging.Error(err))
			return fmt.Errorf("couldn't subscribe to the Ethereum log filterer: %w", err)
		}
		logs = l
		return nil
	})

	return logs
}

func (f *LogFilterer) newCollateralBridgeQuery(startAt uint64, stopAt uint64) eth.FilterQuery {
	query := eth.FilterQuery{
		FromBlock: new(big.Int).SetUint64(startAt),
		ToBlock:   new(big.Int).SetUint64(stopAt),
		Addresses: []ethcmn.Address{
			f.collateralBridge.Address(),
		},
		Topics: [][]ethcmn.Hash{{
			f.collateralBridgeABI.Events[eventAssetDeposited].ID,
			f.collateralBridgeABI.Events[eventAssetWithdrawn].ID,
			f.collateralBridgeABI.Events[eventAssetListed].ID,
			f.collateralBridgeABI.Events[eventAssetRemoved].ID,
		}},
	}
	return query
}

func (f *LogFilterer) newStakingBridgeQuery(startAt uint64, stopAt uint64) eth.FilterQuery {
	query := eth.FilterQuery{
		FromBlock: new(big.Int).SetUint64(startAt),
		ToBlock:   new(big.Int).SetUint64(stopAt),
		Addresses: []ethcmn.Address{
			f.stakingBridge.Address(),
		},
		Topics: [][]ethcmn.Hash{{
			f.stakingBridgeABI.Events[eventStakeDeposited].ID,
			f.stakingBridgeABI.Events[eventStakeRemoved].ID,
		}},
	}
	return query
}

func (f *LogFilterer) newVestingBridgeQuery(startAt uint64, stopAt uint64) eth.FilterQuery {
	query := eth.FilterQuery{
		FromBlock: new(big.Int).SetUint64(startAt),
		ToBlock:   new(big.Int).SetUint64(stopAt),
		Addresses: []ethcmn.Address{
			f.vestingBridge.Address(),
		},
		Topics: [][]ethcmn.Hash{{
			// We use staking bridge ABI as stacking and vesting bridge share
			// the same ABI.
			f.stakingBridgeABI.Events[eventStakeDeposited].ID,
			f.stakingBridgeABI.Events[eventStakeRemoved].ID,
		}},
	}
	return query
}

// toCollateralChainEvent transform a log to a ChainEvent. It must succeed, otherwise
// it raises a fatal error. At this point, if we can't parse the log, it means:
// - a new event type as been added to the query without being adding support in
//   this method,
// - or, the log doesn't have a backward or forward compatible format.
// Either way, this is a programming error.
func (f *LogFilterer) toCollateralChainEvent(log ethtypes.Log) *types.ChainEvent {
	switch log.Topics[0] {
	case f.collateralBridgeABI.Events[eventAssetDeposited].ID:
		event, err := f.collateralBridgeFilterer.ParseAssetDeposited(log)
		if err != nil {
			f.log.Fatal("Couldn't parse AssetDeposited event", logging.Error(err))
			return nil
		}
		f.debugAssetDeposited(event)
		return f.toERC20Deposit(event)
	case f.collateralBridgeABI.Events[eventAssetWithdrawn].ID:
		event, err := f.collateralBridgeFilterer.ParseAssetWithdrawn(log)
		if err != nil {
			f.log.Fatal("Couldn't parse AssetWithdrawn event", logging.Error(err))
			return nil
		}
		f.debugAssetWithdrawn(event)
		return f.toERC20Withdraw(event)
	case f.collateralBridgeABI.Events[eventAssetListed].ID:
		event, err := f.collateralBridgeFilterer.ParseAssetListed(log)
		if err != nil {
			f.log.Fatal("Couldn't parse AssetListed event", logging.Error(err))
			return nil
		}
		f.debugAssetListed(event)
		return toERC20AssetList(event)
	case f.collateralBridgeABI.Events[eventAssetRemoved].ID:
		event, err := f.collateralBridgeFilterer.ParseAssetRemoved(log)
		if err != nil {
			f.log.Fatal("Couldn't parse AssetRemoved event", logging.Error(err))
			return nil
		}
		f.debugAssetRemoved(event)
		return toERC20AssetDelist(event)
	default:
		f.log.Fatal("Unsupported Ethereum log event", logging.String("event-id", log.Topics[0].String()))
		return nil
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

func (f *LogFilterer) toERC20Withdraw(event *bridge.BridgeAssetWithdrawn) *commandspb.ChainEvent {
	return &commandspb.ChainEvent{
		TxId: event.Raw.TxHash.Hex(),
		Event: &commandspb.ChainEvent_Erc20{
			Erc20: &vgproto.ERC20Event{
				Index: uint64(event.Raw.Index),
				Block: event.Raw.BlockNumber,
				Action: &vgproto.ERC20Event_Withdrawal{
					Withdrawal: &vgproto.ERC20Withdrawal{
						VegaAssetId:           f.assets.GetVegaIDFromEthereumAddress(event.AssetSource.Hex()),
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

func (f *LogFilterer) toERC20Deposit(event *bridge.BridgeAssetDeposited) *commandspb.ChainEvent {
	return &commandspb.ChainEvent{
		TxId: event.Raw.TxHash.Hex(),
		Event: &commandspb.ChainEvent_Erc20{
			Erc20: &vgproto.ERC20Event{
				Index: uint64(event.Raw.Index),
				Block: event.Raw.BlockNumber,
				Action: &vgproto.ERC20Event_Deposit{
					Deposit: &vgproto.ERC20Deposit{
						VegaAssetId:           f.assets.GetVegaIDFromEthereumAddress(event.AssetSource.Hex()),
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
				Index: uint64(event.Raw.Index),
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
				Index: uint64(event.Raw.Index),
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

// toStakingChainEvent transform a log to a ChainEvent. It must succeed, otherwise
// it raises a fatal error. At this point, if we can't parse the log, it means:
// - a new event type as been added to the query without being adding support in
//   this method,
// - or, the log doesn't have a backward or forward compatible format.
// Either way, this is a programming error.
func (f *LogFilterer) toStakingChainEvent(log ethtypes.Log, blockTime uint64) *types.ChainEvent {
	switch log.Topics[0] {
	case f.stakingBridgeABI.Events[eventStakeDeposited].ID:
		event, err := f.stakingBridgeFilterer.ParseStakeDeposited(log)
		if err != nil {
			f.log.Fatal("Couldn't parse StakeDeposited event", logging.Error(err))
			return nil
		}
		f.debugStakeDeposited(event)

		return toStakeDeposited(event, blockTime)
	case f.stakingBridgeABI.Events[eventStakeRemoved].ID:
		event, err := f.stakingBridgeFilterer.ParseStakeRemoved(log)
		if err != nil {
			f.log.Fatal("Couldn't parse StakeRemoved event", logging.Error(err))
			return nil
		}
		f.debugStakeRemoved(event)
		return toStakeRemoved(event, blockTime)
	default:
		f.log.Fatal("Unsupported Ethereum log event", logging.String("event-id", log.Topics[0].String()))
		return nil
	}
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
				Index: uint64(event.Raw.Index),
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
				Index: uint64(event.Raw.Index),
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
// the value from the cache if present, otherwise, it retrieves it from the
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
			f.log.Error("Couldn't retrieve the block header for given number on the staking bridge",
				logging.Uint64("block-number", blockNumber),
				logging.Error(err),
			)
			return fmt.Errorf("couldn't retrieve the block header with number \"%d\" on the staking bridge: %w", blockNumber, err)
		}
		header = h
		return nil
	})
	return header.Time
}

// We are retrying infinitely, on purpose, as we don't want the Ethereum
// Forwarder to exit, and this under any circumstances. Failure is not an option.
func infiniteRetry(fn backoff.Operation) {
	// No need to retrieve the error, as we are waiting indefinitely for a
	// success.
	_ = backoff.Retry(fn, backoff.NewConstantBackOff(durationBetweenTwoRetry))
}
