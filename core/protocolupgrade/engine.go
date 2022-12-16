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

package protocolupgrade

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/txn"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/blang/semver"
	"github.com/cenkalti/backoff"
	"github.com/golang/protobuf/proto"
)

type protocolUpgradeProposal struct {
	blockHeight    uint64
	vegaReleaseTag string
	accepted       map[string]struct{}
}

func protocolUpgradeProposalID(upgradeBlockHeight uint64, vegaReleaseTag string) string {
	return fmt.Sprintf("%v@%v", vegaReleaseTag, upgradeBlockHeight)
}

// TrimReleaseTag removes 'v' or 'V' at the beginning of the tag if present.
func TrimReleaseTag(tag string) string {
	if len(tag) == 0 {
		return tag
	}

	switch tag[0] {
	case 'v', 'V':
		return tag[1:]
	default:
		return tag
	}
}

func (p *protocolUpgradeProposal) approvers() []string {
	accepted := make([]string, 0, len(p.accepted))
	for k := range p.accepted {
		accepted = append(accepted, k)
	}
	sort.Strings(accepted)
	return accepted
}

type ValidatorTopology interface {
	IsTendermintValidator(pubkey string) bool
	IsSelfTendermintValidator() bool
	GetVotingPower(pubkey string) int64
	GetTotalVotingPower() int64
}

type Commander interface {
	CommandSync(ctx context.Context, cmd txn.Command, payload proto.Message, f func(error), bo *backoff.ExponentialBackOff)
}

type Broker interface {
	Send(event events.Event)
}

type Engine struct {
	log            *logging.Logger
	config         Config
	broker         Broker
	topology       ValidatorTopology
	hashKeys       []string
	currentVersion string

	currentBlockHeight uint64
	activeProposals    map[string]*protocolUpgradeProposal
	events             map[string]*eventspb.ProtocolUpgradeEvent
	lock               sync.RWMutex

	requiredMajority   num.Decimal
	upgradeStatus      *types.UpgradeStatus
	coreReadyToUpgrade bool
}

func New(log *logging.Logger, config Config, broker Broker, topology ValidatorTopology, version string) *Engine {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())
	return &Engine{
		activeProposals: map[string]*protocolUpgradeProposal{},
		events:          map[string]*eventspb.ProtocolUpgradeEvent{},
		log:             log,
		config:          config,
		broker:          broker,
		topology:        topology,
		hashKeys:        []string{(&types.PayloadProtocolUpgradeProposals{}).Key()},
		upgradeStatus:   &types.UpgradeStatus{},
		currentVersion:  version,
	}
}

func (e *Engine) OnRequiredMajorityChanged(_ context.Context, requiredMajority num.Decimal) error {
	e.requiredMajority = requiredMajority
	return nil
}

func (e *Engine) IsValidProposal(ctx context.Context, pk string, upgradeBlockHeight uint64, vegaReleaseTag string) error {
	if !e.topology.IsTendermintValidator(pk) {
		// not a tendermint validator, so we don't care about their intention
		return errors.New("only tendermint validator can propose a protocol upgrade")
	}

	if upgradeBlockHeight == 0 {
		return errors.New("upgrade block out of range")
	}

	if upgradeBlockHeight <= e.currentBlockHeight {
		return errors.New("upgrade block earlier than current block height")
	}

	newv, err := semver.Parse(TrimReleaseTag(vegaReleaseTag))
	if err != nil {
		err = fmt.Errorf("invalid protocol version for upgrade received: version (%s), %w", vegaReleaseTag, err)
		e.log.Error("", logging.Error(err))
		return err
	}

	if semver.MustParse(TrimReleaseTag(e.currentVersion)).GT(newv) {
		return errors.New("upgrade version is too old")
	}

	return nil
}

