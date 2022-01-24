package staking

import (
	"context"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcmn "github.com/ethereum/go-ethereum/common"
)

type EthereumClient interface {
	bind.ContractFilterer
}

type OnChainVerifier struct {
	log              *logging.Logger
	ethClient        EthereumClient
	ethConfirmations EthConfirmations

	mu                     sync.RWMutex
	stakingBridgeAddresses []ethcmn.Address
}

func NewOnChainVerifier(
	cfg Config,
	log *logging.Logger,
	ethClient EthereumClient,
	ethConfirmations EthConfirmations,
) *OnChainVerifier {
	log = log.Named("on-chain-verifier")
	log.SetLevel(cfg.Level.Get())

	return &OnChainVerifier{
		log:              log,
		ethClient:        ethClient,
		ethConfirmations: ethConfirmations,
	}
}

func (o *OnChainVerifier) UpdateStakingBridgeAddresses(stakingBridgeAddresses []ethcmn.Address) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.stakingBridgeAddresses = stakingBridgeAddresses

	if o.log.GetLevel() <= logging.DebugLevel {
		var addresses []string
		for _, v := range o.stakingBridgeAddresses {
			addresses = append(addresses, v.Hex())
		}
		o.log.Debug("staking bridge addresses updated",
			logging.Strings("addresses", addresses))
	}
}

func (o *OnChainVerifier) CheckStakeDeposited(
	event *types.StakeDeposited) error {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if o.log.GetLevel() <= logging.DebugLevel {
		o.log.Debug("checking stake deposited event on chain",
			logging.String("event", event.String()),
		)
	}

	decodedPubKeySlice, err := hex.DecodeString(event.VegaPubKey)
	if err != nil {
		o.log.Error("invalid pubkey in stake deposited event", logging.Error(err))
		return err
	}
	var decodedPubKey [32]byte
	copy(decodedPubKey[:], decodedPubKeySlice[0:32])

	for _, address := range o.stakingBridgeAddresses {
		if o.log.GetLevel() <= logging.DebugLevel {
			o.log.Debug("checking stake deposited event on chain",
				logging.String("bridge-address", address.Hex()),
				logging.String("event", event.String()),
			)
		}
		filterer, err := NewStakingFilterer(address, o.ethClient)
		if err != nil {
			o.log.Error("could not instantiate staking bridge filterer",
				logging.String("address", address.Hex()))
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		iter, err := filterer.FilterStakeDeposited(
			&bind.FilterOpts{
				Start:   event.BlockNumber - 1,
				Context: ctx,
			},
			// user
			[]ethcmn.Address{ethcmn.HexToAddress(event.EthereumAddress)},
			// vega_public_key
			[][32]byte{decodedPubKey})
		if err != nil {
			o.log.Error("could not start stake deposited filter",
				logging.Error(err))
			continue
		}
		defer iter.Close()

		vegaPubKey := strings.TrimPrefix(event.VegaPubKey, "0x")
		amountDeposited := event.Amount.BigInt()

		for iter.Next() {
			if o.log.GetLevel() <= logging.DebugLevel {
				o.log.Debug("found stake deposited event on chain",
					logging.String("bridge-address", address.Hex()),
					logging.String("amount", iter.Event.Amount.String()),
					logging.String("user", iter.Event.User.Hex()),
				)
			}

			if hex.EncodeToString(iter.Event.VegaPublicKey[:]) == vegaPubKey &&
				iter.Event.Amount.Cmp(amountDeposited) == 0 &&
				iter.Event.Raw.BlockNumber == event.BlockNumber &&
				uint64(iter.Event.Raw.Index) == event.LogIndex {
				// now we know the event is OK,
				// just need to check for confirmations
				return o.ethConfirmations.Check(event.BlockNumber)
			}
		}
	}

	return ErrNoStakeDepositedEventFound
}

func (o *OnChainVerifier) CheckStakeRemoved(event *types.StakeRemoved) error {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if o.log.GetLevel() <= logging.DebugLevel {
		o.log.Debug("checking stake removed event on chain",
			logging.String("event", event.String()),
		)
	}

	decodedPubKeySlice, err := hex.DecodeString(event.VegaPubKey)
	if err != nil {
		o.log.Error("invalid pubkey inn stake deposited event", logging.Error(err))
		return err
	}
	var decodedPubKey [32]byte
	copy(decodedPubKey[:], decodedPubKeySlice[0:32])

	for _, address := range o.stakingBridgeAddresses {
		if o.log.GetLevel() <= logging.DebugLevel {
			o.log.Debug("checking stake removed event on chain",
				logging.String("bridge-address", address.Hex()),
				logging.String("event", event.String()),
			)
		}
		filterer, err := NewStakingFilterer(address, o.ethClient)
		if err != nil {
			o.log.Error("could not instantiate staking bridge filterer",
				logging.String("address", address.Hex()))
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		iter, err := filterer.FilterStakeRemoved(
			&bind.FilterOpts{
				Start:   event.BlockNumber - 1,
				Context: ctx,
			},
			// user
			[]ethcmn.Address{ethcmn.HexToAddress(event.EthereumAddress)},
			// vega_public_key
			[][32]byte{decodedPubKey})
		if err != nil {
			o.log.Error("could not start stake deposited filter",
				logging.Error(err))
			continue
		}
		defer iter.Close()

		vegaPubKey := strings.TrimPrefix(event.VegaPubKey, "0x")
		amountDeposited := event.Amount.BigInt()

		for iter.Next() {
			if o.log.GetLevel() <= logging.DebugLevel {
				o.log.Debug("found stake removed event on chain",
					logging.String("bridge-address", address.Hex()),
					logging.String("amount", iter.Event.Amount.String()),
					logging.String("user", iter.Event.User.Hex()),
				)
			}

			if hex.EncodeToString(iter.Event.VegaPublicKey[:]) == vegaPubKey &&
				iter.Event.Amount.Cmp(amountDeposited) == 0 &&
				iter.Event.Raw.BlockNumber == event.BlockNumber &&
				uint64(iter.Event.Raw.Index) == event.LogIndex {
				// now we know the event is OK,
				// just need to check for confirmations
				return o.ethConfirmations.Check(event.BlockNumber)
			}
		}
	}

	return ErrNoStakeRemovedEventFound
}
