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
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/bridges"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

var ErrNoPendingSignaturesForNodeID = errors.New("there are no pending signatures for the given nodeID")

type Signatures interface {
	PreparePromotionsSignatures(
		ctx context.Context,
		currentTime time.Time,
		epochSeq uint64,
		previousState map[string]StatusAddress,
		newState map[string]StatusAddress,
	)
	SetNonce(currentTime time.Time)
	PrepareValidatorSignatures(ctx context.Context, validators []NodeIDAddress, epochSeq uint64, added bool)
	EmitValidatorAddedSignatures(ctx context.Context, submitter, nodeID string, currentTime time.Time) error
	EmitValidatorRemovedSignatures(ctx context.Context, submitter, nodeID string, currentTime time.Time) error
	ClearStaleSignatures()
	SerialisePendingSignatures() *snapshot.ToplogySignatures
	RestorePendingSignatures(*snapshot.ToplogySignatures)
}

type signatureData struct {
	NodeID     string
	EthAddress string
	Nonce      *num.Uint
	EpochSeq   uint64
	Added      bool
}

type ERC20Signatures struct {
	log              *logging.Logger
	notary           Notary
	multiSigTopology MultiSigTopology
	multisig         *bridges.ERC20MultiSigControl
	lastNonce        *num.Uint
	broker           Broker
	isValidatorSetup bool

	// stored nonce's etc. to be able to generate signatures to remove/add an ethereum address from the multisig bundle
	pendingSignatures map[string]*signatureData
	issuedSignatures  map[string]struct{}
}

func NewSignatures(
	log *logging.Logger,
	multiSigTopology MultiSigTopology,
	notary Notary,
	nw NodeWallets,
	broker Broker,
	isValidatorSetup bool,
) *ERC20Signatures {
	s := &ERC20Signatures{
		log:               log,
		notary:            notary,
		multiSigTopology:  multiSigTopology,
		lastNonce:         num.UintZero(),
		broker:            broker,
		isValidatorSetup:  isValidatorSetup,
		pendingSignatures: map[string]*signatureData{},
		issuedSignatures:  map[string]struct{}{},
	}
	if isValidatorSetup {
		s.multisig = bridges.NewERC20MultiSigControl(nw.GetEthereum())
	}
	return s
}

type StatusAddress struct {
	Status           ValidatorStatus
	EthAddress       string
	SubmitterAddress string
}

type NodeIDAddress struct {
	NodeID           string
	EthAddress       string
	SubmitterAddress string
}

func (s *ERC20Signatures) getSignatureData(nodeID string, added bool) []*signatureData {
	r := []*signatureData{}
	for _, p := range s.pendingSignatures {
		if p.NodeID == nodeID && p.Added == added {
			r = append(r, p)
		}
	}
	sort.SliceStable(r, func(i, j int) bool {
		return r[i].EthAddress < r[j].EthAddress
	})
	return r
}

func (s *ERC20Signatures) PreparePromotionsSignatures(
	ctx context.Context,
	currentTime time.Time,
	epochSeq uint64,
	previousState map[string]StatusAddress,
	newState map[string]StatusAddress,
) {
	toAdd := []NodeIDAddress{}
	toRemove := []NodeIDAddress{}

	// first let's cover all the previous validators
	for k, state := range previousState {
		if val, ok := newState[k]; !ok {
			// in this case we were a validator before, but not even in the validator set anymore,
			// we can remove it.
			if state.Status == ValidatorStatusTendermint {
				toRemove = append(toRemove, NodeIDAddress{k, state.EthAddress, state.SubmitterAddress})
			}
		} else {
			// we've been removed from the validator set then
			if state.Status == ValidatorStatusTendermint && val.Status != ValidatorStatusTendermint {
				toRemove = append(toRemove, NodeIDAddress{k, state.EthAddress, state.SubmitterAddress})
			} else if state.Status != ValidatorStatusTendermint && val.Status == ValidatorStatusTendermint {
				// now we've become a validator
				toAdd = append(toAdd, NodeIDAddress{k, state.EthAddress, state.SubmitterAddress})
			}
		}
	}

	// now let's cover all which might have been added but might not have been in the previousStates?
	// is that even possible?
	for k, val := range newState {
		if _, ok := previousState[k]; !ok {
			// this is a new validator which didn't exist before
			if val.Status == ValidatorStatusTendermint {
				toAdd = append(toAdd, NodeIDAddress{k, val.EthAddress, val.SubmitterAddress})
			}
		}
	}

	s.PrepareValidatorSignatures(ctx, toAdd, epochSeq, true)
	s.PrepareValidatorSignatures(ctx, toRemove, epochSeq, false)

	// check if the node being added has supplied a submitterAddress because if it has we can automatically emit
	// signatures for the node to add itself
	for _, v := range toAdd {
		if v.SubmitterAddress != "" {
			s.log.Debug("sending automatic add signatures", logging.String("submitter", v.SubmitterAddress), logging.String("nodeID", v.NodeID))
			s.EmitValidatorAddedSignatures(ctx, v.SubmitterAddress, v.NodeID, currentTime)
		}
	}

	// for each node being removed check if the Tendermint nodes have a submitter address because if they do
	// we can automatically emit remove signatures for them
	for _, r := range toRemove {
		for _, v := range newState {
			if v.SubmitterAddress != "" && v.Status == ValidatorStatusTendermint {
				s.log.Debug("sending automatic remove signatures", logging.String("submitter", v.SubmitterAddress), logging.String("nodeID", r.NodeID))
				s.EmitValidatorRemovedSignatures(ctx, v.SubmitterAddress, r.NodeID, currentTime)
			}
		}
	}
}

