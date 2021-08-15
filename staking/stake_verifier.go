package staking

import (
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"sync"
	"time"

	vgproto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/validators"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcmn "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

var (
	ErrNoStakeDepositedEventFound = errors.New("no stake deposited event found")
	ErrNoStakeRemovedEventFound   = errors.New("no stake removed event found")
	ErrNotAnEthereumConfig        = errors.New("not an ethereum config")
	ErrMissingConfirmations       = errors.New("missing confirmations")
)

type TimeTicker interface {
	NotifyOnTick(func(context.Context, time.Time))
}

type EthereumClient interface {
	bind.ContractFilterer
	HeaderByNumber(context.Context, *big.Int) (*ethtypes.Header, error)
}

// Witness provide foreign chain resources validations
//go:generate go run github.com/golang/mock/mockgen -destination mocks/witness_mock.go -package mocks code.vegaprotocol.io/vega/banking Witness
type Witness interface {
	StartCheck(validators.Resource, func(interface{}, bool), time.Time) error
}

type StakeVerifier struct {
	log         *logging.Logger
	cfg         Config
	accs        *Accounting
	currentTime time.Time
	witness     Witness
	broker      Broker

	ethClient EthereumClient

	mu                sync.RWMutex
	ethCfg            vgproto.EthereumConfig
	contractAddresses []ethcmn.Address

	ethConfirmations EthereumConfirmations

	pendingSDs []*pendingSD
	pendingSRs []*pendingSR

	finalizedEvents []*types.StakingEvent
}

type pendingSD struct {
	*types.StakeDeposited
	check func() error
}

func (p pendingSD) GetID() string { return p.ID }
func (p *pendingSD) Check() error { return p.check() }

type pendingSR struct {
	*types.StakeRemoved
	check func() error
}

func (p pendingSR) GetID() string { return p.ID }
func (p *pendingSR) Check() error { return p.check() }

func NewStakeVerifier(
	log *logging.Logger,
	cfg Config,
	accs *Accounting,
	tt TimeTicker,
	ethClient EthereumClient,
	witness Witness,
	broker Broker,
) (sv *StakeVerifier) {
	defer func() {
		tt.NotifyOnTick(sv.onTick)
	}()

	return &StakeVerifier{
		log:     log,
		cfg:     cfg,
		accs:    accs,
		witness: witness,
		ethConfirmations: EthereumConfirmations{
			ethClient: ethClient,
		},
	}
}

func (s *StakeVerifier) ProcessStakeRemove(
	ctx context.Context, event *types.StakeRemoved) error {
	pending := &pendingSR{
		StakeRemoved: event,
		check:        func() error { return s.checkStakeRemovedOnChain(event) },
	}

	s.pendingSRs = append(s.pendingSRs, pending)

	return s.witness.StartCheck(
		pending, s.onEventVerified, s.currentTime.Add(2*time.Hour))
}

func (s *StakeVerifier) onEventVerified(event interface{}, ok bool) {
	// switch pending := event.(type) {
	// case *pendingSD:
	// 	evt = pending.IntoStakingEvent()
	// case *pendingSR:
	// 	evt = pending.IntoStakingEvent()
	// }

	pending, ok := event.(interface{ IntoStakingEvent() *types.StakingEvent })
	if !ok {
		s.log.Error("stake verifier received invalid event")
	}

	evt := pending.IntoStakingEvent()
	evt.Status = types.StakingEventStatusRejected
	if ok {
		evt.Status = types.StakingEventStatusAccepted
	}
	evt.FinalizedAt = s.currentTime.UnixNano()
	s.finalizedEvents = append(s.finalizedEvents, evt)
}

func (s *StakeVerifier) ProcessStakeDeposited(
	ctx context.Context, event *types.StakeDeposited) error {
	return nil
}

func (s *StakeVerifier) OnEthereumConfigUpdate(rawcfg interface{}) error {
	cfg, ok := rawcfg.(*vgproto.EthereumConfig)
	if !ok {
		return ErrNotAnEthereumConfig
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.ethCfg = *cfg
	s.contractAddresses = nil
	for _, address := range s.ethCfg.StakingBridgeAddresses {
		s.contractAddresses = append(
			s.contractAddresses, ethcmn.HexToAddress(address))
	}

	s.ethConfirmations.Set(uint64(s.ethCfg.Confirmations))

	return nil
}

func (s *StakeVerifier) checkStakeDepositedOnChain(
	event *types.StakeDeposited) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	decodedPubKeySlice, err := hex.DecodeString(event.VegaPubKey)
	if err != nil {
		s.log.Error("invalid pubkey inn stake deposited event", logging.Error(err))
		return err
	}
	var decodedPubKey [32]byte
	copy(decodedPubKey[:], decodedPubKeySlice[0:32])

	for _, address := range s.contractAddresses {
		filterer, err := NewStakingFilterer(
			address, s.ethClient)
		if err != nil {
			s.log.Error("could not instantiate staking bridge filterer",
				logging.String("address", address.Hex()))
			continue
		}

		iter, err := filterer.FilterStakeDeposited(
			&bind.FilterOpts{
				Start: event.BlockNumber - 1,
			},
			// user
			[]ethcmn.Address{ethcmn.HexToAddress(event.EthereumAddress)},
			// vega_public_key
			[][32]byte{decodedPubKey})
		if err != nil {
			s.log.Error("could not start stake deposited filter",
				logging.Error(err))
		}
		defer iter.Close()

		vegaPubKey := strings.TrimPrefix(event.VegaPubKey, "0x")
		amountDeposited := event.Amount.BigInt()

		for iter.Next() {
			if hex.EncodeToString(iter.Event.VegaPublicKey[:]) == vegaPubKey &&
				iter.Event.Amount.Cmp(amountDeposited) == 0 &&
				iter.Event.Raw.BlockNumber == event.BlockNumber &&
				uint64(iter.Event.Raw.Index) == event.LogIndex {
				// now we know the event is OK,
				// just need to check for confirmations
				return s.ethConfirmations.Check(event.BlockNumber)
			}
		}
	}

	return ErrNoStakeDepositedEventFound
}

func (s *StakeVerifier) checkStakeRemovedOnChain(event *types.StakeRemoved) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	decodedPubKeySlice, err := hex.DecodeString(event.VegaPubKey)
	if err != nil {
		s.log.Error("invalid pubkey inn stake deposited event", logging.Error(err))
		return err
	}
	var decodedPubKey [32]byte
	copy(decodedPubKey[:], decodedPubKeySlice[0:32])

	for _, address := range s.contractAddresses {
		filterer, err := NewStakingFilterer(
			address, s.ethClient)
		if err != nil {
			s.log.Error("could not instantiate staking bridge filterer",
				logging.String("address", address.Hex()))
			continue
		}

		iter, err := filterer.FilterStakeRemoved(
			&bind.FilterOpts{
				Start: event.BlockNumber - 1,
			},
			// user
			[]ethcmn.Address{ethcmn.HexToAddress(event.EthereumAddress)},
			// vega_public_key
			[][32]byte{decodedPubKey})
		if err != nil {
			s.log.Error("could not start stake deposited filter",
				logging.Error(err))
		}
		defer iter.Close()

		vegaPubKey := strings.TrimPrefix(event.VegaPubKey, "0x")
		amountDeposited := event.Amount.BigInt()

		for iter.Next() {
			if hex.EncodeToString(iter.Event.VegaPublicKey[:]) == vegaPubKey &&
				iter.Event.Amount.Cmp(amountDeposited) == 0 &&
				iter.Event.Raw.BlockNumber == event.BlockNumber &&
				uint64(iter.Event.Raw.Index) == event.LogIndex {
				// now we know the event is OK,
				// just need to check for confirmations
				return s.ethConfirmations.Check(event.BlockNumber)
			}
		}
	}

	return ErrNoStakeRemovedEventFound
}

