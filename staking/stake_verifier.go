package staking

import (
	"context"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	vgproto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/logging"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcmn "github.com/ethereum/go-ethereum/common"
)

var (
	ErrNoStakeDepositedEventFound = errors.New("no stake deposited event found")
	ErrNoStakeRemovedEventFound   = errors.New("no stake removed event found")
	ErrNotAnEthereumConfig        = errors.New("not an ethereum config")
)

type TimeTicker interface {
	NotifyOnTick(func(context.Context, time.Time))
}

type EthereumClient interface {
	bind.ContractFilterer
}

type StakeVerifier struct {
	log         *logging.Logger
	cfg         Config
	accs        *Accounting
	currentTime time.Time

	ethClient EthereumClient

	mu                sync.RWMutex
	ethCfg            vgproto.EthereumConfig
	contractAddresses []ethmcm.Address
}

func NewStakeVerifier(
	log *logging.Logger,
	cfg Config,
	accs *Accounting,
	tt TimeTicker,
	ethClient EthereumClient,
) (sv *StakeVerifier) {
	defer func() {
		tt.NotifyOnTick(sv.onTick)
	}()

	return &StakeVerifier{
		log:  log,
		cfg:  cfg,
		accs: accs,
	}
}

func (s *StakeVerifier) onTick(ctx context.Context, t time.Time) {

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
	return nil
}

func (s *StakeVerifier) checkStakeDepositedOnChain(
	event *vgproto.StakeDeposited,
	blockNumber, logIndex uint64,
) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	decodedPubKeySlice, err := hex.DecodeString(event.VegaPublicKey)
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
				blockNumber - 1,
			},
			// user
			[]ethcmn.Address{ethcmn.HexToAddress(event.EthereumAddress)},
			// vega_public_key
			[][32]byte{decodedPubKey})

		amountDeposited := event.Amount.BigInt()
		defer iter.Close()
		var event *StakeDeposited
		for iter.Next() {

		}
	}

	return ErrNoStakeDepositedEventFound
}

func (s StakeVerifier) chheckStakeRemovedOnChain() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return ErrNoStakeRemovedEventFound
}
