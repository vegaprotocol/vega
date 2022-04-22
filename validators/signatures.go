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

//go:generate go run github.com/golang/mock/mockgen -destination mocks/signatures_mock.go -package mocks code.vegaprotocol.io/vega/validators Signatures
type Signatures interface {
	EmitPromotionsSignatures(
		ctx context.Context,
		currentTime time.Time,
		epochSeq uint64,
		previousState map[string]StatusAddress,
		newState map[string]StatusAddress,
	)
	EmitNewValidatorsSignatures(
		ctx context.Context,
		validators []NodeIDAddress,
		currentTime time.Time,
		epochSeq uint64,
	)
	EmitRemoveValidatorsSignatures(
		ctx context.Context,
		remove []NodeIDAddress,
		validators []NodeIDAddress,
		currentTime time.Time,
		epochSeq uint64,
	)
	SetNonce(t time.Time)
}

type ERC20Signatures struct {
	log              *logging.Logger
	notary           Notary
	multisig         *bridges.ERC20MultiSigControl
	lastNonce        *num.Uint
	broker           Broker
	isValidatorSetup bool
}

func NewSignatures(
	log *logging.Logger,
	notary Notary,
	nw NodeWallets,
	broker Broker,
	isValidatorSetup bool,
) *ERC20Signatures {
	s := &ERC20Signatures{
		log:              log,
		notary:           notary,
		lastNonce:        num.Zero(),
		broker:           broker,
		isValidatorSetup: isValidatorSetup,
	}
	if isValidatorSetup {
		s.multisig = bridges.NewERC20MultiSigControl(nw.GetEthereum())
	}
	return s
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
	epochSeq uint64,
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

	s.SetNonce(currentTime)
	s.EmitNewValidatorsSignatures(ctx, toAdd, currentTime, epochSeq)
	s.EmitRemoveValidatorsSignatures(ctx, toRemove, allValidators, currentTime, epochSeq)
}

func (s *ERC20Signatures) SetNonce(t time.Time) {
	s.lastNonce = num.NewUint(uint64(t.Unix()) + 1)
}

func (s *ERC20Signatures) EmitNewValidatorsSignatures(
	ctx context.Context,
	validators []NodeIDAddress,
	currentTime time.Time,
	epochSeq uint64,
) {
	sort.Slice(validators, func(i, j int) bool {
		return validators[i].EthAddress < validators[j].EthAddress
	})
	evts := []events.Event{}

	for _, signer := range validators {
		var sig []byte

		resid := hex.EncodeToString(
			vgcrypto.Hash([]byte(signer.EthAddress + s.lastNonce.String())))

		if s.isValidatorSetup {
			signature, err := s.multisig.AddSigner(
				signer.EthAddress,
				signer.EthAddress,
				s.lastNonce,
			)
			if err != nil {
				s.log.Panic("could not sign remove signer event, wallet not configured properly",
					logging.Error(err))
			}
			sig = signature.Signature
		}

		s.notary.StartAggregate(
			resid, types.NodeSignatureKindERC20MultiSigSignerAdded, sig)

		evts = append(evts, events.NewERC20MultiSigSignerAdded(
			ctx,
			eventspb.ERC20MultiSigSignerAdded{
				SignatureId: resid,
				ValidatorId: signer.NodeID,
				Timestamp:   currentTime.UnixNano(),
				EpochSeq:    num.NewUint(epochSeq).String(),
				NewSigner:   signer.EthAddress,
				Submitter:   signer.EthAddress,
				Nonce:       s.lastNonce.String(),
			},
		))

		s.lastNonce.AddUint64(s.lastNonce, 1)
	}

	s.broker.SendBatch(evts)
}

func (s *ERC20Signatures) EmitRemoveValidatorsSignatures(
	ctx context.Context,
	remove []NodeIDAddress,
	validators []NodeIDAddress,
	currentTime time.Time,
	epochSeq uint64,
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
			var sig []byte
			// Here resid is a concat of the oldsigner, the submitter and the nonce
			resid := hex.EncodeToString(
				vgcrypto.Hash([]byte(oldSigner.EthAddress + validator.EthAddress + s.lastNonce.String())))

			if s.isValidatorSetup {
				signature, err := s.multisig.RemoveSigner(
					oldSigner.EthAddress, validator.EthAddress, s.lastNonce)
				if err != nil {
					s.log.Panic("could not sign remove signer event, wallet not configured properly",
						logging.Error(err))
				}
				sig = signature.Signature
			}
			s.notary.StartAggregate(
				resid, types.NodeSignatureKindERC20MultiSigSignerRemoved, sig)

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
				EpochSeq:            num.NewUint(epochSeq).String(),
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
	_ context.Context, _ time.Time, _ uint64, _ map[string]StatusAddress, _ map[string]StatusAddress,
) {
	n.log.Error("noopSignatures implementation in use in production")
}

func (n *noopSignatures) EmitNewValidatorsSignatures(
	_ context.Context, _ []NodeIDAddress, _ time.Time, _ uint64,
) {
	n.log.Error("noopSignatures implementation in use in production")
}

func (n *noopSignatures) EmitRemoveValidatorsSignatures(
	_ context.Context, _ []NodeIDAddress, _ []NodeIDAddress, _ time.Time, _ uint64,
) {
	n.log.Error("noopSignatures implementation in use in production")
}

func (n *noopSignatures) SetNonce(_ time.Time) {
	n.log.Error("noopSignatures implementation in use in production")
}
