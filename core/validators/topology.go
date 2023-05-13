// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package validators

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	proto "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	v1 "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/ethereum/go-ethereum/common"
	abcitypes "github.com/tendermint/tendermint/abci/types"
)

var (
	ErrVegaNodeAlreadyRegisterForChain = errors.New("a vega node is already registered with the blockchain node")
	ErrInvalidChainPubKey              = errors.New("invalid blockchain public key")
	ErrIssueSignaturesUnexpectedKind   = errors.New("unexpected node-signature kind")
)

// Broker needs no mocks.
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

type Wallet interface {
	PubKey() crypto.PublicKey
	ID() crypto.PublicKey
	Signer
}

type MultiSigTopology interface {
	IsSigner(address string) bool
	ExcessSigners(addresses []string) bool
	GetSigners() []string
	GetThreshold() uint32
}

type ValidatorPerformance interface {
	ValidatorPerformanceScore(address string, votingPower, totalPower int64, performanceScalingFactor num.Decimal) num.Decimal
	BeginBlock(ctx context.Context, proposer string)
	Serialize() *v1.ValidatorPerformance
	Deserialize(*v1.ValidatorPerformance)
	Reset()
}

// Notary ...
type Notary interface {
	StartAggregate(resID string, kind types.NodeSignatureKind, signature []byte)
	IsSigned(ctx context.Context, id string, kind types.NodeSignatureKind) ([]types.NodeSignature, bool)
	OfferSignatures(kind types.NodeSignatureKind, f func(resources string) []byte)
}

type ValidatorData struct {
	ID               string `json:"id"`
	VegaPubKey       string `json:"vega_pub_key"`
	VegaPubKeyIndex  uint32 `json:"vega_pub_key_index"`
	EthereumAddress  string `json:"ethereum_address"`
	TmPubKey         string `json:"tm_pub_key"`
	InfoURL          string `json:"info_url"`
	Country          string `json:"country"`
	Name             string `json:"name"`
	AvatarURL        string `json:"avatar_url"`
	FromEpoch        uint64 `json:"from_epoch"`
	SubmitterAddress string `json:"submitter_address"`
}

func (v ValidatorData) IsValid() bool {
	if len(v.ID) <= 0 || len(v.VegaPubKey) <= 0 ||
		len(v.EthereumAddress) <= 0 || len(v.TmPubKey) <= 0 {
		return false
	}
	return true
}

// HashVegaPubKey returns hash VegaPubKey encoded as hex string.
func (v ValidatorData) HashVegaPubKey() (string, error) {
	decoded, err := hex.DecodeString(v.VegaPubKey)
	if err != nil {
		return "", fmt.Errorf("couldn't decode public key: %w", err)
	}

	return hex.EncodeToString(vgcrypto.Hash(decoded)), nil
}

// ValidatorMapping maps a tendermint pubkey with a vega pubkey.
type ValidatorMapping map[string]ValidatorData

type validators map[string]*valState

type Topology struct {
	log                  *logging.Logger
	cfg                  Config
	wallets              NodeWallets
	broker               Broker
	timeService          TimeService
	validatorPerformance ValidatorPerformance
	multiSigTopology     MultiSigTopology

	// vega pubkey to validator data
	validators validators

	chainValidators []string

	// this is the runtime information
	// has the validator been added to the validator set
	isValidator bool

	// this is about the node setup,
	// is the node configured to be a validator
	isValidatorSetup bool

	// Vega key rotations
	pendingPubKeyRotations pendingKeyRotationMapping
	pubKeyChangeListeners  []func(ctx context.Context, oldPubKey, newPubKey string)

	// Ethereum key rotations
	// pending are those lined up to happen in a future block, unresolved are ones
	// that have happened but we are waiting to see the old key has been removed from the contract
	pendingEthKeyRotations    pendingEthereumKeyRotationMapping
	unresolvedEthKeyRotations map[string]PendingEthereumKeyRotation

	mu sync.RWMutex

	tss *topologySnapshotState

	rng                *rand.Rand // random generator seeded by block
	currentBlockHeight uint64

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

	cmd              Commander
	checkpointLoaded bool
	notary           Notary
	signatures       Signatures

	// validator heartbeat parameters
	blocksToKeepMalperforming int64
	timeBetweenHeartbeats     time.Duration
	timeToSendHeartbeat       time.Duration

	performanceScalingFactor num.Decimal
}

