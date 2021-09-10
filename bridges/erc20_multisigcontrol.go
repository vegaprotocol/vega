package bridges

import (
	"code.vegaprotocol.io/vega/types/num"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type Signer interface {
	Sign([]byte) ([]byte, error)
}

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
) (msg []byte, sig []byte, err error) {
	typString, err := abi.NewType("string", "", nil)
	if err != nil {
		return nil, nil, err
	}
	typU16, err := abi.NewType("uint16", "", nil)
	if err != nil {
		return nil, nil, err
	}
	typU256, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}

	msg, err = e.packBufAndSubmitter(buf, submitter)
	if err != nil {
		return nil, nil, err
	}

	sig, err = e.sign(msg)
	if err != nil {
		return nil, nil, err
	}

	return msg, sig, nil
}

func (e *ERC20MultiSigControl) AddSigner(
	newSigner, submitter string,
	nonce *num.Uint,
) (msg []byte, sig []byte, err error) {
	typAddr, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, nil, err
	}
	typString, err := abi.NewType("string", "", nil)
	if err != nil {
		return nil, nil, err
	}
	typU256, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, nil, err
	}

	args := abi.Arguments([]abi.Argument{
		{
			Name: "address",
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
		return nil, nil, err
	}

	msg, err = e.packBufAndSubmitter(buf, submitter)
	if err != nil {
		return nil, nil, err
	}

	sig, err = e.sign(msg)
	if err != nil {
		return nil, nil, err
	}

	return msg, sig, nil
}

func (e *ERC20MultiSigControl) RemoveSigner(
	oldSigner, submitter string,
	nonce *num.Uint,
) (msg []byte, sig []byte, err error) {
	typAddr, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, nil, err
	}
	typString, err := abi.NewType("string", "", nil)
	if err != nil {
		return nil, nil, err
	}
	typU256, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, nil, err
	}

	args := abi.Arguments([]abi.Argument{
		{
			Name: "address",
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
		return nil, nil, err
	}

	msg, err = e.packBufAndSubmitter(buf, submitter)
	if err != nil {
		return nil, nil, err
	}

	sig, err = e.sign(msg)
	if err != nil {
		return nil, nil, err
	}

	return msg, sig, nil
}

func (e *ERC20MultiSigControl) packBufAndSubmitter(
	buf []byte, submitter string,
) ([]byte, error) {
	typBytes, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}
	typAddr, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, err
	}

	submitterAddr := ethcmn.HexToAddress(submitter)
	args2 := abi.Arguments([]abi.Argument{
		{
			Name: "bytes",
			Type: typBytes,
		},
		{
			Name: "address",
			Type: typAddr,
		},
	})

	return args2.Pack(buf, submitterAddr)
}

func (e *ERC20MultiSigControl) sign(msg []byte) ([]byte, error) {
	// hash our message before signing it
	hash := crypto.Keccak256(msg)
	return e.signer.Sign(hash)
}
