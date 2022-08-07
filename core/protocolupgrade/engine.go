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
	blockHeight        uint64
	vegaReleaseTag     string
	dataNodeReleaseTag string
	accepted           map[string]struct{}
}

func protocolUpgradeProposalID(upgradeBlockHeight uint64, vegaReleaseTag string, dataNodeReleaseTag string) string {
	return fmt.Sprintf("%v@%v/%v", vegaReleaseTag, upgradeBlockHeight, dataNodeReleaseTag)
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
	NumberOfTendermintValidators() uint
}

type Commander interface {
	CommandSync(ctx context.Context, cmd txn.Command, payload proto.Message, f func(error), bo *backoff.ExponentialBackOff)
}

type Broker interface {
	Send(event events.Event)
}

type Engine struct {
	log      *logging.Logger
	config   Config
	broker   Broker
	topology ValidatorTopology
	hashKeys []string

	currentBlockHeight uint64
	activeProposals    map[string]*protocolUpgradeProposal
	events             map[string]*eventspb.ProtocolUpgradeEvent
	lock               sync.RWMutex

	requiredMajority num.Decimal
	upgradeStatus    *types.UpgradeStatus
}

func New(log *logging.Logger, config Config, broker Broker, topology ValidatorTopology) *Engine {
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
	}
}

func (e *Engine) OnRequiredMajorityChanged(_ context.Context, requiredMajority num.Decimal) error {
	e.requiredMajority = requiredMajority
	return nil
}

// UpgradeProposal records the intention of a validator to upgrade the protocol to a release tag at block height.
func (e *Engine) UpgradeProposal(ctx context.Context, pk string, upgradeBlockHeight uint64, vegaReleaseTag, dataNodeReleaseTag string) error {
	e.lock.RLock()
	defer e.lock.RUnlock()
	if !e.topology.IsTendermintValidator(pk) {
		// not a tendermint validator, so we don't care about their intention
		return nil
	}

	if upgradeBlockHeight <= e.currentBlockHeight {
		return errors.New("upgrade block earlier than current block height")
	}

	if e.upgradeStatus.AcceptedReleaseInfo != nil {
		return errors.New("protocol upgrade already scheduled")
	}

	_, err := semver.Parse(vegaReleaseTag)
	if err != nil {
		err = fmt.Errorf("invalid protocol version for upgrade received: version (%s), %w", vegaReleaseTag, err)
		e.log.Error("", logging.Error(err))
		return err
	}

	_, err = semver.Parse(dataNodeReleaseTag)
	if err != nil {
		err = fmt.Errorf("invalid protocol version for upgrade received: version (%s), %w", dataNodeReleaseTag, err)
		e.log.Error("", logging.Error(err))
		return err
	}

	ID := protocolUpgradeProposalID(upgradeBlockHeight, vegaReleaseTag, dataNodeReleaseTag)

	// if it's the first time we see this ID, generate an active proposal entry
	if _, ok := e.activeProposals[ID]; !ok {
		e.activeProposals[ID] = &protocolUpgradeProposal{
			blockHeight:        upgradeBlockHeight,
			vegaReleaseTag:     vegaReleaseTag,
			dataNodeReleaseTag: dataNodeReleaseTag,
			accepted:           map[string]struct{}{},
		}
	}

	active := e.activeProposals[ID]
	active.accepted[pk] = struct{}{}
	e.sendAndKeepEvent(ctx, ID, active)

	activeIDs := make([]string, 0, len(e.activeProposals))
	for k := range e.activeProposals {
		activeIDs = append(activeIDs, k)
	}
	sort.Strings(activeIDs)
	for _, activeID := range activeIDs {
		if activeID == ID {
			continue
		}
		activeProposal := e.activeProposals[activeID]
		// if there is a vote for another proposal from the pk, remove it and send an update
		if _, ok := activeProposal.accepted[pk]; ok {
			delete(activeProposal.accepted, pk)
			e.sendAndKeepEvent(ctx, activeID, activeProposal)
		}
	}
	return nil
}

func (e *Engine) sendAndKeepEvent(ctx context.Context, ID string, activeProposal *protocolUpgradeProposal) {
	evt := events.NewProtocolUpgradeProposalEvent(ctx, activeProposal.blockHeight, activeProposal.vegaReleaseTag, activeProposal.dataNodeReleaseTag, activeProposal.approvers(), eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_PENDING)
	evtProto := evt.Proto()
	e.events[ID] = &evtProto
	e.broker.Send(evt)
}

// TimeForUpgrade is called by abci at the beginning of the block (before calling begin block of the engine) - if a block height for upgrade is set and is equal
// to the last block's height then return true.
func (e *Engine) TimeForUpgrade() bool {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.upgradeStatus.AcceptedReleaseInfo != nil && e.currentBlockHeight == e.upgradeStatus.AcceptedReleaseInfo.UpgradeBlockHeight
}

