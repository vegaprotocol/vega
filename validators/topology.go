package validators

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	proto "code.vegaprotocol.io/protos/vega"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	v1 "code.vegaprotocol.io/protos/vega/snapshot/v1"
	vgcrypto "code.vegaprotocol.io/shared/libs/crypto"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	abcitypes "github.com/tendermint/tendermint/abci/types"
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
	Signer
}

type MultiSigTopology interface {
	IsSigner(address string) bool
	ExcessSigners(addresses []string) bool
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/val_performance_mock.go -package mocks code.vegaprotocol.io/vega/validators ValidatorPerformance
type ValidatorPerformance interface {
	ValidatorPerformanceScore(address string, votingPower, totalPower int64) num.Decimal
	BeginBlock(ctx context.Context, proposer string)
	Serialize() *v1.ValidatorPerformance
	Deserialize(*v1.ValidatorPerformance)
	Reset()
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
	FromEpoch       uint64 `json:"from_epoch"`
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
	log                  *logging.Logger
	cfg                  Config
	wallets              NodeWallets
	broker               Broker
	validatorPerformance ValidatorPerformance
	currentTime          time.Time
	multiSigTopology     MultiSigTopology

	// vega pubkey to validator data
	validators map[string]*valState

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

	rng *rand.Rand // random generator seeded by block

	// net params
	numberOfTendermintValidators         int
	numberOfErsatzValidators             int
	validatorIncumbentBonusFactor        num.Decimal
	ersatzValidatorsFactor               num.Decimal
	minimumStake                         *num.Uint
	minimumEthereumEventsForNewValidator uint64
	numberEthMultisigSigners             int

	// transient data for updating tendermint on validator voting power changes.
	validatorPowerUpdates []abcitypes.ValidatorUpdate
	epochSeq              uint64
	newEpochStarted       bool

	cmd Commander
}

func (t *Topology) OnEpochEvent(_ context.Context, epoch types.Epoch) {
	t.epochSeq = epoch.Seq
	if epoch.Action == proto.EpochAction_EPOCH_ACTION_START {
		t.newEpochStarted = true
		t.rng = rand.New(rand.NewSource(epoch.StartTime.Unix()))
		t.validatorPerformance.Reset()
	}
}

func NewTopology(
	log *logging.Logger, cfg Config, wallets NodeWallets, broker Broker, isValidatorSetup bool, cmd Commander, msTopology MultiSigTopology,
) *Topology {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	t := &Topology{
		log:                           log,
		cfg:                           cfg,
		wallets:                       wallets,
		broker:                        broker,
		validators:                    map[string]*valState{},
		chainValidators:               []string{},
		tss:                           &topologySnapshotState{changed: true},
		pendingPubKeyRotations:        pendingKeyRotationMapping{},
		isValidatorSetup:              isValidatorSetup,
		validatorPerformance:          NewValidatorPerformance(log),
		validatorIncumbentBonusFactor: num.DecimalZero(),
		ersatzValidatorsFactor:        num.DecimalZero(),
		multiSigTopology:              msTopology,
		cmd:                           cmd,
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
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.validators)
}

// Get returns validator data based on validator master public key.
func (t *Topology) Get(key string) *ValidatorData {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if data, ok := t.validators[key]; ok {
		return &data.data
	}

	return nil
}

// AllVegaPubKeys returns all the validators vega public keys.
func (t *Topology) AllVegaPubKeys() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	keys := make([]string, 0, len(t.validators))
	for _, data := range t.validators {
		keys = append(keys, data.data.VegaPubKey)
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
	return t.wallets.GetVega().PubKey().Hex()
}

func (t *Topology) SelfNodeID() string {
	if !t.isValidatorSetup {
		return ""
	}
	return t.wallets.GetVega().ID().Hex()
}

// IsValidatorNodeID takes a nodeID and returns true if the node is a validator node.
func (t *Topology) IsValidatorNodeID(nodeID string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
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
		if data.data.VegaPubKey == pubkey {
			return true
		}
	}

	return false
}

func (t *Topology) BeginBlock(ctx context.Context, req abcitypes.RequestBeginBlock) {
	// we're not adding or removing nodes only potentially changing their state so should be safe
	t.mu.RLock()
	defer t.mu.RUnlock()

	t.currentTime = req.Header.Time
	// resetting the seed every block, to both get some more unpredictability and still deterministic
	// and play nicely with snapshot
	t.rng = rand.New(rand.NewSource(req.Header.Time.Unix()))

	t.checkHeartbeat(ctx)
	t.validatorPerformance.BeginBlock(ctx, hex.EncodeToString(req.Header.ProposerAddress))
	blockHeight := uint64(req.Header.Height)
	t.currentBlockHeight = blockHeight
	t.keyRotationBeginBlockLocked(ctx)
}

