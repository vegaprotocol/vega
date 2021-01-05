package banking

import (
	"errors"
	"math/big"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/assets/common"
	types "code.vegaprotocol.io/vega/proto/gen/golang"

	uuid "github.com/satori/go.uuid"
)

var (
	ErrUnknownAssetAction = errors.New("unknown asset action")
)

type withdrawal struct {
	nonce *big.Int
}

type txRef struct {
	asset    common.AssetClass
	hash     string
	index    uint64
	logIndex uint
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
	builtinD *types.BuiltinAssetDeposit
	erc20D   *types.ERC20Deposit

	// all asset list related types
	withdrawal *withdrawal
	erc20AL    *types.ERC20AssetList

	// all withdrawal related types
	erc20W *types.ERC20Withdrawal
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

func (t *assetAction) IsERC20Withdrawal() bool {
	return t.erc20W != nil
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

func (t *assetAction) ERC20Withdrawal() *types.ERC20Withdrawal {
	return t.erc20W
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
	case t.IsERC20Withdrawal():
		return t.erc20W.String()
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
	case t.IsERC20Withdrawal():
		return t.checkERC20Withdrawal()
	default:
		return ErrUnknownAssetAction
	}
}

func (t *assetAction) checkBuiltinAssetDeposit() error {
	asset, _ := t.asset.BuiltinAsset()
	// builtin deposits do not have hash, and we don't need one
	// so let's just add some random id
	t.ref = txRef{asset.GetAssetClass(), uuid.NewV4().String(), 0, 0}
	return nil
}

func (t *assetAction) checkERC20Deposit() error {
	asset, _ := t.asset.ERC20()
	_, _, hash, _, logIndex, err := asset.ValidateDeposit(t.erc20D, t.blockNumber, t.txIndex)
	if err != nil {
		return err
	}
	t.ref = txRef{asset.GetAssetClass(), hash, t.txIndex, logIndex}
	return nil
}

func (t *assetAction) checkERC20Withdrawal() error {
	asset, _ := t.asset.ERC20()
	nonce, hash, logIndex, err := asset.ValidateWithdrawal(t.erc20W, t.blockNumber, t.txIndex)
	if err != nil {
		return err
	}
	t.withdrawal = &withdrawal{
		nonce: nonce,
	}
	t.ref = txRef{asset.GetAssetClass(), hash, t.txIndex, logIndex}
	return nil
}

func (t *assetAction) checkERC20AssetList() error {
	asset, _ := t.asset.ERC20()
	hash, logIndex, err := asset.ValidateAssetList(t.erc20AL, t.blockNumber, t.txIndex)
	if err != nil {
		return err
	}
	t.ref = txRef{asset.GetAssetClass(), hash, t.txIndex, logIndex}
	return nil
}