// UpgradeProposal records the intention of a validator to upgrade the protocol to a release tag at block height.
func (e *Engine) UpgradeProposal(ctx context.Context, pk string, upgradeBlockHeight uint64, vegaReleaseTag string) error {
	e.lock.RLock()
	defer e.lock.RUnlock()

	e.log.Debug("Adding protocol upgrade proposal",
		logging.String("validatorPubKey", pk),
		logging.Uint64("upgradeBlockHeight", upgradeBlockHeight),
		logging.String("vegaReleaseTag", vegaReleaseTag),
		logging.String("currentVersion", e.currentVersion),
	)

	if err := e.IsValidProposal(ctx, pk, upgradeBlockHeight, vegaReleaseTag); err != nil {
		return err
	}

	ID := protocolUpgradeProposalID(upgradeBlockHeight, vegaReleaseTag)

	// if the proposed upgrade version is different from the current version we create a new proposal and keep it
	// if it is the same as the current version, this is taken as a signal to withdraw previous vote for another proposal - in this case the validator will have no vote for no proposal.
	if vegaReleaseTag != e.currentVersion {
		// if it's the first time we see this ID, generate an active proposal entry
		if _, ok := e.activeProposals[ID]; !ok {
			e.activeProposals[ID] = &protocolUpgradeProposal{
				blockHeight:    upgradeBlockHeight,
				vegaReleaseTag: vegaReleaseTag,
				accepted:       map[string]struct{}{},
			}
		}

		active := e.activeProposals[ID]
		active.accepted[pk] = struct{}{}
		e.sendAndKeepEvent(ctx, ID, active)

		e.log.Debug("Successfully added protocol upgrade proposal",
			logging.String("validatorPubKey", pk),
			logging.Uint64("upgradeBlockHeight", upgradeBlockHeight),
			logging.String("vegaReleaseTag", vegaReleaseTag),
		)
	}

	activeIDs := make([]string, 0, len(e.activeProposals))
	for k := range e.activeProposals {
		activeIDs = append(activeIDs, k)
	}
	sort.Strings(activeIDs)

	// each validator can only have one vote so if we got a vote for a different release than they voted for before, we remove that vote
	for _, activeID := range activeIDs {
		if activeID == ID {
			continue
		}
		activeProposal := e.activeProposals[activeID]
		// if there is a vote for another proposal from the pk, remove it and send an update
		if _, ok := activeProposal.accepted[pk]; ok {
			delete(activeProposal.accepted, pk)
			e.sendAndKeepEvent(ctx, activeID, activeProposal)

			e.log.Debug("Removed validator vote from previous proposal",
				logging.String("validatorPubKey", pk),
				logging.Uint64("upgradeBlockHeight", activeProposal.blockHeight),
				logging.String("vegaReleaseTag", activeProposal.vegaReleaseTag),
			)
		}
		if len(activeProposal.accepted) == 0 {
			delete(e.activeProposals, activeID)
			delete(e.events, activeID)

			e.log.Debug("Removed previous upgrade proposal",
				logging.String("validatorPubKey", pk),
				logging.Uint64("upgradeBlockHeight", activeProposal.blockHeight),
				logging.String("vegaReleaseTag", activeProposal.vegaReleaseTag),
			)
		}
	}

	return nil
}

func (e *Engine) sendAndKeepEvent(ctx context.Context, ID string, activeProposal *protocolUpgradeProposal) {
	status := eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_PENDING
	if len(activeProposal.approvers()) == 0 {
		status = eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_REJECTED
	}
	evt := events.NewProtocolUpgradeProposalEvent(ctx, activeProposal.blockHeight, activeProposal.vegaReleaseTag, activeProposal.approvers(), status)
	evtProto := evt.Proto()
	e.events[ID] = &evtProto
	e.broker.Send(evt)
}

// TimeForUpgrade is called by abci at the beginning of the block (before calling begin block of the engine) - if a block height for upgrade is set and is equal
// to the last block's height then return true.
func (e *Engine) TimeForUpgrade() bool {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.upgradeStatus.AcceptedReleaseInfo != nil && e.currentBlockHeight-e.upgradeStatus.AcceptedReleaseInfo.UpgradeBlockHeight == 0
}

func (e *Engine) isAccepted(p *protocolUpgradeProposal) bool {
	// if the block is already behind us or we've already accepted a proposal return false
	if p.blockHeight < e.currentBlockHeight {
		return false
	}
	totalVotingPower := e.topology.GetTotalVotingPower()
	if totalVotingPower <= 0 {
		return false
	}
	totalD := num.DecimalFromInt64(totalVotingPower)
	ratio := num.DecimalZero()
	for k := range p.accepted {
		ratio = ratio.Add(num.DecimalFromInt64(e.topology.GetVotingPower(k)).Div(totalD))
	}
	return ratio.GreaterThan(e.requiredMajority)
}

