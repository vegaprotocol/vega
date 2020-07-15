package banking

import (
	"errors"

	"code.vegaprotocol.io/vega/assets"
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

type assetAction struct {
	id    string
	state uint32
	asset *assets.Asset

	// erc20 specifics
	blockNumber uint64
	txIndex     uint64

	// all deposit related types
	deposit  *deposit
	builtinD *types.BuiltinAssetDeposit
	erc20D   *types.ERC20Deposit
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

func (t *assetAction) BuiltinAssetDesposit() *types.BuiltinAssetDeposit {
	return t.builtinD
}

func (t *assetAction) ERC20Deposit() *types.ERC20Deposit {
	return t.erc20D
}

func (t *assetAction) String() string {
	switch {
	case t.IsBuiltinAssetDeposit():
		return t.builtinD.String()
	case t.IsERC20Deposit():
		return t.erc20D.String()
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
	partyID, assetID, amount, err := asset.ValidateDeposit(t.erc20D, t.blockNumber, t.txIndex)
	if err != nil {
		return err
	}
	t.deposit = &deposit{
		amount:  amount,
		partyID: partyID,
		assetID: assetID,
	}

	return nil
}
