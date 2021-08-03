package validators

import (
	"context"
	"encoding/base64"
	"encoding/hex"
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
	SendBatch(events []events.Event)
}

type ValidatorData struct {
	PubKey  string `json:"pub_key"`
	InfoURL string `json:"info_url"`
	Country string `json:"country"`
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
	chainValidators   [][]byte

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
		chainValidators:   [][]byte{},
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
func (t *Topology) Exists(key []byte) bool {
	_, ok := t.vegaValidatorRefs[string(key)]
	return ok
}

func (t *Topology) GetByKey(key []byte) *ValidatorData {
	t.mu.RLock()
	defer t.mu.RUnlock()
	tmPubKey, ok := t.vegaValidatorRefs[string(key)]
	if !ok {
		return nil
	}

	tmPubKeyBase64 := hex.EncodeToString([]byte(tmPubKey))
	if data, ok := t.validators[tmPubKeyBase64]; ok {
		return &data
	}

	return nil
}

func (t *Topology) AllPubKeys() [][]byte {
	t.mu.RLock()
	defer t.mu.RUnlock()
	keys := make([][]byte, 0, len(t.validators))
	for _, data := range t.validators {
		keys = append(keys, []byte(data.PubKey))
	}
	return keys
}

func (t *Topology) SelfVegaPubKey() []byte {
	return t.wallet.PubKeyOrAddress().Bytes()
}

// UpdateValidatorSet updates the chain validator set
// It overwrites the previous set.
func (t *Topology) UpdateValidatorSet(keys [][]byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.chainValidators = keys
}

//IsValidatorNode takes a nodeID and returns true if the node is a validator node
func (t *Topology) IsValidatorNode(nodeID string) bool {
	_, ok := t.validators[nodeID]
	return ok
}

func (t *Topology) AddNodeRegistration(ctx context.Context, nr *commandspb.NodeRegistration) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := hex.EncodeToString(nr.ChainPubKey)
	if _, ok := t.validators[key]; ok {
		return ErrVegaNodeAlreadyRegisterForChain
	}
	// check if this tm pubkey exists in the network
	var ok bool
	for _, k := range t.chainValidators {
		if string(k) == string(nr.ChainPubKey) {
			ok = true
			break
		}
	}
	if !ok {
		return ErrInvalidChainPubKey
	}

	// then add it to the topology
	t.validators[key] = ValidatorData{
		PubKey:  string(nr.PubKey),
		InfoURL: nr.InfoUrl,
		Country: nr.Country,
	}
	t.vegaValidatorRefs[string(nr.PubKey)] = string(nr.ChainPubKey)

	// Send event to notify core about new validator
	t.sendValidatorUpdateEvent(ctx, nr)

	t.log.Info("new node registration successful",
		logging.String("node-key", hex.EncodeToString(nr.PubKey)),
		logging.String("tm-key", hex.EncodeToString(nr.ChainPubKey)))
	return nil
}

func (t *Topology) sendValidatorUpdateEvent(ctx context.Context, nr *commandspb.NodeRegistration) {
	t.broker.Send(events.NewValidatorUpdateEvent(
		ctx,
		string(nr.PubKey),
		string(nr.ChainPubKey),
		nr.InfoUrl,
		nr.Country,
	))
}

func (t *Topology) LoadValidatorsOnGenesis(ctx context.Context, rawstate []byte) error {
	state, err := LoadGenesisState(rawstate)
	if err != nil {
		return err
	}

	pubKey := t.wallet.PubKeyOrAddress().Hex()

	// tm is base64 encoded, vega is hex
	for tm, data := range state {
		tmBytes, err := base64.StdEncoding.DecodeString(tm)
		if err != nil {
			return err
		}

		vegaBytes, err := hex.DecodeString(data.PubKey)
		if err != nil {
			return err
		}

		if pubKey == data.PubKey {
			t.isValidator = true
		}

		nr := &commandspb.NodeRegistration{
			PubKey:      vegaBytes,
			ChainPubKey: tmBytes,
			InfoUrl:     data.InfoURL,
			Country:     data.Country,
		}
		if err := t.AddNodeRegistration(ctx, nr); err != nil {
			return err
		}
	}

	return nil
}
