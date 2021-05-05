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
	var errs []error
	// first check that no fields are empty
	if len(tx.InputData) <= 0 {
		errs = append(errs, errors.New("tx.input_data is required"))
	}
	if tx.Signature == nil {
		errs = append(errs, errors.New("tx.signature is required"))
	} else {
		errs = append(errs, checkSignature(tx.Signature)...)
	}
	if tx.From == nil {
		errs = append(errs, errors.New("tx.from is required"))
	} else if len(tx.GetPubKey()) <= 0 {
		errs = append(errs, errors.New("tx.from.pub_key is required"))
	}

	// at this point if we have any error we can leave stop processing already
	if len(errs) > 0 {
		return Errors(errs).ErrorOrNil()
	}

	// now we should be able to do signature validation
	if errs := validateSignature(tx.InputData, tx.Signature, tx.GetPubKey()); errs != nil {
		return Errors(errs).ErrorOrNil()
	}

	// now we can unmarshal the transaction, and apply validation
	// on the inputData
	errs = checkInputData(tx.InputData)

	return Errors(errs).ErrorOrNil()
}

func validateSignature(
	inputData []byte,
	signature *commandspb.Signature,
	pubKey []byte,
) []error {
	// build new signature algorithm using the algo from the sig
	validator, err := crypto.NewSignatureAlgorithm(signature.Algo)
	if err != nil {
		return []error{err}
	}
	ok, err := validator.Verify(pubKey, inputData, signature.Bytes)
	if err != nil {
		return []error{err}
	}
	if !ok {
		return []error{ErrInvalidSignature}
	}
	return nil
}

func checkInputData(inputData []byte) []error {
	input := commandspb.InputData{}
	err := proto.Unmarshal(inputData, &input)
	if err != nil {
		return []error{err}
	}

	var errs []error

	if input.Nonce == 0 {
		errs = append(errs, errors.New("input_data.nonce is required to be > 0"))
	}

	if input.Command == nil {
		errs = append(errs, errors.New("input_data.command is required"))
	} else {
		var err error
		switch cmd := input.Command.(type) {
		case *commandspb.InputData_OrderSubmission:
			err = CheckOrderSubmission(cmd.OrderSubmission)
		default:
			err = errors.New("input_data.command is not supported")
		}

		if isErrs, ok := err.(Errors); ok {
			errs = append(errs, isErrs...)
		} else {
			errs = append(errs, err)
		}
	}

	return errs
}

func checkSignature(signature *commandspb.Signature) []error {
	var errs []error
	if len(signature.Bytes) <= 0 {
		errs = append(errs, errors.New("signature.bytes is required"))
	}
	if len(signature.Algo) <= 0 {
		errs = append(errs, errors.New("signature.algo is required"))
	}
	return errs
}
