package validators

import (
	"context"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

var (
	ErrVegaNodeAlreadyRegisterForChain = errors.New("a vega node is already registered with the blockchain node")
	ErrChainAlreadyRegisterForVega     = errors.New("a blockchain node is already registered with the vega node")
	ErrInvalidChainPubKey              = errors.New("invalid blockchain public key")
	ErrClientNotInitialized            = errors.New("blockchain client not initialised")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/blockchain_client_mock.go -package mocks code.vegaprotocol.io/vega/validators BlockchainClient
type BlockchainClient interface {
	GetStatus(ctx context.Context) (*tmctypes.ResultStatus, error)
	Validators() ([]*tmtypes.Validator, error)
	GenesisValidators() ([]*tmtypes.Validator, error)
}

type Topology struct {
	log *logging.Logger
	clt BlockchainClient
	// tendermint validator pubkey to vega pubkey
	validators map[string]string
	// just pubkeys of vega node for easy lookup
	vegaValidatorRefs map[string]struct{}
	chainValidators   []*tmtypes.Validator

	selfChain *tmtypes.Validator

	// don't recalculate readyness all the time
	ready bool

	mu sync.Mutex
}

func NewTopology(log *logging.Logger, clt BlockchainClient) *Topology {

	t := &Topology{
		log:               log,
		clt:               clt,
		validators:        map[string]string{},
		chainValidators:   []*tmtypes.Validator{},
		vegaValidatorRefs: map[string]struct{}{},
	}

	go t.handleGenesisValidators()
	return t
}

func (t *Topology) Len() int {
	return len(t.vegaValidatorRefs)
}

// Exists check if a vega public key is part of the validator set
func (t *Topology) Exists(key []byte) bool {
	_, ok := t.vegaValidatorRefs[string(key)]
	return ok
}

func (t *Topology) SetChain(clt BlockchainClient) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.clt = clt
}

func (t *Topology) SelfChainPubKey() []byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.selfChain != nil {
		return t.selfChain.PubKey.Bytes()
	}
	return nil
}

func (t *Topology) SelfVegaPubKey() []byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.selfChain != nil {
		return []byte(t.validators[string(t.selfChain.PubKey.Bytes())])
	}
	return nil
}

func (t *Topology) Ready() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.ready {
		return true
	}

	if len(t.chainValidators) <= 0 {
		return false
	}
	for _, v := range t.chainValidators {
		if _, ok := t.validators[string(v.PubKey.Bytes())]; !ok {
			return false
		}
	}
	t.ready = true
	return t.ready
}

func (t *Topology) AddNodeRegistration(nr *types.NodeRegistration) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.validators[string(nr.ChainPubKey)]; ok {
		return ErrVegaNodeAlreadyRegisterForChain
	}
	// check if this tm pubkey exists in the network
	var ok bool
	for _, v := range t.chainValidators {
		if string(v.PubKey.Bytes()) == string(nr.ChainPubKey) {
			ok = true
			break
		}
	}
	if !ok {
		return ErrInvalidChainPubKey
	}

	// then add it to the topology
	t.validators[string(nr.ChainPubKey)] = string(nr.PubKey)
	t.vegaValidatorRefs[string(nr.PubKey)] = struct{}{}
	t.log.Info("new node registration successful",
		logging.String("node-key", hex.EncodeToString(nr.PubKey)),
		logging.String("tm-key", hex.EncodeToString(nr.ChainPubKey)))
	return nil
}

func (t *Topology) handleGenesisValidators() {
	if err := t.loadBlockchainInfos(); err == nil {
		return
	}

	tk := time.NewTicker(500 * time.Millisecond)
	defer tk.Stop()
	for _ = range tk.C {
		// try to get the validators
		err := t.loadBlockchainInfos()
		if err == nil {
			t.log.Info("validator list loaded successfully from tendermint", logging.String("self-tm", hex.EncodeToString(t.selfChain.PubKey.Bytes())))
			t.mu.Lock()
			for _, v := range t.chainValidators {
				t.log.Info("tendermint validator", logging.String("infos", hex.EncodeToString(v.PubKey.Bytes())))
			}
			t.mu.Unlock()
			return
		}

		t.log.Debug("unable to load validators list", logging.Error(err))
	}
}

// load the lists of  validators, and this
func (t *Topology) loadBlockchainInfos() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.clt == nil {
		return ErrClientNotInitialized
	}
	status, err := t.clt.GetStatus(context.Background())
	if err != nil {
		return err
	}

	// no error set the validators stuff
	t.selfChain = &tmtypes.Validator{
		Address:     status.ValidatorInfo.Address,
		PubKey:      status.ValidatorInfo.PubKey,
		VotingPower: status.ValidatorInfo.VotingPower,
	}

	t.chainValidators, err = t.clt.GenesisValidators()
	return err
}
