package erc20

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"code.vegaprotocol.io/vega/assets/common"
	"code.vegaprotocol.io/vega/assets/erc20/bridge"
	"code.vegaprotocol.io/vega/nodewallet"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	MAX_NONCE             = 100000000
	whitelistContractName = "whitelist_asset"
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

func New(id string, asset *types.ERC20, w nodewallet.Wallet) (*ERC20, error) {
	wal, ok := w.(nodewallet.ETHWallet)
	if !ok {
		return nil, ErrMissingETHWalletFromNodeWallet
	}

	return &ERC20{
		asset: &types.Asset{
			ID: id,
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

	b.ok = true
	return nil
}

// SignBridgewhitelisting create and sign the message to
// be sent to the bridge to whitelist the asset
// return the generated message and the signature for this message
func (b *ERC20) SignBridgeWhitelisting() (msg []byte, sig []byte, err error) {
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
	typBytes, err := abi.NewType("bytes", "", nil)
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

	nonce, err := rand.Int(rand.Reader, big.NewInt(MAX_NONCE))
	if err != nil {
		return nil, nil, err
	}
	addr := ethcmn.HexToAddress(b.address)
	buf, err := args.Pack([]interface{}{addr, nonce, whitelistContractName}...)
	if err != nil {
		return nil, nil, err
	}

	bridgeAddr := ethcmn.HexToAddress(b.wallet.BridgeAddress())
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

	msg, err = args2.Pack(buf, bridgeAddr)
	if err != nil {
		return nil, nil, err
	}

	// hash our message before signing it
	hash := crypto.Keccak256(msg)

	// now sign the message using our wallet private key
	sig, err = b.wallet.Sign(hash)
	if err != nil {
		return nil, nil, err
	}

	return msg, sig, nil
}

func (b *ERC20) ValidateWhitelist() error {
	return nil
}

func (b *ERC20) ValidateWithdrawal() error {
	return nil
}

func (b *ERC20) SignWithdrawal() ([]byte, error) {
	return nil, nil
}

func (b *ERC20) ValidateDeposit() error {
	bf, err := bridge.NewBridgeFilterer(
		ethcmn.HexToAddress(b.wallet.BridgeAddress()), b.wallet.Client())
	if err != nil {
		return err
	}

	iter, err := bf.FilterAssetDeposited(
		&bind.FilterOpts{},
		// user_address
		[]ethcmn.Address{},
		[]ethcmn.Address{},

		//[]ethcmn.Address{ethcmn.HexToAddress("0x000000000000000000000000b89a165ea8b619c14312db316baaa80d2a98b493")},
		// asset_source
		//[]ethcmn.Address{ethcmn.HexToAddress("0x000000000000000000000000955c6789a7fbee203b4be0f01428e769308813f2")},
		[]*big.Int{})

	for iter.Next() {
		fmt.Printf("%v - %v - %v\n", iter.Event.Amount, iter.Event.AssetId, iter.Event.AssetSource)
	}

	return nil
}

func (b *ERC20) String() string {
	return fmt.Sprintf("id(%v) name(%v) symbol(%v) totalSupply(%v) decimals(%v)",
		b.asset.ID, b.asset.Name, b.asset.Symbol, b.asset.TotalSupply,
		b.asset.Decimals)
}
