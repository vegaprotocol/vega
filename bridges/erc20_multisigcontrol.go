package bridges

import (
	"code.vegaprotocol.io/vega/types/num"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcmn "github.com/ethereum/go-ethereum/common"
)

type ERC20MultiSigControl struct {
	signer Signer
}

func NewERC20MultiSigControl(signer Signer) *ERC20MultiSigControl {
	return &ERC20MultiSigControl{
		signer: signer,
	}
}

func (e *ERC20MultiSigControl) SetThreshold(
	newThreshold uint16,
	submitter string,
	nonce *num.Uint,
) (*SignaturePayload, error) {
	typString, err := abi.NewType("string", "", nil)
	if err != nil {
		return nil, err
	}
	typU16, err := abi.NewType("uint16", "", nil)
	if err != nil {
		return nil, err
	}
	typU256, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}

	args := abi.Arguments([]abi.Argument{
		{
			Name: "new_threshold",
			Type: typU16,
		},
		{
			Name: "nonce",
			Type: typU256,
		},
		{
			Name: "func_name",
			Type: typString,
		},
	})

	buf, err := args.Pack([]interface{}{newThreshold, nonce.BigInt(), "set_threshold"}...)
	if err != nil {
		return nil, err
	}

	msg, err := packBufAndSubmitter(buf, submitter)
	if err != nil {
		return nil, err
	}

	return sign(e.signer, msg)
}

func (e *ERC20MultiSigControl) AddSigner(
	newSigner, submitter string,
	nonce *num.Uint,
) (*SignaturePayload, error) {
	typAddr, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, err
	}
	typString, err := abi.NewType("string", "", nil)
	if err != nil {
		return nil, err
	}
	typU256, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}

	args := abi.Arguments([]abi.Argument{
		{
			Name: "new_signer",
			Type: typAddr,
		},
		{
			Name: "nonce",
			Type: typU256,
		},
		{
			Name: "func_name",
			Type: typString,
		},
	})

	newSignerAddr := ethcmn.HexToAddress(newSigner)
	buf, err := args.Pack([]interface{}{newSignerAddr, nonce.BigInt(), "add_signer"}...)
	if err != nil {
		return nil, err
	}

	msg, err := packBufAndSubmitter(buf, submitter)
	if err != nil {
		return nil, err
	}

	return sign(e.signer, msg)
}

func (e *ERC20MultiSigControl) RemoveSigner(
	oldSigner, submitter string,
	nonce *num.Uint,
) (*SignaturePayload, error) {
	typAddr, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, err
	}
	typString, err := abi.NewType("string", "", nil)
	if err != nil {
		return nil, err
	}
	typU256, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}

	args := abi.Arguments([]abi.Argument{
		{
			Name: "old_signer",
			Type: typAddr,
		},
		{
			Name: "nonce",
			Type: typU256,
		},
		{
			Name: "func_name",
			Type: typString,
		},
	})

	oldSignerAddr := ethcmn.HexToAddress(oldSigner)
	buf, err := args.Pack([]interface{}{oldSignerAddr, nonce.BigInt(), "remove_signer"}...)
	if err != nil {
		return nil, err
	}

	msg, err := packBufAndSubmitter(buf, submitter)
	if err != nil {
		return nil, err
	}

	return sign(e.signer, msg)
}
