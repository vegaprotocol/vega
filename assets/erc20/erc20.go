package erc20

import (
	"crypto/rand"
	"encoding/hex"
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
	ErrMissingETHWalletFromNodeWallet  = errors.New("missing eth wallet from node wallet")
	ErrUnableToFindDeposit             = errors.New("unable to find erc20 deposit event")
	ErrUnableToFindERC20AssetWhitelist = errors.New("unable to find erc20 asset whitelist event")
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

func (b *ERC20) ProtoAsset() *types.Asset {
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

func (b *ERC20) ValidateWhitelist(w *types.ERC20AssetList, blockNumber, txIndex uint64) error {
	bf, err := bridge.NewBridgeFilterer(
		ethcmn.HexToAddress(b.wallet.BridgeAddress()), b.wallet.Client())
	if err != nil {
		return err
	}

	iter, err := bf.FilterAssetWhitelisted(
		&bind.FilterOpts{
			Start: blockNumber - 1,
		},
		[]ethcmn.Address{ethcmn.HexToAddress(b.address)},
		[]*big.Int{},
		[][32]byte{},
	)

	if err != nil {
		return err
	}

	defer iter.Close()
	var event *bridge.BridgeAssetWhitelisted
	for iter.Next() {
		if hex.EncodeToString(iter.Event.VegaId[:]) == w.VegaAssetID {
			event = iter.Event
			break
		}
	}

	if event == nil {
		return ErrUnableToFindERC20AssetWhitelist
	}

	return nil
}

func (b *ERC20) SignWithdrawal() ([]byte, error) {
	return nil, nil
}

func (b *ERC20) ValidateWithdrawal() error {
	return nil
}

func (b *ERC20) ValidateDeposit(d *types.ERC20Deposit, blockNumber, txIndex uint64) (partyID, assetID string, amount uint64, err error) {
	bf, err := bridge.NewBridgeFilterer(
		ethcmn.HexToAddress(b.wallet.BridgeAddress()), b.wallet.Client())
	if err != nil {
		return "", "", 0, err
	}

	iter, err := bf.FilterAssetDeposited(
		&bind.FilterOpts{
			Start: blockNumber - 1,
		},
		// user_address
		[]ethcmn.Address{ethcmn.HexToAddress(d.SourceEthereumAddress)},
		// asset_source
		[]ethcmn.Address{ethcmn.HexToAddress(b.address)},
		[]*big.Int{})

	if err != nil {
		return "", "", 0, err
	}

	defer iter.Close()
	var event *bridge.BridgeAssetDeposited
	for iter.Next() {
		if hex.EncodeToString(iter.Event.VegaPublicKey[:]) == d.TargetPartyID &&
			iter.Event.Raw.BlockNumber == blockNumber &&
			uint64(iter.Event.Raw.TxIndex) == txIndex {
			event = iter.Event
			break
		}
	}

	if event == nil {
		return "", "", 0, ErrUnableToFindDeposit
	}

	return d.TargetPartyID, d.VegaAssetID, iter.Event.Amount.Uint64(), nil
}

func (b *ERC20) String() string {
	return fmt.Sprintf("id(%v) name(%v) symbol(%v) totalSupply(%v) decimals(%v)",
		b.asset.ID, b.asset.Name, b.asset.Symbol, b.asset.TotalSupply,
		b.asset.Decimals)
}
