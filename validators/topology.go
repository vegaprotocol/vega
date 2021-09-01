package validators

import (
	"context"
	"errors"
	"sync"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
)

var (
	ErrVegaNodeAlreadyRegisterForChain = errors.New("a vega node is already registered with the blockchain node")
	ErrInvalidChainPubKey              = errors.New("invalid blockchain public key")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/wallet_mock.go -package mocks code.vegaprotocol.io/vega/validators Wallet
type Wallet interface {
	PubKeyOrAddress() crypto.PublicKeyOrAddress
}

// Broker needs no mocks
type Broker interface {
	Send(event events.Event)
}

type ValidatorData struct {
	VegaPubKey      string `json:"vega_pub_key"`
	EthereumAddress string `json:"ethereum_address"`
	InfoURL         string `json:"info_url"`
	Country         string `json:"country"`
}

// ValidatorMapping maps a tendermint pubkey with a vega pubkey
type ValidatorMapping map[string]ValidatorData

type Topology struct {
	log    *logging.Logger
	cfg    Config
	wallet Wallet
	broker Broker

	// tendermint validator pubkey to vega pubkey
	validators ValidatorMapping
	// vega pubkeys to tendermint pub keys for easy lookup
	vegaValidatorRefs map[string]string
	chainValidators   []string

	isValidator bool

	mu sync.RWMutex
}

func NewTopology(log *logging.Logger, cfg Config, wallet Wallet, broker Broker) *Topology {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	t := &Topology{
		log:               log,
		cfg:               cfg,
		wallet:            wallet,
		broker:            broker,
		validators:        ValidatorMapping{},
		chainValidators:   []string{},
		vegaValidatorRefs: map[string]string{},
	}

	return t
}

// ReloadConf updates the internal configuration
func (t *Topology) ReloadConf(cfg Config) {
	t.log.Info("reloading configuration")
	if t.log.GetLevel() != cfg.Level.Get() {
		t.log.Info("updating log level",
			logging.String("old", t.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		t.log.SetLevel(cfg.Level.Get())
	}

	t.cfg = cfg
}

func (t *Topology) IsValidator() bool {
	return t.isValidator
}

func (t *Topology) Len() int {
	return len(t.vegaValidatorRefs)
}

// Exists check if a vega public key is part of the validator set
func (t *Topology) Exists(key string) bool {
	_, ok := t.vegaValidatorRefs[key]
	if t.log.GetLevel() <= logging.DebugLevel {
		validators := make([]string, 0, len(t.vegaValidatorRefs))
		for k := range t.vegaValidatorRefs {
			validators = append(validators, k)
		}
		var s = "requested non-existing validator"
		if ok {
			s = "requested existing validator"
		}
		t.log.Debug(s,
			logging.Strings("validators", validators),
			logging.String("pubkey", key),
		)
	}
	return ok
}

// Get returns validator data based on validator public key
func (t *Topology) Get(key string) *ValidatorData {
	t.mu.RLock()
	defer t.mu.RUnlock()
	tmPubKey, ok := t.vegaValidatorRefs[key]
	if !ok {
		return nil
	}

	if data, ok := t.validators[tmPubKey]; ok {
		return &data
	}

	return nil
}

func (t *Topology) AllPubKeys() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	keys := make([]string, 0, len(t.validators))
	for _, data := range t.validators {
		keys = append(keys, data.VegaPubKey)
	}
	return keys
}

func (t *Topology) SelfVegaPubKey() string {
	return t.wallet.PubKeyOrAddress().Hex()
}

// UpdateValidatorSet updates the chain validator set
// It overwrites the previous set.
func (t *Topology) UpdateValidatorSet(keys []string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.chainValidators = keys
}

// IsValidatorNode takes a nodeID and returns true if the node is a validator node
func (t *Topology) IsValidatorNode(nodeID string) bool {
	_, ok := t.validators[nodeID]
	return ok
}

func (t *Topology) AddNodeRegistration(ctx context.Context, nr *commandspb.NodeRegistration) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.validators[nr.ChainPubKey]; ok {
		return ErrVegaNodeAlreadyRegisterForChain
	}
	// check if this tm pubkey exists in the network
	var ok bool
	for _, k := range t.chainValidators {
		if k == nr.ChainPubKey {
			ok = true
			break
		}
	}
	if !ok {
		return ErrInvalidChainPubKey
	}

	// then add it to the topology
	t.validators[nr.ChainPubKey] = ValidatorData{
		VegaPubKey:      nr.VegaPubKey,
		EthereumAddress: nr.EthereumAddress,
		InfoURL:         nr.InfoUrl,
		Country:         nr.Country,
	}
	t.vegaValidatorRefs[nr.VegaPubKey] = nr.ChainPubKey

	// Send event to notify core about new validator
	t.sendValidatorUpdateEvent(ctx, nr)

	t.log.Info("new node registration successful",
		logging.String("vega-key", nr.VegaPubKey),
		logging.String("eth-addr", nr.EthereumAddress),
		logging.String("tm-key", nr.ChainPubKey))
	return nil
}

func (t *Topology) sendValidatorUpdateEvent(ctx context.Context, nr *commandspb.NodeRegistration) {
	t.broker.Send(events.NewValidatorUpdateEvent(
		ctx,
		nr.VegaPubKey,
		nr.EthereumAddress,
		nr.ChainPubKey,
		nr.InfoUrl,
		nr.Country,
	))
}

func (t *Topology) LoadValidatorsOnGenesis(ctx context.Context, rawstate []byte) (err error) {
	t.log.Debug("Entering validators.Topology.LoadValidatorsOnGenesis")
	defer func() {
		t.log.Debug("Leaving validators.Topology.LoadValidatorsOnGenesis without error")
		if err != nil {
			t.log.Debug("Failure in validators.Topology.LoadValidatorsOnGenesis", logging.Error(err))
		}
	}()

	state, err := LoadGenesisState(rawstate)
	if err != nil {
		return err
	}

	pubKey := t.wallet.PubKeyOrAddress().Hex()

	// tm is base64 encoded, vega is hex
	for tm, data := range state {
		if pubKey == data.VegaPubKey {
			t.isValidator = true
		}

		nr := &commandspb.NodeRegistration{
			VegaPubKey:      data.VegaPubKey,
			EthereumAddress: data.EthereumAddress,
			ChainPubKey:     tm,
			InfoUrl:         data.InfoURL,
			Country:         data.Country,
		}
		if err := t.AddNodeRegistration(ctx, nr); err != nil {
			return err
		}
	}

	return nil
}
