package validators

import (
	"context"
	"encoding/hex"
	"sort"
	"time"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/bridges"
	"code.vegaprotocol.io/vega/events"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

type Signatures interface {
	EmitPromotionsSignatures(
		ctx context.Context,
		currentTime time.Time,
		previousState map[string]StatusAddress,
		newState map[string]StatusAddress,
	)
}

type ERC20Signatures struct {
	log       *logging.Logger
	notary    Notary
	multisig  *bridges.ERC20MultiSigControl
	lastNonce *num.Uint
	broker    Broker
}

func NewSignatures(
	log *logging.Logger,
	notary Notary,
	ethSigner Signer,
	broker Broker,
) *ERC20Signatures {
	return &ERC20Signatures{
		log:       log,
		notary:    notary,
		multisig:  bridges.NewERC20MultiSigControl(ethSigner),
		lastNonce: num.Zero(),
		broker:    broker,
	}
}

type StatusAddress struct {
	Status     ValidatorStatus
	EthAddress string
}

type NodeIDAddress struct {
	NodeID     string
	EthAddress string
}

func (s *ERC20Signatures) EmitPromotionsSignatures(
	ctx context.Context,
	currentTime time.Time,
	previousState map[string]StatusAddress,
	newState map[string]StatusAddress,
) {
	toAdd := []NodeIDAddress{}
	toRemove := []NodeIDAddress{}
	allValidators := []NodeIDAddress{}

	// first let's cover all the previous validators
	for k, state := range previousState {
		if val, ok := newState[k]; !ok {
			// in this case we were a validator before, but not even in the validator set anymore,
			// we can remove it.
			if state.Status == ValidatorStatusTendermint {
				toRemove = append(toRemove, NodeIDAddress{k, state.EthAddress})
			}
		} else {
			// we've been removed from the validator set then
			if state.Status == ValidatorStatusTendermint && val.Status != ValidatorStatusTendermint {
				toRemove = append(toRemove, NodeIDAddress{k, state.EthAddress})
			} else if state.Status != ValidatorStatusTendermint && val.Status == ValidatorStatusTendermint {
				// now we've become a validator
				toAdd = append(toAdd, NodeIDAddress{k, state.EthAddress})
			}
		}
	}

	// now let's cover all which might have been added but might not have been in the previousStates?
	// is that even possible?
	for k, val := range newState {
		if val.Status == ValidatorStatusTendermint {
			allValidators = append(allValidators, NodeIDAddress{k, val.EthAddress})
		}
		if _, ok := previousState[k]; !ok {
			// this is a new validator which didn't exist before
			if val.Status == ValidatorStatusTendermint {
				toAdd = append(toAdd, NodeIDAddress{k, val.EthAddress})
			}
		}
	}

	s.lastNonce = num.NewUint(uint64(currentTime.Unix()) + 1)
	s.emitNewValidatorsSignatures(ctx, toAdd, currentTime)
	s.emitRemoveValidatorsSignatures(ctx, toRemove, allValidators, currentTime)
}

func (s *ERC20Signatures) emitNewValidatorsSignatures(
	ctx context.Context,
	validators []NodeIDAddress,
	currentTime time.Time,
) {
	sort.Slice(validators, func(i, j int) bool {
		return validators[i].EthAddress < validators[j].EthAddress
	})
	evts := []events.Event{}

	for _, signer := range validators {
		resid := hex.EncodeToString(
			vgcrypto.Hash([]byte(signer.EthAddress + s.lastNonce.String())))
		signature, err := s.multisig.AddSigner(
			signer.EthAddress,
			signer.EthAddress,
			s.lastNonce,
		)
		if err != nil {
			s.log.Panic("could not sign remove signer event, wallet not configured properly",
				logging.Error(err))
		}

		s.notary.StartAggregate(
			resid, types.NodeSignatureKindERC20MultiSigSignerAdded, signature.Signature)

		evts = append(evts, events.NewERC20MultiSigSignerAdded(
			ctx,
			eventspb.ERC20MultiSigSignerAdded{
				SignatureId: resid,
				ValidatorId: signer.NodeID,
				Timestamp:   currentTime.UnixNano(),
				NewSigner:   signer.EthAddress,
				Submitter:   signer.EthAddress,
				Nonce:       s.lastNonce.String(),
			},
		))

		s.lastNonce.AddUint64(s.lastNonce, 1)
	}

	s.broker.SendBatch(evts)
}

func (s *ERC20Signatures) emitRemoveValidatorsSignatures(
	ctx context.Context,
	remove []NodeIDAddress,
	validators []NodeIDAddress,
	currentTime time.Time,
) {
	sort.Slice(validators, func(i, j int) bool {
		return validators[i].EthAddress < validators[j].EthAddress
	})
	sort.Slice(remove, func(i, j int) bool {
		return remove[i].EthAddress < remove[j].EthAddress
	})
	evts := []events.Event{}

	// for each validators to be removed, we emit a signature
	// so any of the current validators could execute the transaction
	// to remove them
	for _, oldSigner := range remove {
		submitters := []*eventspb.ERC20MulistSigSignerRemovedSubmitter{}
		for _, validator := range validators {
			// Here resid is a concat of the oldsigner, the submitter and the nonce
			resid := hex.EncodeToString(
				vgcrypto.Hash([]byte(oldSigner.EthAddress + validator.EthAddress + s.lastNonce.String())))
			signature, err := s.multisig.RemoveSigner(
				oldSigner.EthAddress, validator.EthAddress, s.lastNonce)
			if err != nil {
				s.log.Panic("could not sign remove signer event, wallet not configured properly",
					logging.Error(err))
			}

			s.notary.StartAggregate(
				resid, types.NodeSignatureKindERC20MultiSigSignerRemoved, signature.Signature)

			submitters = append(submitters, &eventspb.ERC20MulistSigSignerRemovedSubmitter{
				SignatureId: resid,
				Submitter:   validator.EthAddress,
			})
		}
		evts = append(evts, events.NewERC20MultiSigSignerRemoved(
			ctx, eventspb.ERC20MultiSigSignerRemoved{
				SignatureSubmitters: submitters,
				ValidatorId:         oldSigner.NodeID,
				Timestamp:           currentTime.UnixNano(),
				OldSigner:           oldSigner.EthAddress,
				Nonce:               s.lastNonce.String(),
			},
		))

		s.lastNonce.AddUint64(s.lastNonce, 1)
	}

	s.broker.SendBatch(evts)
}

type noopSignatures struct {
	log *logging.Logger
}

func (n *noopSignatures) EmitPromotionsSignatures(
	_ context.Context, _ time.Time, _ map[string]StatusAddress, _ map[string]StatusAddress,
) {
	n.log.Error("noopSignatures implementation in use in production")
}
