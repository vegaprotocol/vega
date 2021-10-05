package banking

import (
	"errors"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/assets/common"
	"code.vegaprotocol.io/vega/types"
	uuid "github.com/satori/go.uuid"
)

var (
	ErrUnknownAssetAction = errors.New("unknown asset action")
)

type txRef struct {
	asset       common.AssetClass
	blockNumber uint64
	hash        string
	logIndex    uint64
}

type assetAction struct {
	id    string
	state uint32
	asset *assets.Asset

	// erc20 specifics
	blockNumber uint64
	txIndex     uint64
	hash        string

	// all deposit related types
	builtinD *types.BuiltinAssetDeposit
	erc20D   *types.ERC20Deposit

	erc20AL *types.ERC20AssetList
}

func (t *assetAction) GetID() string {
	return t.id
}

func (t *assetAction) IsBuiltinAssetDeposit() bool {
	return t.builtinD != nil
}

func (t *assetAction) IsERC20Deposit() bool {
	return t.erc20D != nil
}

func (t *assetAction) IsERC20AssetList() bool {
	return t.erc20AL != nil
}

func (t *assetAction) BuiltinAssetDesposit() *types.BuiltinAssetDeposit {
	return t.builtinD
}

func (t *assetAction) ERC20Deposit() *types.ERC20Deposit {
	return t.erc20D
}

func (t *assetAction) ERC20AssetList() *types.ERC20AssetList {
	return t.erc20AL
}

func (t *assetAction) String() string {
	switch {
	case t.IsBuiltinAssetDeposit():
		return t.builtinD.String()
	case t.IsERC20Deposit():
		return t.erc20D.String()
	case t.IsERC20AssetList():
		return t.erc20AL.String()
	default:
		return ""
	}
}

func (t *assetAction) Check() error {
	switch {
	case t.IsBuiltinAssetDeposit():
		return t.checkBuiltinAssetDeposit()
	case t.IsERC20Deposit():
		return t.checkERC20Deposit()
	case t.IsERC20AssetList():
		return t.checkERC20AssetList()
	default:
		return ErrUnknownAssetAction
	}
}

func (t *assetAction) checkBuiltinAssetDeposit() error {
	return nil
}

func (t *assetAction) checkERC20Deposit() error {
	asset, _ := t.asset.ERC20()
	return asset.ValidateDeposit(t.erc20D, t.blockNumber, t.txIndex)
}

func (t *assetAction) checkERC20AssetList() error {
	asset, _ := t.asset.ERC20()
	return asset.ValidateAssetList(t.erc20AL, t.blockNumber, t.txIndex)
}

func (t *assetAction) getRef() txRef {
	switch {
	case t.IsBuiltinAssetDeposit():
		return txRef{common.Builtin, 0, uuid.NewV4().String(), 0}
	case t.IsERC20Deposit():
		return txRef{common.ERC20, t.blockNumber, t.hash, t.txIndex}
	case t.IsERC20AssetList():
		return txRef{common.ERC20, t.blockNumber, t.hash, t.txIndex}
	default:
		return txRef{} // this is basically unreachable
	}
}
