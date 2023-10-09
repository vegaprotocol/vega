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

package staking

import (
	"context"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"

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
	event *types.StakeDeposited,
) error {
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
				Start:   event.BlockNumber,
				End:     &event.BlockNumber,
				Context: ctx,
			},
			// user
			[]ethcmn.Address{ethcmn.HexToAddress(event.EthereumAddress)},
			// vega_public_key
			[][32]byte{decodedPubKey})
		if err != nil {
			o.log.Error("Couldn't start filtering on stake deposited event", logging.Error(err))
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

			if !iter.Event.Raw.Removed && // ignore removed events
				hex.EncodeToString(iter.Event.VegaPublicKey[:]) == vegaPubKey &&
				iter.Event.Amount.Cmp(amountDeposited) == 0 &&
				iter.Event.Raw.BlockNumber == event.BlockNumber &&
				uint64(iter.Event.Raw.Index) == event.LogIndex &&
				iter.Event.Raw.TxHash.Hex() == event.TxID {
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
				Start:   event.BlockNumber,
				End:     &event.BlockNumber,
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

			if !iter.Event.Raw.Removed && // ignore removed events
				hex.EncodeToString(iter.Event.VegaPublicKey[:]) == vegaPubKey &&
				iter.Event.Amount.Cmp(amountDeposited) == 0 &&
				iter.Event.Raw.BlockNumber == event.BlockNumber &&
				uint64(iter.Event.Raw.Index) == event.LogIndex &&
				iter.Event.Raw.TxHash.Hex() == event.TxID {
				// now we know the event is OK,
				// just need to check for confirmations
				return o.ethConfirmations.Check(event.BlockNumber)
			}
		}
	}

	return ErrNoStakeRemovedEventFound
}