func (s *ERC20Signatures) SetNonce(t time.Time) {
	s.lastNonce = num.NewUint(uint64(t.Unix()) + 1)
}

// PrepareValidatorSignatures make nonces and store the data needed to generate signatures to add/remove from the multisig control contract.
func (s *ERC20Signatures) PrepareValidatorSignatures(ctx context.Context, validators []NodeIDAddress, epochSeq uint64, added bool) {
	sort.Slice(validators, func(i, j int) bool {
		return validators[i].EthAddress < validators[j].EthAddress
	})

	for _, signer := range validators {
		d := &signatureData{
			NodeID:     signer.NodeID,
			EthAddress: signer.EthAddress,
			Nonce:      s.lastNonce.Clone(),
			EpochSeq:   epochSeq,
			Added:      added,
		}
		s.lastNonce.AddUint64(s.lastNonce, 1)

		// we're ok to override whatever is in here since we can't need to both add and remove an eth-address at the same time
		// so we'll replace it either with the correct action, or with the same action but with a later epoch which is fine
		// because you can still get signatures to do what you need
		s.pendingSignatures[signer.EthAddress] = d
		s.log.Debug("prepared multisig signatures for", logging.Bool("added", added), logging.String("id", signer.NodeID), logging.String("eth-address", signer.EthAddress))
	}
}

// EmitValidatorAddedSignatures emit signatures to add nodeID's ethereum address onto that can be submitter to the contract by submitter.
func (s *ERC20Signatures) EmitValidatorAddedSignatures(ctx context.Context, submitter, nodeID string, currentTime time.Time) error {
	toEmit := s.getSignatureData(nodeID, true)
	if len(toEmit) == 0 {
		return ErrNoPendingSignaturesForNodeID
	}

	evts := []events.Event{}
	for _, pending := range toEmit {
		var sig []byte
		nonce := pending.Nonce

		resid := hex.EncodeToString(vgcrypto.Hash([]byte(submitter + nonce.String())))
		if _, ok := s.issuedSignatures[resid]; ok {
			// we've already issued a signature for this submitter we don't want to do it again, it'll annoy the notary engine
			s.log.Debug("add signatures already issued", logging.String("submitter", submitter), logging.String("add-address", pending.EthAddress))
			continue
		}

		if s.isValidatorSetup {
			signature, err := s.multisig.AddSigner(pending.EthAddress, submitter, nonce)
			if err != nil {
				s.log.Panic("could not sign remove signer event, wallet not configured properly",
					logging.Error(err))
			}
			sig = signature.Signature
		}

		s.notary.StartAggregate(resid, types.NodeSignatureKindERC20MultiSigSignerAdded, sig)
		evts = append(evts, events.NewERC20MultiSigSignerAdded(
			ctx,
			eventspb.ERC20MultiSigSignerAdded{
				SignatureId: resid,
				ValidatorId: nodeID,
				Timestamp:   currentTime.UnixNano(),
				EpochSeq:    num.NewUint(pending.EpochSeq).String(),
				NewSigner:   pending.EthAddress,
				Submitter:   submitter,
				Nonce:       nonce.String(),
			},
		))

		// store that we issued it for this submitter
		s.issuedSignatures[resid] = struct{}{}
	}
	s.broker.SendBatch(evts)
	return nil
}

