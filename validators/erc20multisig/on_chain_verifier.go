package erc20multisig

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/bridges/multisig"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcmn "github.com/ethereum/go-ethereum/common"
)

var (
	ErrNoSignerEventFound       = errors.New("no signer event found")
	ErrNoThresholdSetEventFound = errors.New("no threshold set event found")
	ErrUnsupportedSignerEvent   = errors.New("unsupported signer event")
)

type EthereumClient interface {
	bind.ContractFilterer
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/eth_confirmations_mock.go -package mocks code.vegaprotocol.io/vega/validators/erc20multisig EthConfirmations
type EthConfirmations interface {
	Check(uint64) error
}

type OnChainVerifier struct {
	config           Config
	log              *logging.Logger
	ethClient        EthereumClient
	ethConfirmations EthConfirmations

	mu              sync.RWMutex
	multiSigAddress ethcmn.Address
}

func NewOnChainVerifier(
	config Config,
	log *logging.Logger,
	ethClient EthereumClient,
	ethConfirmations EthConfirmations,
) *OnChainVerifier {
	log = log.Named(namedLogger + ".on-chain-verifier")
	log.SetLevel(config.Level.Get())

	return &OnChainVerifier{
		config:           config,
		log:              log,
		ethClient:        ethClient,
		ethConfirmations: ethConfirmations,
	}
}

func (o *OnChainVerifier) UpdateMultiSigAddress(multiSigAddress ethcmn.Address) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.multiSigAddress = multiSigAddress

	if o.log.GetLevel() <= logging.DebugLevel {
		o.log.Debug("multi sig bridge addresses updated",
			logging.String("addresses", o.multiSigAddress.Hex()))
	}
}

func (o *OnChainVerifier) CheckSignerEvent(event *types.SignerEvent) error {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if o.log.GetLevel() <= logging.DebugLevel {
		o.log.Debug("checking signer event on chain",
			logging.String("contract-address", o.multiSigAddress.Hex()),
			logging.String("event", event.String()),
		)
	}

	filterer, err := multisig.NewMultiSigControlFilterer(
		o.multiSigAddress,
		o.ethClient,
	)
	if err != nil {
		o.log.Error("could not instantiate multisig control filterer",
			logging.Error(err))
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch event.Kind {
	case types.SignerEventKindAdded:
		return o.filterSignerAdded(ctx, filterer, event)
	case types.SignerEventKindRemoved:
		return o.filterSignerRemoved(ctx, filterer, event)
	default:
		return ErrUnsupportedSignerEvent
	}
}

func (o *OnChainVerifier) CheckThresholdSetEvent(
	event *types.SignerThresholdSetEvent) error {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if o.log.GetLevel() <= logging.DebugLevel {
		o.log.Debug("checking threshold set event on chain",
			logging.String("contract-address", o.multiSigAddress.Hex()),
			logging.String("event", event.String()),
		)
	}

	filterer, err := multisig.NewMultiSigControlFilterer(
		o.multiSigAddress,
		o.ethClient,
	)
	if err != nil {
		o.log.Error("could not instantiate multisig control filterer",
			logging.Error(err))
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	iter, err := filterer.FilterThresholdSet(
		&bind.FilterOpts{
			Start:   event.BlockNumber,
			End:     &event.BlockNumber,
			Context: ctx,
		},
	)
	if err != nil {
		o.log.Error("Couldn't start filtering on signer added event",
			logging.Error(err))
		return err
	}
	defer iter.Close()

	for iter.Next() {
		if o.log.GetLevel() <= logging.DebugLevel {
			o.log.Debug("found threshold set event on chain",
				logging.Uint16("new-threshold", iter.Event.NewThreshold),
			)
		}

		nonce, _ := big.NewInt(0).SetString(event.Nonce, 10)
		if iter.Event.Raw.BlockNumber == event.BlockNumber &&
			uint64(iter.Event.Raw.Index) == event.LogIndex &&
			iter.Event.NewThreshold == uint16(event.Threshold) &&
			nonce.Cmp(iter.Event.Nonce) == 0 {
			// now we know the event is OK,
			// just need to check for confirmations
			return o.ethConfirmations.Check(event.BlockNumber)
		}
	}

	return ErrNoThresholdSetEventFound
}

func (o *OnChainVerifier) filterSignerAdded(
	ctx context.Context,
	filterer *multisig.MultiSigControlFilterer,
	event *types.SignerEvent,
) error {
	iter, err := filterer.FilterSignerAdded(
		&bind.FilterOpts{
			Start:   event.BlockNumber,
			End:     &event.BlockNumber,
			Context: ctx,
		},
	)
	if err != nil {
		o.log.Error("Couldn't start filtering on signer added event",
			logging.Error(err))
		return err
	}
	defer iter.Close()

	for iter.Next() {
		if o.log.GetLevel() <= logging.DebugLevel {
			o.log.Debug("found signer added event on chain",
				logging.String("new-signer", iter.Event.NewSigner.Hex()),
			)
		}

		nonce, _ := big.NewInt(0).SetString(event.Nonce, 10)
		if iter.Event.Raw.BlockNumber == event.BlockNumber &&
			uint64(iter.Event.Raw.Index) == event.LogIndex &&
			iter.Event.NewSigner.Hex() == event.Address &&
			nonce.Cmp(iter.Event.Nonce) == 0 {
			// now we know the event is OK,
			// just need to check for confirmations
			return o.ethConfirmations.Check(event.BlockNumber)
		}
	}

	return ErrNoSignerEventFound
}

func (o *OnChainVerifier) filterSignerRemoved(
	ctx context.Context,
	filterer *multisig.MultiSigControlFilterer,
	event *types.SignerEvent,
) error {
	iter, err := filterer.FilterSignerRemoved(
		&bind.FilterOpts{
			Start:   event.BlockNumber,
			End:     &event.BlockNumber,
			Context: ctx,
		},
	)
	if err != nil {
		o.log.Error("Couldn't start filtering on signer removed event",
			logging.Error(err))
		return err
	}
	defer iter.Close()

	for iter.Next() {
		if o.log.GetLevel() <= logging.DebugLevel {
			o.log.Debug("found signer removed event on chain",
				logging.String("old-signer", iter.Event.OldSigner.Hex()),
			)
		}

		nonce, _ := big.NewInt(0).SetString(event.Nonce, 10)
		if iter.Event.Raw.BlockNumber == event.BlockNumber &&
			uint64(iter.Event.Raw.Index) == event.LogIndex &&
			iter.Event.OldSigner.Hex() == event.Address &&
			nonce.Cmp(iter.Event.Nonce) == 0 {
			// now we know the event is OK,
			// just need to check for confirmations
			return o.ethConfirmations.Check(event.BlockNumber)
		}
	}

	return ErrNoSignerEventFound
}