func (e *Engine) isAccepted(p *protocolUpgradeProposal) bool {
	count := 0
	// if the block is already behind us or we've already accepted a proposal return false
	if p.blockHeight < e.currentBlockHeight || e.upgradeStatus.AcceptedReleaseInfo != nil {
		return false
	}
	for k := range p.accepted {
		if e.topology.IsTendermintValidator(k) {
			count++
		}
	}
	ratio := num.NewDecimalFromFloat(float64(count)).Div(num.DecimalFromInt64(int64(e.topology.NumberOfTendermintValidators())))
	return ratio.GreaterThan(e.requiredMajority)
}

// BeginBlock is called at the beginning of the block, to mark the current block height and check if there are proposals that are accepted/rejected.
// If there is more than one active proposal that is accepted (unlikely) we choose the one with the earliest upgrade block.
func (e *Engine) BeginBlock(ctx context.Context, blockHeight uint64) {
	e.lock.Lock()
	e.currentBlockHeight = blockHeight
	e.lock.Unlock()

	var accepted *protocolUpgradeProposal

	proposalIDs := make([]string, 0, len(e.activeProposals))
	for k := range e.activeProposals {
		proposalIDs = append(proposalIDs, k)
	}
	sort.Strings(proposalIDs)

	for _, ID := range proposalIDs {
		pup := e.activeProposals[ID]
		if e.isAccepted(pup) {
			if accepted == nil || accepted.blockHeight > pup.blockHeight {
				accepted = pup
			}
		} else {
			if blockHeight > pup.blockHeight {
				delete(e.activeProposals, ID)
				delete(e.events, ID)
				e.log.Info("protocol upgrade rejected", logging.String("vega-release-tag", pup.vegaReleaseTag), logging.String("datanode-release-tag", pup.dataNodeReleaseTag), logging.Uint64("upgrade-block-height", pup.blockHeight))
				e.broker.Send(events.NewProtocolUpgradeProposalEvent(ctx, pup.blockHeight, pup.vegaReleaseTag, pup.dataNodeReleaseTag, pup.approvers(), eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_REJECTED))
			}
		}
	}
	if accepted != nil {
		ID := protocolUpgradeProposalID(accepted.blockHeight, accepted.vegaReleaseTag, accepted.dataNodeReleaseTag)
		delete(e.activeProposals, ID)
		delete(e.events, ID)
		e.broker.Send(events.NewProtocolUpgradeProposalEvent(ctx, accepted.blockHeight, accepted.vegaReleaseTag, accepted.dataNodeReleaseTag, accepted.approvers(), eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_APPROVED))
		e.lock.Lock()
		e.upgradeStatus.AcceptedReleaseInfo = &types.ReleaseInfo{
			VegaReleaseTag:     accepted.vegaReleaseTag,
			DatanodeReleaseTag: accepted.dataNodeReleaseTag,
			UpgradeBlockHeight: accepted.blockHeight,
		}
		e.lock.Unlock()
		e.log.Info("protocol upgrade agreed", logging.String("release-tag", accepted.vegaReleaseTag), logging.String("datanode-release-tag", accepted.dataNodeReleaseTag), logging.Uint64("upgrade-block-height", accepted.blockHeight))

		// if a proposal has been accepted we are auto rejecting all other proposals
		for _, ID := range proposalIDs {
			if pup, ok := e.activeProposals[ID]; ok {
				e.log.Info("protocol upgrade rejected", logging.String("vega-release-tag", pup.vegaReleaseTag), logging.String("datanode-release-tag", pup.dataNodeReleaseTag), logging.Uint64("upgrade-block-height", pup.blockHeight))
				e.broker.Send(events.NewProtocolUpgradeProposalEvent(ctx, pup.blockHeight, pup.vegaReleaseTag, pup.dataNodeReleaseTag, pup.approvers(), eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_REJECTED))
				delete(e.activeProposals, ID)
				delete(e.events, ID)
			}
		}
	}
}

// SetReadyForUpgrade is called by abci after a snapshot has been taken and the process is ready to be shutdown.
func (e *Engine) SetReadyForUpgrade() {
	e.lock.Lock()
	defer e.lock.Unlock()
	if int(e.currentBlockHeight)-int(e.upgradeStatus.AcceptedReleaseInfo.UpgradeBlockHeight) != 1 {
		e.log.Panic("can only call SetReadyForUpgrade at the block following the block height for upgrade", logging.Uint64("block-height", e.currentBlockHeight), logging.Int("block-height-for-upgrade", int(e.upgradeStatus.AcceptedReleaseInfo.UpgradeBlockHeight)))
	}
	e.log.Info("marking vega as ready to shut down")
	e.upgradeStatus.ReadyToUpgrade = true
}

// GetUpgradeStatus is an RPC call that returns the status of an upgrade.
func (e *Engine) GetUpgradeStatus() types.UpgradeStatus {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return *e.upgradeStatus
}
