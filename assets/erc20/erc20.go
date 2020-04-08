package erc20

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/assets/common"
	"code.vegaprotocol.io/vega/nodewallet"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcmn "github.com/ethereum/go-ethereum/common"
)

var (
	ErrMissingETHWalletFromNodeWallet = errors.New("missing eth wallet from node wallet")
)

type ERC20 struct {
	asset   *types.Asset
	address string
	ok      bool
	wallet  nodewallet.ETHWallet
}

func New(id uint64, asset *types.ERC20, w nodewallet.Wallet) (*ERC20, error) {
	wal, ok := w.(nodewallet.ETHWallet)
	if !ok {
		return nil, ErrMissingETHWalletFromNodeWallet
	}

	return &ERC20{
		asset: &types.Asset{
			Id: id,
			Source: &types.Asset_Erc20{
				Erc20: asset,
			},
		},
		address: asset.ContractAddress,
		wallet:  wal,
	}, nil
}

func (b *ERC20) Data() *types.Asset {
	return b.asset
}

func (b *ERC20) GetAssetClass() common.AssetClass {
	return common.ERC20
}

func (b *ERC20) IsValid() bool {
	return b.ok
}

func (b *ERC20) Validate() error {
	t, err := NewToken(ethcmn.HexToAddress(b.address), b.wallet.Client())
	if err != nil {
		return err
	}

	name, err := t.Name(&bind.CallOpts{})
	if err != nil {
		return err
	}

	symbol, err := t.Symbol(&bind.CallOpts{})
	if err != nil {
		return err
	}

	decimals, err := t.Decimals(&bind.CallOpts{})
	if err != nil {
		return err
	}

	totalSupply, err := t.TotalSupply(&bind.CallOpts{})
	if err != nil {
		return err
	}

	// non of the checks failed,
	// with got all data needed, lets' update the struct
	// and make this asset valid
	b.asset.Name = name
	b.asset.Symbol = symbol
	b.asset.Decimals = uint64(decimals)
	b.asset.TotalSupply = totalSupply.String()

	fmt.Printf("OK\n")
	b.ok = true
	return nil
}

func (b *ERC20) SignBridgeWhitelisting() ([]byte, error) {
	return nil, nil
}

func (b *ERC20) ValidateWithdrawal() error {
	return nil
}

func (b *ERC20) SignWithdrawal() ([]byte, error) {
	return nil, nil
}

func (b *ERC20) ValidateDeposit() error {
	return nil
}

func (b *ERC20) String() string {
	return fmt.Sprintf("id(%v) name(%v) symbol(%v) totalSupply(%v) decimals(%v)",
		b.asset.Id, b.asset.Name, b.asset.Symbol, b.asset.TotalSupply,
		b.asset.Decimals)
}
