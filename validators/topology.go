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
	ErrVegaNodeAlreadyRegisterForTM = errors.New("a vega node is already registrered with the tendermint node")
	ErrTMAlreadyRegisterForVega     = errors.New("a TM node is already registered with the vega node")
	ErrInvalidTMPubKey              = errors.New("invalid tendermint public key")
	ErrClientNotInitialized         = errors.New("blockchain client not initiazlized")
)

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
	tmValidators      []*tmtypes.Validator

	selfTM *tmtypes.Validator

	mu sync.Mutex
}

func NewTopology(log *logging.Logger, clt BlockchainClient) *Topology {

	t := &Topology{
		log:          log,
		clt:          clt,
		validators:   map[string]string{},
		tmValidators: []*tmtypes.Validator{},
	}

	go t.handleGenesisValidators()
	return t
}

func (t *Topology) Len() int {
	return len(t.vegaValidatorRefs)
}

// Exists check if a vega public key is part of the validator set
func (t *Topology) Exists(key []byte) bool {
	pubKey := hex.EncodeToString(PubKey)
	_, ok := p.nodes[pubKey]
	return ok
}

func (t *Topology) SetChain(clt BlockchainClient) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.clt = clt
}

func (t *Topology) SelfTMPubKey() []byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.selfTM != nil {
		return t.selfTM.PubKey.Bytes()
	}
	return nil
}

func (t *Topology) Ready() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.tmValidators) <= 0 {
		return false
	}
	for _, v := range t.tmValidators {
		if _, ok := t.validators[string(v.PubKey.Bytes())]; !ok {
			return false
		}
	}
	return true
}

func (t *Topology) AddNodeRegistration(nr *types.NodeRegistration) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.validators[string(nr.TmPubKey)]; ok {
		return ErrVegaNodeAlreadyRegisterForTM
	}
	// check if this tm pubkey exists in the network
	var ok bool
	for _, v := range t.tmValidators {
		if string(v.PubKey.Bytes()) == string(nr.TmPubKey) {
			ok = true
		}
	}
	if !ok {
		return ErrInvalidTMPubKey
	}

	// then add it to the topology
	t.validators[string(nr.TmPubKey)] = string(nr.PubKey)
	t.log.Info("new node registration successful",
		logging.String("node-key", hex.EncodeToString(nr.PubKey)),
		logging.String("tm-key", hex.EncodeToString(nr.TmPubKey)))
	return nil
}

func (t *Topology) handleGenesisValidators() {
	tk := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-tk.C:
			// try to get the validators
			err := t.loadBlockchainInfos()
			if err == nil {
				t.log.Info("validator list loaded successfully from tendermint", logging.String("self-tm", hex.EncodeToString(t.selfTM.PubKey.Bytes())))
				t.mu.Lock()
				for _, v := range t.tmValidators {
					t.log.Info("tendermint validator", logging.String("infos", hex.EncodeToString(v.PubKey.Bytes())))
				}
				t.mu.Unlock()
				return
			}
			t.log.Info("unable to load validators list", logging.Error(err))
		}
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
	t.selfTM = &tmtypes.Validator{
		Address:     status.ValidatorInfo.Address,
		PubKey:      status.ValidatorInfo.PubKey,
		VotingPower: status.ValidatorInfo.VotingPower,
	}

	t.tmValidators, err = t.clt.GenesisValidators()
	return err
}