func (s *StakeVerifier) onTick(ctx context.Context, t time.Time) {
	s.currentTime = t

	for _, evt := range s.finalizedEvents {
		s.accs.AddEvent(ctx, evt)
		s.broker.Send(events.NewStakingEvent(ctx, *evt))
	}

	s.finalizedEvents = nil
}

type EthereumConfirmations struct {
	ethClient EthereumClient

	mu                  sync.Mutex
	required            uint64
	curHeight           uint64
	curHeightLastUpdate time.Time
}

func (e *EthereumConfirmations) Set(confirmations uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.required = confirmations
}

func (e *EthereumConfirmations) Check(block uint64) error {
	curBlock, err := e.currentHeight(context.Background())
	if err != nil {
		return err
	}

	if curBlock < block ||
		(curBlock-block) < uint64(e.required) {
		return ErrMissingConfirmations
	}

	return nil
}

func (e *EthereumConfirmations) currentHeight(
	ctx context.Context) (uint64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// if last update of the heigh was more that 15 seconds
	// ago, we try to update, we assume an eth block takes
	// ~15 seconds
	now := time.Now()
	if e.curHeightLastUpdate.Add(15).Before(now) {
		// get the last block header
		h, err := e.ethClient.HeaderByNumber(context.Background(), nil)
		if err != nil {
			return e.curHeight, err
		}
		e.curHeightLastUpdate = now
		e.curHeight = h.Number.Uint64()
	}

	return e.curHeight, nil
}