func (t *Topology) AddNewNode(ctx context.Context, nr *commandspb.AnnounceNode, status ValidatorStatus) error {
	// write lock!
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.validators[nr.Id]; ok {
		return ErrVegaNodeAlreadyRegisterForChain
	}

	data := ValidatorData{
		ID:              nr.Id,
		VegaPubKey:      nr.VegaPubKey,
		VegaPubKeyIndex: nr.VegaPubKeyIndex,
		EthereumAddress: nr.EthereumAddress,
		TmPubKey:        nr.ChainPubKey,
		InfoURL:         nr.InfoUrl,
		Country:         nr.Country,
		Name:            nr.Name,
		AvatarURL:       nr.AvatarUrl,
		FromEpoch:       nr.FromEpoch,
	}

	// then add it to the topology
	t.validators[nr.Id] = &valState{
		data:                            data,
		status:                          status,
		blockAdded:                      0,
		statusChangeBlock:               0,
		lastBlockWithPositiveRanking:    -1,
		numberOfEthereumEventsForwarded: 0,
		heartbeatTracker:                &validatorHeartbeatTracker{},
	}

	if status == ValidatorStatusTendermint {
		t.validators[nr.Id].validatorPower = 10
	}

	t.tss.changed = true

	// Send event to notify core about new validator
	t.sendValidatorUpdateEvent(ctx, data, true)

	t.log.Info("new node registration successful",
		logging.String("id", nr.Id),
		logging.String("vega-key", nr.VegaPubKey),
		logging.String("eth-addr", nr.EthereumAddress),
		logging.String("tm-key", nr.ChainPubKey))
	return nil
}

func (t *Topology) sendValidatorUpdateEvent(ctx context.Context, data ValidatorData, added bool) {
	t.broker.Send(events.NewValidatorUpdateEvent(
		ctx,
		data.ID,
		data.VegaPubKey,
		data.VegaPubKeyIndex,
		data.EthereumAddress,
		data.TmPubKey,
		data.InfoURL,
		data.Country,
		data.Name,
		data.AvatarURL,
		data.FromEpoch,
		added,
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

	// tm is base64 encoded, vega is hex
	for tm, data := range state {
		if !data.IsValid() {
			return fmt.Errorf("missing required field from validator data: %#v", data)
		}

		// this node is started and expect to be a validator
		// but so far we haven't seen ourselve as validators for
		// this network.
		if t.isValidatorSetup && !t.isValidator {
			t.checkValidatorDataWithSelfWallets(data)
		}

		nr := &commandspb.AnnounceNode{
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
		if err := t.AddNewNode(ctx, nr, ValidatorStatusTendermint); err != nil {
			return err
		}
	}

	return nil
}

// checkValidatorDataWithSelfWallets in the genesis file, validators data
// are a mapping of a tendermint pubkey to validator info.
// in here we are going to check if:
// - the tm pubkey is the same as the one stored in the nodewallet
//  - if no we return straight away and consider ourself as non validator
//  - if yes then we do the following checks
// - check that all pubkeys / addresses matches what's in the node wallet
//  - if they all match, we are a validator!
//  - if they don't, we panic, that's a missconfiguration from the checkValidatorDataWithSelfWallets, ever the genesis or the node is misconfigured
func (t *Topology) checkValidatorDataWithSelfWallets(data ValidatorData) {
	if data.TmPubKey != t.wallets.GetTendermintPubkey() {
		return
	}

	// if any of these are wrong, the nodewallet didn't import
	// the keys set in the genesis block
	hasError := t.wallets.GetVega().ID().Hex() != data.ID ||
		t.wallets.GetVega().PubKey().Hex() != data.VegaPubKey ||
		strings.TrimLeft(t.wallets.GetEthereumAddress(), "0x") != strings.TrimLeft(data.EthereumAddress, "0x")

	if hasError {
		t.log.Panic("invalid node wallet configurations, the genesis validator mapping differ to the wallets imported by the nodewallet",
			logging.String("genesis-tendermint-pubkey", data.TmPubKey),
			logging.String("nodewallet-tendermint-pubkey", t.wallets.GetTendermintPubkey()),
			logging.String("genesis-vega-pubkey", data.VegaPubKey),
			logging.String("nodewallet-vega-pubkey", t.wallets.GetVega().PubKey().Hex()),
			logging.String("genesis-vega-id", data.ID),
			logging.String("nodewallet-vega-id", t.wallets.GetVega().ID().Hex()),
			logging.String("genesis-ethereum-address", data.EthereumAddress),
			logging.String("nodewallet-ethereum-address", t.wallets.GetEthereumAddress()),
		)
	}

	t.isValidator = true
}
