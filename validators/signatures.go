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
		previousState map[string]statusAddress,
		newState map[string]statusAddress,
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

type statusAddress struct {
	status     ValidatorStatus
	ethAddress string
}

type nodeIDAddress struct {
	nodeID     string
	ethAddress string
}

func (s *ERC20Signatures) EmitPromotionsSignatures(
	ctx context.Context,
	currentTime time.Time,
	previousState map[string]statusAddress,
	newState map[string]statusAddress,
) {
	toAdd := []nodeIDAddress{}
	toRemove := []nodeIDAddress{}
	allValidators := []nodeIDAddress{}

	// first let's cover all the previous validators
	for k, state := range previousState {
		if val, ok := newState[k]; !ok {
			// in this case we were a validator before, but not even in the validator set anymore,
			// we can remove it.
			if state.status == ValidatorStatusTendermint {
				toRemove = append(toRemove, nodeIDAddress{k, state.ethAddress})
			}
		} else {
			// we've been removed from the validator set then
			if state.status == ValidatorStatusTendermint && val.status != ValidatorStatusTendermint {
				toRemove = append(toRemove, nodeIDAddress{k, state.ethAddress})
			} else if state.status != ValidatorStatusTendermint && val.status == ValidatorStatusTendermint {
				// now we've become a validator
				toAdd = append(toAdd, nodeIDAddress{k, state.ethAddress})
			}
		}
	}

	// now let's cover all which might have been added but might not have been in the previousStates?
	// is that even possible?
	for k, val := range newState {
		if val.status == ValidatorStatusTendermint {
			allValidators = append(allValidators, nodeIDAddress{k, val.ethAddress})
		}
		if _, ok := previousState[k]; !ok {
			// this is a new validator which didn't exist before
			if val.status == ValidatorStatusTendermint {
				toAdd = append(toAdd, nodeIDAddress{k, val.ethAddress})
			}
		}
	}

	s.lastNonce = num.NewUint(uint64(currentTime.Unix()) + 1)
	s.emitNewValidatorsSignatures(ctx, toAdd, currentTime)
	s.emitRemoveValidatorsSignatures(ctx, toRemove, allValidators, currentTime)
}

func (s *ERC20Signatures) emitNewValidatorsSignatures(
	ctx context.Context,
	validators []nodeIDAddress,
	currentTime time.Time,
) {
	sort.Slice(validators, func(i, j int) bool {
		return validators[i].ethAddress < validators[j].ethAddress
	})
	evts := []events.Event{}

	for _, signer := range validators {
		resid := hex.EncodeToString(
			vgcrypto.Hash([]byte(signer.ethAddress + s.lastNonce.String())))
		signature, err := s.multisig.AddSigner(
			signer.ethAddress,
			signer.ethAddress,
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
				ValidatorId: signer.nodeID,
				Timestamp:   currentTime.UnixNano(),
				NewSigner:   signer.ethAddress,
				Submitter:   signer.ethAddress,
				Nonce:       s.lastNonce.String(),
			},
		))

		s.lastNonce.AddUint64(s.lastNonce, 1)
	}

	s.broker.SendBatch(evts)
}

func (s *ERC20Signatures) emitRemoveValidatorsSignatures(
	ctx context.Context,
	remove []nodeIDAddress,
	validators []nodeIDAddress,
	currentTime time.Time,
) {
	sort.Slice(validators, func(i, j int) bool {
		return validators[i].ethAddress < validators[j].ethAddress
	})
	sort.Slice(remove, func(i, j int) bool {
		return remove[i].ethAddress < remove[j].ethAddress
	})
	evts := []events.Event{}

	// for each validators to be removed, we emit a signature
	// so any of the current validators could execute the transaction
	// to remove them
	for _, oldSigner := range remove {
		resid := hex.EncodeToString(
			vgcrypto.Hash([]byte(oldSigner.ethAddress + s.lastNonce.String())))
		for _, validator := range validators {
			signature, err := s.multisig.RemoveSigner(
				oldSigner.ethAddress, validator.ethAddress, s.lastNonce)
			if err != nil {
				s.log.Panic("could not sign remove signer event, wallet not configured properly",
					logging.Error(err))
			}

			s.notary.StartAggregate(
				resid, types.NodeSignatureKindERC20MultiSigSignerRemoved, signature.Signature)

			evts = append(evts, events.NewERC20MultiSigSignerAdded(
				ctx,
				eventspb.ERC20MultiSigSignerAdded{
					SignatureId: resid,
					ValidatorId: oldSigner.nodeID,
					Timestamp:   currentTime.UnixNano(),
					NewSigner:   oldSigner.ethAddress,
					Submitter:   validator.ethAddress,
					Nonce:       s.lastNonce.String(),
				},
			))
		}

		s.lastNonce.AddUint64(s.lastNonce, 1)
	}

	s.broker.SendBatch(evts)
}

type noopSignatures struct {
	log *logging.Logger
}

func (n *noopSignatures) EmitPromotionsSignatures(
	_ context.Context, _ time.Time, _ map[string]statusAddress, _ map[string]statusAddress,
) {
	n.log.Error("noopSignatures implementation in use in production")
}