func (t *Topology) OnEpochEvent(ctx context.Context, epoch types.Epoch) {
	t.epochSeq = epoch.Seq
	if epoch.Action == proto.EpochAction_EPOCH_ACTION_START {
		t.newEpochStarted = true
		// this is needed because when we load a checkpoint on genesis t.rng is not initialised as it's done before calling beginBlock
		// so we need to initialise the rng to something.
		if t.rng == nil {
			t.rng = rand.New(rand.NewSource(epoch.StartTime.Unix()))
		}
	}
	// this is a workaround to the topology loaded from checkpoint before the epoch.
	if t.checkpointLoaded {
		evts := make([]events.Event, 0, len(t.validators))
		seq := num.NewUint(t.epochSeq).String()
		t.checkpointLoaded = false
		nodeIDs := make([]string, 0, len(t.validators))
		for k := range t.validators {
			nodeIDs = append(nodeIDs, k)
		}
		sort.Strings(nodeIDs)
		for _, nid := range nodeIDs {
			node := t.validators[nid]
			if node.rankingScore == nil {
				continue
			}
			evts = append(evts, events.NewValidatorRanking(ctx, seq, node.data.ID, node.rankingScore.StakeScore, node.rankingScore.PerformanceScore, node.rankingScore.RankingScore, protoStatusToString(node.rankingScore.PreviousStatus), protoStatusToString(node.rankingScore.Status), int(node.rankingScore.VotingPower)))
		}
		// send ranking events for all loaded validators so data node knows the current ranking
		t.broker.SendBatch(evts)
	}
}

func NewTopology(
	log *logging.Logger, cfg Config, wallets NodeWallets, broker Broker, isValidatorSetup bool, cmd Commander, msTopology MultiSigTopology, timeService TimeService,
) *Topology {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	t := &Topology{
		log:                           log,
		cfg:                           cfg,
		wallets:                       wallets,
		broker:                        broker,
		timeService:                   timeService,
		validators:                    map[string]*valState{},
		chainValidators:               []string{},
		tss:                           &topologySnapshotState{},
		pendingPubKeyRotations:        pendingKeyRotationMapping{},
		pendingEthKeyRotations:        pendingEthereumKeyRotationMapping{},
		unresolvedEthKeyRotations:     map[string]PendingEthereumKeyRotation{},
		isValidatorSetup:              isValidatorSetup,
		validatorPerformance:          NewValidatorPerformance(log),
		validatorIncumbentBonusFactor: num.DecimalZero(),
		ersatzValidatorsFactor:        num.DecimalZero(),
		multiSigTopology:              msTopology,
		cmd:                           cmd,
		signatures:                    &noopSignatures{log},
	}

	return t
}

// OnEpochLengthUpdate updates the duration of an epoch - which is used to calculate the number of blocks to keep a malperforming validators.
// The number of blocks is calculated as 10 epochs x duration of epoch in seconds, assuming block time is 1s.
func (t *Topology) OnEpochLengthUpdate(ctx context.Context, l time.Duration) error {
	t.blocksToKeepMalperforming = int64(10 * l.Seconds())
	// set time between hearbeats to 1% of the epoch duration in seconds as blocks
	// e.g. if epoch is 1 day = 86400 seconds (blocks) then time between hb becomes 864
	// if epoch is 300 seconds then blocks becomes 50 (lower bound applied).
	blocks := int64(math.Max(l.Seconds()*0.01, 50.0))
	t.timeBetweenHeartbeats = time.Duration(blocks * int64(time.Second))
	t.timeToSendHeartbeat = time.Duration(blocks * int64(time.Second) / 2)
	return nil
}

// SetNotary this is not good, the topology depends on the notary
// which in return also depends on the topology... Luckily they
// do not require recursive calls as for each calls are one offs...
// anyway we may want to extract the code requiring the notary somewhere
// else or have different pattern somehow...
func (t *Topology) SetNotary(notary Notary) {
	t.signatures = NewSignatures(t.log, t.multiSigTopology, notary, t.wallets, t.broker, t.isValidatorSetup)
	t.notary = notary
}

// SetSignatures this is not good, same issue as for SetNotary method.
// This is only used as a helper for testing..
func (t *Topology) SetSignatures(signatures Signatures) {
	t.signatures = signatures
}

// SetIsValidator will set the flag for `self` so that it is considered a real validator
// for example, when a node has announced itself and is accepted as a PENDING validator.
func (t *Topology) SetIsValidator() {
	t.isValidator = true
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

// Len return the number of validators with status Tendermint, the only validators that matter.
func (t *Topology) Len() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	count := 0
	for _, v := range t.validators {
		if v.status == ValidatorStatusTendermint {
			count++
		}
	}
	return count
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

func (t *Topology) IsSelfTendermintValidator() bool {
	return t.IsTendermintValidator(t.SelfVegaPubKey())
}

func (t *Topology) GetTotalVotingPower() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	total := int64(0)
	for _, data := range t.validators {
		total += data.validatorPower
	}
	return total
}

func (t *Topology) GetVotingPower(pubkey string) int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, data := range t.validators {
		if data.data.VegaPubKey == pubkey && data.status == ValidatorStatusTendermint {
			return data.validatorPower
		}
	}

	return int64(0)
}

// IsValidatorVegaPubKey returns true if the given key is a Vega validator public key and the validators is of status Tendermint.
func (t *Topology) IsTendermintValidator(pubkey string) (ok bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, data := range t.validators {
		if data.data.VegaPubKey == pubkey && data.status == ValidatorStatusTendermint {
			return true
		}
	}

	return false
}

func (t *Topology) NumberOfTendermintValidators() uint {
	t.mu.RLock()
	defer t.mu.RUnlock()

	count := uint(0)
	for _, data := range t.validators {
		if data.status == ValidatorStatusTendermint {
			count++
		}
	}
	return count
}

