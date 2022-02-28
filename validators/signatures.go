package validators

import (
	"encoding/hex"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/bridges"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

type Signatures interface {
	EmitPromotionsSignatures(
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
}

func NewSignatures(log *logging.Logger, notary Notary, ethSigner Signer) *ERC20Signatures {
	return &ERC20Signatures{
		log:       log,
		notary:    notary,
		multisig:  bridges.NewERC20MultiSigControl(ethSigner),
		lastNonce: num.Zero(),
	}
}

type statusAddress struct {
	status     ValidatorStatus
	ethAddress string
}

func (s *ERC20Signatures) EmitPromotionsSignatures(
	currentTime time.Time,
	previousState map[string]statusAddress,
	newState map[string]statusAddress,
) {
	toAdd := []string{}
	toRemove := []string{}
	allValidators := []string{}

	// first let's cover all the previous validators
	for k, state := range previousState {
		if val, ok := newState[k]; !ok {
			// in this case we were a validator before, but not even in the validator set anymore,
			// we can remove it.
			if state.status == ValidatorStatusTendermint {
				toRemove = append(toRemove, state.ethAddress)
			}
		} else {
			// we've been removed from the validator set then
			if state.status == ValidatorStatusTendermint && val.status != ValidatorStatusTendermint {
				toRemove = append(toRemove, state.ethAddress)
			} else if state.status != ValidatorStatusTendermint && val.status == ValidatorStatusTendermint {
				// now we've become a validator
				toAdd = append(toAdd, state.ethAddress)
			}
		}
	}

	// now let's cover all which might have been added but might not have been in the previousStates?
	// is that even possible?
	for k, val := range newState {
		if val.status == ValidatorStatusTendermint {
			allValidators = append(allValidators, val.ethAddress)
		}
		if _, ok := previousState[k]; !ok {
			// this is a new validator which didn't exist before
			if val.status == ValidatorStatusTendermint {
				toAdd = append(toAdd, val.ethAddress)
			}
		}
	}

	s.lastNonce = num.NewUint(uint64(currentTime.Unix()) + 1)
	s.emitNewValidatorsSignatures(toAdd)
	s.emitRemoveValidatorsSignatures(toRemove, allValidators)
}

func (s *ERC20Signatures) emitNewValidatorsSignatures(validators []string) {
	sort.Strings(validators)

	for _, signer := range validators {
		resid := hex.EncodeToString(
			vgcrypto.Hash([]byte(signer + s.lastNonce.String())))
		signature, err := s.multisig.AddSigner(
			signer,
			signer,
			s.lastNonce,
		)
		if err != nil {
			s.log.Panic("could not sign remove signer event, wallet not configured properly",
				logging.Error(err))
		}

		s.notary.StartAggregate(
			resid, types.NodeSignatureKindERC20MultiSigSignerAdded, signature.Signature)

		s.lastNonce.AddUint64(s.lastNonce, 1)
	}
}

func (s *ERC20Signatures) emitRemoveValidatorsSignatures(
	remove []string,
	validators []string,
) {
	sort.Strings(remove)
	sort.Strings(validators)

	// for each validators to be removed, we emit a signature
	// so any of the current validators could execute the transaction
	// to remove them
	for _, oldSigner := range remove {
		resid := hex.EncodeToString(
			vgcrypto.Hash([]byte(oldSigner + s.lastNonce.String())))
		for _, validator := range validators {
			signature, err := s.multisig.RemoveSigner(oldSigner, validator, s.lastNonce)
			if err != nil {
				s.log.Panic("could not sign remove signer event, wallet not configured properly",
					logging.Error(err))
			}

			s.notary.StartAggregate(
				resid, types.NodeSignatureKindERC20MultiSigSignerRemoved, signature.Signature)
		}

		s.lastNonce.AddUint64(s.lastNonce, 1)
	}
}

type noopSignatures struct {
	log *logging.Logger
}

func (n *noopSignatures) EmitPromotionsSignatures(
	_ time.Time, _ map[string]statusAddress, _ map[string]statusAddress,
) {
	n.log.Error("noopSignatures implementation in use in production")
}
