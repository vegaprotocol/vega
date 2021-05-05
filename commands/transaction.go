package commands

import (
	"errors"

	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/wallet/crypto"

	"github.com/golang/protobuf/proto"
)

var (
	ErrInvalidSignature = errors.New("invalid signature")
)

func CheckTransaction(tx *commandspb.Transaction) error {
	errs := NewErrors()
	// first check that no fields are empty
	if len(tx.InputData) == 0 {
		errs.AddForProperty("tx.input_data", ErrIsRequired)
	}
	if tx.Signature == nil {
		errs.AddForProperty("tx.signature", ErrIsRequired)
	} else {
		errs.Merge(checkSignature(tx.Signature))
	}
	if tx.From == nil {
		errs.AddForProperty("tx.from", ErrIsRequired)
	} else if len(tx.GetPubKey()) == 0 {
		errs.AddForProperty("tx.from.pub_key", ErrIsRequired)
	}

	// at this point if we have any error we can leave stop processing already
	if !errs.Empty() {
		return errs.ErrorOrNil()
	}

	// now we should be able to do signature validation
	errs.Merge(validateSignature(tx.InputData, tx.Signature, tx.GetPubKey()))
	if !errs.Empty() {
		return errs.ErrorOrNil()
	}

	// now we can unmarshal the transaction, and apply validation
	// on the inputData
	errs.Merge(checkInputData(tx.InputData))

	return errs.ErrorOrNil()
}

func validateSignature(inputData []byte, signature *commandspb.Signature, pubKey []byte) Errors {
	errs := NewErrors()
	// build new signature algorithm using the algo from the sig
	validator, err := crypto.NewSignatureAlgorithm(signature.Algo)
	if err != nil {
		return errs.FinalAdd(err)
	}
	ok, err := validator.Verify(pubKey, inputData, signature.Bytes)
	if err != nil {
		return errs.FinalAdd(err)
	}
	if !ok {
		return errs.FinalAdd(ErrInvalidSignature)
	}
	return errs
}

func checkInputData(inputData []byte) Errors {
	errs := NewErrors()

	input := commandspb.InputData{}
	err := proto.Unmarshal(inputData, &input)
	if err != nil {
		return errs.FinalAdd(err)
	}

	if input.Nonce == 0 {
		errs.AddForProperty("input_data.nonce", ErrMustBePositive)
	}

	if input.Command == nil {
		errs.AddForProperty("input_data.command", ErrIsRequired)
	} else {
		switch cmd := input.Command.(type) {
		case *commandspb.InputData_OrderSubmission:
			errs.Merge(checkOrderSubmission(cmd.OrderSubmission))
		case *commandspb.InputData_OrderAmendment:
			errs.Merge(checkOrderAmendment(cmd.OrderAmendment))
		case *commandspb.InputData_VoteSubmission:
			errs.Merge(checkVoteSubmission(cmd.VoteSubmission))
		case *commandspb.InputData_WithdrawSubmission:
			errs.Merge(checkWithdrawSubmission(cmd.WithdrawSubmission))
		case *commandspb.InputData_LiquidityProvisionSubmission:
			errs.Merge(checkLiquidityProvisionSubmission(
				cmd.LiquidityProvisionSubmission))
		default:
			errs.AddForProperty("input_data.command", ErrIsNotSupported)
		}
	}

	return errs
}

func checkSignature(signature *commandspb.Signature) Errors {
	errs := NewErrors()
	if len(signature.Bytes) == 0 {
		errs.AddForProperty("signature.bytes", ErrIsRequired)
	}
	if len(signature.Algo) == 0 {
		errs.AddForProperty("signature.algo", ErrIsRequired)
	}
	return errs
}