func (e *Engine) getProposalIDs() []string {
	proposalIDs := make([]string, 0, len(e.activeProposals))
	for k := range e.activeProposals {
		proposalIDs = append(proposalIDs, k)
	}
	sort.Strings(proposalIDs)
	return proposalIDs
}

// BeginBlock is called at the beginning of the block, to mark the current block height and check if there are proposals that are accepted/rejected.
// If there is more than one active proposal that is accepted (unlikely) we choose the one with the earliest upgrade block.
func (e *Engine) BeginBlock(ctx context.Context, blockHeight uint64) {
	e.lock.Lock()
	e.currentBlockHeight = blockHeight
	e.lock.Unlock()

	var accepted *protocolUpgradeProposal
	for _, ID := range e.getProposalIDs() {
		pup := e.activeProposals[ID]
		if e.isAccepted(pup) {
			if accepted == nil || accepted.blockHeight > pup.blockHeight {
				accepted = pup
			}
		} else {
			if blockHeight >= pup.blockHeight {
				delete(e.activeProposals, ID)
				delete(e.events, ID)
				e.log.Info("protocol upgrade rejected", logging.String("vega-release-tag", pup.vegaReleaseTag), logging.Uint64("upgrade-block-height", pup.blockHeight))
				e.broker.Send(events.NewProtocolUpgradeProposalEvent(ctx, pup.blockHeight, pup.vegaReleaseTag, pup.approvers(), eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_REJECTED))
			}
		}
	}
	e.lock.Lock()

	if accepted != nil {
		e.upgradeStatus.AcceptedReleaseInfo = &types.ReleaseInfo{
			VegaReleaseTag:     accepted.vegaReleaseTag,
			UpgradeBlockHeight: accepted.blockHeight,
		}
	} else {
		e.upgradeStatus.AcceptedReleaseInfo = &types.ReleaseInfo{}
	}

	e.lock.Unlock()
}

// Cleanup is called by the abci before the final snapshot is taken to clear remaining state. It emits events for the accepted and rejected proposals.
func (e *Engine) Cleanup(ctx context.Context) {
	e.lock.Lock()
	defer e.lock.Unlock()
	for _, ID := range e.getProposalIDs() {
		pup := e.activeProposals[ID]
		status := eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_APPROVED

		if !e.isAccepted(pup) {
			e.log.Info("protocol upgrade rejected", logging.String("vega-release-tag", pup.vegaReleaseTag), logging.Uint64("upgrade-block-height", pup.blockHeight))
			status = eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_REJECTED
		}

		e.broker.Send(events.NewProtocolUpgradeProposalEvent(ctx, pup.blockHeight, pup.vegaReleaseTag, pup.approvers(), status))
		delete(e.activeProposals, ID)
		delete(e.events, ID)
	}
}

// SetCoreReadyForUpgrade is called by abci after a snapshot has been taken and the core process is ready to
// wait for data node to process if connected or to be shutdown.
func (e *Engine) SetCoreReadyForUpgrade() {
	e.lock.Lock()
	defer e.lock.Unlock()
	if int(e.currentBlockHeight)-int(e.upgradeStatus.AcceptedReleaseInfo.UpgradeBlockHeight) != 0 {
		e.log.Panic("can only call SetCoreReadyForUpgrade at the block of the block height for upgrade", logging.Uint64("block-height", e.currentBlockHeight), logging.Int("block-height-for-upgrade", int(e.upgradeStatus.AcceptedReleaseInfo.UpgradeBlockHeight)))
	}
	e.log.Info("marking vega core as ready to shut down")

	e.coreReadyToUpgrade = true
}

func (e *Engine) CoreReadyForUpgrade() bool {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.coreReadyToUpgrade
}

// SetReadyForUpgrade is called by abci after both core and data node has processed all required events before the update.
// This will modify the RPC API.
func (e *Engine) SetReadyForUpgrade() {
	e.lock.Lock()
	defer e.lock.Unlock()
	if !e.coreReadyToUpgrade {
		e.log.Panic("can only call SetReadyForUpgrade when core node is ready up upgrade")
	}
	e.log.Info("marking vega core and data node as ready to shut down")

	e.upgradeStatus.ReadyToUpgrade = true
}

// GetUpgradeStatus is an RPC call that returns the status of an upgrade.
func (e *Engine) GetUpgradeStatus() types.UpgradeStatus {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return *e.upgradeStatus
}