// EmitValidatorRemovedSignatures emit signatures to remove nodeID's ethereum address onto that can be submitter to the contract by submitter.
func (s *ERC20Signatures) EmitValidatorRemovedSignatures(ctx context.Context, submitter, nodeID string, currentTime time.Time) error {
	toEmit := s.getSignatureData(nodeID, false)
	if len(toEmit) == 0 {
		return ErrNoPendingSignaturesForNodeID
	}

	evts := []events.Event{}
	// its possible for a nodeID to need to remove 2 of their etheruem addresses from the contract. For example if they initiate a key rotation
	// and after adding their new key but before they've removed their old key they get demoted. At that point they can have 2 signers on the
	// contract that will need to be removed and that all vaidators could want to issue signatures for
	for _, pending := range toEmit {
		var sig []byte
		nonce := pending.Nonce

		resid := hex.EncodeToString(vgcrypto.Hash([]byte(pending.EthAddress + submitter + nonce.String())))
		if _, ok := s.issuedSignatures[resid]; ok {
			// we've already issued a signature for this submitter we don't want to do it again, it'll annoy the notary engine
			s.log.Debug("remove signatures already issued", logging.String("submitter", submitter), logging.String("add-address", pending.EthAddress))
			continue
		}

		if s.isValidatorSetup {
			signature, err := s.multisig.RemoveSigner(pending.EthAddress, submitter, nonce)
			if err != nil {
				s.log.Panic("could not sign remove signer event, wallet not configured properly",
					logging.Error(err))
			}
			sig = signature.Signature
		}
		s.notary.StartAggregate(
			resid, types.NodeSignatureKindERC20MultiSigSignerRemoved, sig)

		submitters := []*eventspb.ERC20MultiSigSignerRemovedSubmitter{}
		submitters = append(submitters, &eventspb.ERC20MultiSigSignerRemovedSubmitter{
			SignatureId: resid,
			Submitter:   submitter,
		})

		evts = append(evts, events.NewERC20MultiSigSignerRemoved(
			ctx,
			eventspb.ERC20MultiSigSignerRemoved{
				SignatureSubmitters: submitters,
				ValidatorId:         nodeID,
				Timestamp:           currentTime.UnixNano(),
				EpochSeq:            num.NewUint(pending.EpochSeq).String(),
				OldSigner:           pending.EthAddress,
				Nonce:               nonce.String(),
			},
		))

		// store that we issued it for this submitter
		s.issuedSignatures[resid] = struct{}{}
	}
	s.broker.SendBatch(evts)
	return nil
}

// ClearStaleSignatures checks core's view of who is an isn't on the multisig contract and remove any pending signatures that have
// been resolve e.g if a pending sig to add an address X exists but X is on the contract we can remove the pending sig.
func (s *ERC20Signatures) ClearStaleSignatures() {
	toRemove := []string{}
	for e, p := range s.pendingSignatures {
		if p.Added == s.multiSigTopology.IsSigner(e) {
			toRemove = append(toRemove, e)
		}
	}

	for _, e := range toRemove {
		s.log.Debug("removing stale pending signature", logging.String("eth-address", e))
		delete(s.pendingSignatures, e)
	}
}

type noopSignatures struct {
	log *logging.Logger
}

func (n *noopSignatures) EmitValidatorAddedSignatures(_ context.Context, _, _ string, _ time.Time) error {
	n.log.Error("noopSignatures implementation in use in production")
	return nil
}

func (n *noopSignatures) EmitValidatorRemovedSignatures(_ context.Context, _, _ string, _ time.Time) error {
	n.log.Error("noopSignatures implementation in use in production")
	return nil
}

func (n *noopSignatures) PrepareValidatorSignatures(
	_ context.Context, _ []NodeIDAddress, _ uint64, _ bool,
) {
	n.log.Error("noopSignatures implementation in use in production")
}

func (n *noopSignatures) PreparePromotionsSignatures(
	_ context.Context, _ time.Time, _ uint64, _ map[string]StatusAddress, _ map[string]StatusAddress,
) {
	n.log.Error("noopSignatures implementation in use in production")
}

func (n *noopSignatures) SetNonce(_ time.Time) {
	n.log.Error("noopSignatures implementation in use in production")
}

func (n *noopSignatures) ClearStaleSignatures() {
	n.log.Error("noopSignatures implementation in use in production")
}

func (n *noopSignatures) SerialisePendingSignatures() *snapshot.ToplogySignatures {
	n.log.Error("noopSignatures implementation in use in production")
	return &snapshot.ToplogySignatures{}
}

func (n *noopSignatures) RestorePendingSignatures(*snapshot.ToplogySignatures) {
	n.log.Error("noopSignatures implementation in use in production")
}