func (t *Topology) BeginBlock(ctx context.Context, req abcitypes.RequestBeginBlock) {
	// we're not adding or removing nodes only potentially changing their state so should be safe
	t.mu.RLock()
	defer t.mu.RUnlock()

	// resetting the seed every block, to both get some more unpredictability and still deterministic
	// and play nicely with snapshot
	currentTime := t.timeService.GetTimeNow()
	t.rng = rand.New(rand.NewSource(currentTime.Unix()))

	t.checkHeartbeat(ctx)
	t.validatorPerformance.BeginBlock(ctx, hex.EncodeToString(req.Header.ProposerAddress))
	t.currentBlockHeight = uint64(req.Header.Height)

	t.signatures.SetNonce(currentTime)
	t.signatures.ClearStaleSignatures()
	t.signatures.OfferSignatures()
	t.keyRotationBeginBlockLocked(ctx)
	t.ethereumKeyRotationBeginBlockLocked(ctx)
}

// OnPerformanceScalingChanged updates the network parameter for performance scaling factor.
func (t *Topology) OnPerformanceScalingChanged(ctx context.Context, scalingFactor num.Decimal) error {
	t.performanceScalingFactor = scalingFactor
	return nil
}

func (t *Topology) AddNewNode(ctx context.Context, nr *commandspb.AnnounceNode, status ValidatorStatus) error {
	// write lock!
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.validators[nr.Id]; ok {
		return ErrVegaNodeAlreadyRegisterForChain
	}

	data := ValidatorData{
		ID:               nr.Id,
		VegaPubKey:       nr.VegaPubKey,
		VegaPubKeyIndex:  nr.VegaPubKeyIndex,
		EthereumAddress:  nr.EthereumAddress,
		TmPubKey:         nr.ChainPubKey,
		InfoURL:          nr.InfoUrl,
		Country:          nr.Country,
		Name:             nr.Name,
		AvatarURL:        nr.AvatarUrl,
		FromEpoch:        nr.FromEpoch,
		SubmitterAddress: nr.SubmitterAddress,
	}

	// then add it to the topology
	t.validators[nr.Id] = &valState{
		data:                            data,
		status:                          status,
		blockAdded:                      int64(t.currentBlockHeight),
		statusChangeBlock:               int64(t.currentBlockHeight),
		lastBlockWithPositiveRanking:    -1,
		numberOfEthereumEventsForwarded: 0,
		heartbeatTracker:                &validatorHeartbeatTracker{},
	}

	if status == ValidatorStatusTendermint {
		t.validators[nr.Id].validatorPower = 10
	}

	rankingScoreStatus := statusToProtoStatus(ValidatorStatusToName[status])
	t.validators[nr.Id].rankingScore = &proto.RankingScore{
		StakeScore:       "0",
		PerformanceScore: "0",
		RankingScore:     "0",
		Status:           rankingScoreStatus,
		PreviousStatus:   statusToProtoStatus("pending"),
		VotingPower:      uint32(t.validators[nr.Id].validatorPower),
	}

	// Send event to notify core about new validator
	t.sendValidatorUpdateEvent(ctx, data, true)
	// Send an event to notify the new validator ranking
	epochSeq := num.NewUint(t.epochSeq).String()
	t.broker.Send(events.NewValidatorRanking(ctx, epochSeq, nr.Id, "0", "0", "0", "pending", ValidatorStatusToName[status], int(t.validators[nr.Id].validatorPower)))
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
		t.epochSeq,
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
		// but so far we haven't seen ourselves as validators for
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
//   - if no we return straight away and consider ourself as non validator
//   - if yes then we do the following checks
//
// - check that all pubkeys / addresses matches what's in the node wallet
//   - if they all match, we are a validator!
//   - if they don't, we panic, that's a missconfiguration from the checkValidatorDataWithSelfWallets, ever the genesis or the node is misconfigured
func (t *Topology) checkValidatorDataWithSelfWallets(data ValidatorData) {
	if data.TmPubKey != t.wallets.GetTendermintPubkey() {
		return
	}

	// if any of these are wrong, the nodewallet didn't import
	// the keys set in the genesis block
	hasError := t.wallets.GetVega().ID().Hex() != data.ID ||
		t.wallets.GetVega().PubKey().Hex() != data.VegaPubKey ||
		common.HexToAddress(t.wallets.GetEthereumAddress()) != common.HexToAddress(data.EthereumAddress)

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

func (t *Topology) IssueSignatures(ctx context.Context, submitter, nodeID string, kind types.NodeSignatureKind) error {
	t.log.Debug("received IssueSignatures txn", logging.String("submitter", submitter), logging.String("nodeID", nodeID))
	currentTime := t.timeService.GetTimeNow()
	switch kind {
	case types.NodeSignatureKindERC20MultiSigSignerAdded:
		return t.signatures.EmitValidatorAddedSignatures(ctx, submitter, nodeID, currentTime)
	case types.NodeSignatureKindERC20MultiSigSignerRemoved:
		return t.signatures.EmitValidatorRemovedSignatures(ctx, submitter, nodeID, currentTime)
	default:
		return ErrIssueSignaturesUnexpectedKind
	}
}
