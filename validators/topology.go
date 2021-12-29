package validators

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"

	vgcrypto "code.vegaprotocol.io/shared/libs/crypto"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
)

var (
	ErrVegaNodeAlreadyRegisterForChain = errors.New("a vega node is already registered with the blockchain node")
	ErrInvalidChainPubKey              = errors.New("invalid blockchain public key")
)

// Broker needs no mocks.
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/wallet_mock.go -package mocks code.vegaprotocol.io/vega/validators Wallet
type Wallet interface {
	PubKey() crypto.PublicKey
	ID() crypto.PublicKey
}

type ValidatorData struct {
	ID              string `json:"id"`
	VegaPubKey      string `json:"vega_pub_key"`
	VegaPubKeyIndex uint32 `json:"vega_pub_key_index"`
	EthereumAddress string `json:"ethereum_address"`
	TmPubKey        string `json:"tm_pub_key"`
	InfoURL         string `json:"info_url"`
	Country         string `json:"country"`
	Name            string `json:"name"`
	AvatarURL       string `json:"avatar_url"`
}

func (v ValidatorData) IsValid() bool {
	if len(v.ID) <= 0 || len(v.VegaPubKey) <= 0 ||
		len(v.EthereumAddress) <= 0 || len(v.TmPubKey) <= 0 {
		return false
	}
	return true
}

// HashVegaPubKey returns hash VegaPubKey encoded as hex string.
func (v ValidatorData) HashVegaPubKey() string {
	return hex.EncodeToString(vgcrypto.Hash([]byte(v.VegaPubKey)))
}

// ValidatorMapping maps a tendermint pubkey with a vega pubkey.
type ValidatorMapping map[string]ValidatorData

type Topology struct {
	log    *logging.Logger
	cfg    Config
	wallet Wallet
	broker Broker

	// vega pubkey to validator data
	validators ValidatorMapping

	chainValidators []string

	// this is the runtime information
	// has the validator been added to the validator set
	isValidator bool

	// this is about the node setup,
	// is the node configured to be a validator
	isValidatorSetup bool

	// key rotations
	pendingPubKeyRotations pendingKeyRotationMapping
	pubKeyChangeListeners  []func(ctx context.Context, oldPubKey, newPubKey string)
	currentBlockHeight     uint64

	mu sync.RWMutex

	tss *topologySnapshotState
}

func NewTopology(
	log *logging.Logger, cfg Config, wallet Wallet, broker Broker, isValidatorSetup bool,
) *Topology {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	t := &Topology{
		log:                    log,
		cfg:                    cfg,
		wallet:                 wallet,
		broker:                 broker,
		validators:             ValidatorMapping{},
		chainValidators:        []string{},
		tss:                    &topologySnapshotState{changed: true},
		pendingPubKeyRotations: pendingKeyRotationMapping{},
		isValidatorSetup:       isValidatorSetup,
	}

	return t
}

// ReloadConf updates the internal configuration.
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
	return t.isValidatorSetup && t.isValidator
}

func (t *Topology) Len() int {
	return len(t.validators)
}

// Get returns validator data based on validator master public key.
func (t *Topology) Get(key string) *ValidatorData {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if data, ok := t.validators[key]; ok {
		return &data
	}

	return nil
}

// AllVegaPubKeys returns all the validators vega public keys.
func (t *Topology) AllVegaPubKeys() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	keys := make([]string, 0, len(t.validators))
	for _, data := range t.validators {
		keys = append(keys, data.VegaPubKey)
	}
	return keys
}

// AllNodeIDs returns all the validators node IDs keys.
func (t *Topology) AllNodeIDs() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	keys := make([]string, 0, len(t.validators))
	for k := range t.validators {
		keys = append(keys, k)
	}
	return keys
}

func (t *Topology) SelfVegaPubKey() string {
	if !t.isValidatorSetup {
		return ""
	}
	return t.wallet.PubKey().Hex()
}

