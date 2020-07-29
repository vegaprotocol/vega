package banking

import (
	"errors"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/assets/common"
	types "code.vegaprotocol.io/vega/proto"
)

var (
	ErrUnknownAssetAction = errors.New("unknown asset action")
)

type deposit struct {
	amount  uint64
	assetID string
	partyID string
}

type txRef struct {
	asset common.AssetClass
	hash  string
}

type assetAction struct {
	id    string
	state uint32
	asset *assets.Asset

	// hash of transaction used to ensure a transaction has not been
	// processed twice
	ref txRef

	// erc20 specifics
	blockNumber uint64
	txIndex     uint64

	// all deposit related types
	deposit  *deposit
	builtinD *types.BuiltinAssetDeposit
	erc20D   *types.ERC20Deposit

	// all asset list related types
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
	t.deposit = &deposit{
		amount:  t.builtinD.Amount,
		partyID: t.builtinD.PartyID,
		assetID: t.builtinD.VegaAssetID,
	}
	return nil
}

func (t *assetAction) checkERC20Deposit() error {
	asset, _ := t.asset.ERC20()
	partyID, assetID, hash, amount, err := asset.ValidateDeposit(t.erc20D, t.blockNumber, t.txIndex)
	if err != nil {
		return err
	}
	t.deposit = &deposit{
		amount:  amount,
		partyID: partyID,
		assetID: assetID,
	}
	t.ref = txRef{asset.GetAssetClass(), hash}
	return nil
}

func (t *assetAction) checkERC20AssetList() error {
	asset, _ := t.asset.ERC20()
	hash, err := asset.ValidateWhitelist(t.erc20AL, t.blockNumber, t.txIndex)
	if err != nil {
		return err
	}
	t.ref = txRef{asset.GetAssetClass(), hash}
	return nil
}