func (t *Topology) SelfNodeID() string {
	if !t.isValidatorSetup {
		return ""
	}
	return t.wallet.ID().Hex()
}

// UpdateValidatorSet updates the chain validator set
// It overwrites the previous set.
func (t *Topology) UpdateValidatorSet(keys []string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.chainValidators = keys
	t.tss.changed = true
}

// IsValidatorNodeID takes a nodeID and returns true if the node is a validator node.
func (t *Topology) IsValidatorNodeID(nodeID string) bool {
	_, ok := t.validators[nodeID]
	return ok
}

// IsValidatorVegaPubKey returns true if the given key is a Vega validator public key.
func (t *Topology) IsValidatorVegaPubKey(pubkey string) (ok bool) {
	defer func() {
		if t.log.GetLevel() <= logging.DebugLevel {
			s := "requested non-existing validator"
			if ok {
				s = "requested existing validator"
			}
			t.log.Debug(s,
				logging.Strings("validators", t.AllVegaPubKeys()),
				logging.String("pubkey", pubkey),
			)
		}
	}()

	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, data := range t.validators {
		if data.VegaPubKey == pubkey {
			return true
		}
	}

	return false
}

func (t *Topology) AddNodeRegistration(ctx context.Context, nr *commandspb.NodeRegistration) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.validators[nr.Id]; ok {
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
		t.log.Error("invalid validator tendermint pubkey",
			logging.Strings("expected-keys", t.chainValidators),
			logging.String("got", nr.ChainPubKey),
		)
		return fmt.Errorf("%s: %w", nr.ChainPubKey, ErrInvalidChainPubKey)
	}

	// then add it to the topology
	t.validators[nr.Id] = ValidatorData{
		ID:              nr.Id,
		VegaPubKey:      nr.VegaPubKey,
		VegaPubKeyIndex: nr.VegaPubKeyIndex,
		EthereumAddress: nr.EthereumAddress,
		TmPubKey:        nr.ChainPubKey,
		InfoURL:         nr.InfoUrl,
		Country:         nr.Country,
		Name:            nr.Name,
		AvatarURL:       nr.AvatarUrl,
	}

	t.tss.changed = true

	// Send event to notify core about new validator
	t.sendValidatorUpdateEvent(ctx, nr)

	t.log.Info("new node registration successful",
		logging.String("id", nr.Id),
		logging.String("vega-key", nr.VegaPubKey),
		logging.String("eth-addr", nr.EthereumAddress),
		logging.String("tm-key", nr.ChainPubKey))
	return nil
}

func (t *Topology) sendValidatorUpdateEvent(ctx context.Context, nr *commandspb.NodeRegistration) {
	t.broker.Send(events.NewValidatorUpdateEvent(
		ctx,
		nr.Id,
		nr.VegaPubKey,
		nr.VegaPubKeyIndex,
		nr.EthereumAddress,
		nr.ChainPubKey,
		nr.InfoUrl,
		nr.Country,
		nr.Name,
		nr.AvatarUrl,
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

	walletID := t.SelfNodeID()
	// tm is base64 encoded, vega is hex
	for tm, data := range state {
		if !data.IsValid() {
			return fmt.Errorf("missing required field from validator data: %#v", data)
		}
		if walletID == data.ID {
			t.isValidator = true
		}

		nr := &commandspb.NodeRegistration{
			Id:              data.ID,
			VegaPubKey:      data.VegaPubKey,
			VegaPubKeyIndex: data.VegaPubKeyIndex,
			EthereumAddress: data.EthereumAddress,
			ChainPubKey:     tm,
			InfoUrl:         data.InfoURL,
			Country:         data.Country,
			Name:            data.Name,
			AvatarUrl:       data.AvatarURL,
		}
		if err := t.AddNodeRegistration(ctx, nr); err != nil {
			return err
		}
	}

	return nil
}
