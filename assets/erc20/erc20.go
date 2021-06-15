package erc20

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"

	"code.vegaprotocol.io/vega/assets/common"
	"code.vegaprotocol.io/vega/assets/erc20/bridge"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	MaxNonce              = 100000000
	listAssetContractName = "list_asset"
	withdrawContractName  = "withdraw_asset"
)

var (
	ErrMissingETHWalletFromNodeWallet = errors.New("missing eth wallet from node wallet")
	ErrUnableToFindDeposit            = errors.New("unable to find erc20 deposit event")
	ErrUnableToFindWithdrawal         = errors.New("unable to find erc20 withdrawal event")
	ErrUnableToFindERC20AssetList     = errors.New("unable to find erc20 asset list event")
	ErrMissingConfirmations           = errors.New("missing confirmation from ethereum")
)

type ERC20 struct {
	asset   *proto.Asset
	address string
	ok      bool
	wallet  nodewallet.ETHWallet
}

func New(id string, asset *proto.ERC20, w nodewallet.Wallet) (*ERC20, error) {
	wal, ok := w.(nodewallet.ETHWallet)
	if !ok {
		return nil, ErrMissingETHWalletFromNodeWallet
	}

	return &ERC20{
		asset: &proto.Asset{
			Id: id,
			Source: &proto.AssetSource{
				Source: &proto.AssetSource_Erc20{
					Erc20: asset,
				},
			},
		},
		address: asset.ContractAddress,
		wallet:  wal,
	}, nil
}

func (b *ERC20) ProtoAsset() *proto.Asset {
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

// SignBridgeListing create and sign the message to
// be sent to the bridge to whitelist the asset
// return the generated message and the signature for this message
func (b *ERC20) SignBridgeListing() (msg []byte, sig []byte, err error) {
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
			Name: "vega_asset_id",
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

	nonce, err := rand.Int(rand.Reader, big.NewInt(MaxNonce))
	if err != nil {
		return nil, nil, err
	}
	addr := ethcmn.HexToAddress(b.address)
	vegaAssetIDBytes, _ := hex.DecodeString(b.asset.Id)
	buf, err := args.Pack([]interface{}{addr, vegaAssetIDBytes, nonce, listAssetContractName}...)
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

func (b *ERC20) ValidateAssetList(w *proto.ERC20AssetList, blockNumber, txIndex uint64) (hash string, logIndex uint, err error) {
	bf, err := bridge.NewBridgeFilterer(
		ethcmn.HexToAddress(b.wallet.BridgeAddress()), b.wallet.Client())
	if err != nil {
		return "", 0, err
	}

	var resp = "ok"
	defer func() {
		metrics.EthCallInc("validate_allowlist", b.asset.Id, resp)
	}()

	iter, err := bf.FilterAssetListed(
		&bind.FilterOpts{
			Start: blockNumber - 1,
		},
		[]ethcmn.Address{ethcmn.HexToAddress(b.address)},
		[][32]byte{},
	)

	if err != nil {
		resp = getMaybeHTTPStatus(err)
		return "", 0, err
	}

	defer iter.Close()
	var event *bridge.BridgeAssetListed
	for iter.Next() {
		if hex.EncodeToString(iter.Event.VegaAssetId[:]) == w.VegaAssetId {
			event = iter.Event
			break
		}
	}

	if event == nil {
		return "", 0, ErrUnableToFindERC20AssetList
	}

	// now ensure we have enough confirmations
	if err := b.checkConfirmations(event.Raw.BlockNumber); err != nil {
		return "", 0, err
	}

	return event.Raw.TxHash.Hex(), event.Raw.Index, nil
}

func (b *ERC20) SignWithdrawal(
	amount *num.Uint,
	expirationDate int64,
	ethPartyAddress string,
	withdrawRef *big.Int,
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
			Name: "uint256",
			Type: typU256,
		},
		{
			Name: "uint256",
			Type: typU256,
		},
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

	addr := ethcmn.HexToAddress(b.address)
	hexEthPartyAddress := ethcmn.HexToAddress(ethPartyAddress)

	// we use the withdrawRef as a nonce
	// they are unique as generated as an increment from the banking
	// layer
	buf, err := args.Pack([]interface{}{addr, amount, big.NewInt(expirationDate), hexEthPartyAddress, withdrawRef, withdrawContractName}...)
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

func (b *ERC20) ValidateWithdrawal(w *proto.ERC20Withdrawal, blockNumber, txIndex uint64) (*big.Int, string, uint, error) {
	bf, err := bridge.NewBridgeFilterer(
		ethcmn.HexToAddress(b.wallet.BridgeAddress()), b.wallet.Client())
	if err != nil {
		return nil, "", 0, err
	}

	var resp = "ok"
	defer func() {
		metrics.EthCallInc("validate_withdrawal", b.asset.Id, resp)
	}()

	iter, err := bf.FilterAssetWithdrawn(
		&bind.FilterOpts{
			Start: blockNumber - 1,
		},
		// user_address
		[]ethcmn.Address{ethcmn.HexToAddress(w.TargetEthereumAddress)},
		// asset_source
		[]ethcmn.Address{ethcmn.HexToAddress(b.address)})

	if err != nil {
		resp = getMaybeHTTPStatus(err)
		return nil, "", 0, err
	}

	defer iter.Close()
	var event *bridge.BridgeAssetWithdrawn
	nonce := &big.Int{}
	nonce.SetString(w.ReferenceNonce, 10)
	for iter.Next() {

		// here the event queu send us a 0x... pubkey
		// we do the slice operation to remove it ([2:]
		if nonce.Cmp(iter.Event.Nonce) == 0 &&
			iter.Event.Raw.BlockNumber == blockNumber &&
			uint64(iter.Event.Raw.Index) == txIndex {
			event = iter.Event
			break
		}
	}

	if event == nil {
		return nil, "", 0, ErrUnableToFindWithdrawal
	}

	// now ensure we have enough confirmations
	if err := b.checkConfirmations(event.Raw.BlockNumber); err != nil {
		return nil, "", 0, err
	}

	return nonce, event.Raw.TxHash.Hex(), event.Raw.Index, nil
}

func (b *ERC20) ValidateDeposit(d *types.ERC20Deposit, blockNumber, txIndex uint64) (partyID, assetID, hash string, amount uint64, logIndex uint, err error) {
	bf, err := bridge.NewBridgeFilterer(
		ethcmn.HexToAddress(b.wallet.BridgeAddress()), b.wallet.Client())
	if err != nil {
		return "", "", "", 0, 0, err
	}

	var resp = "ok"
	defer func() {
		metrics.EthCallInc("validate_deposit", b.asset.Id, resp)
	}()

	iter, err := bf.FilterAssetDeposited(
		&bind.FilterOpts{
			Start: blockNumber - 1,
		},
		// user_address
		[]ethcmn.Address{ethcmn.HexToAddress(d.SourceEthereumAddress)},
		// asset_source
		[]ethcmn.Address{ethcmn.HexToAddress(b.address)})

	if err != nil {
		resp = getMaybeHTTPStatus(err)
		return "", "", "", 0, 0, err
	}

	// FIXME We should use num.Uint instead
	depamount, _ := new(big.Int).SetString(d.Amount.String(), 10)
	defer iter.Close()
	var event *bridge.BridgeAssetDeposited
	for iter.Next() {
		// here the event queu send us a 0x... pubkey
		// we do the slice operation to remove it ([2:]
		if hex.EncodeToString(iter.Event.VegaPublicKey[:]) == d.TargetPartyID[2:] &&
			iter.Event.Amount.Cmp(depamount) == 0 &&
			iter.Event.Raw.BlockNumber == blockNumber &&
			uint64(iter.Event.Raw.Index) == txIndex {
			event = iter.Event
			break
		}
	}

	if event == nil {
		return "", "", "", 0, 0, ErrUnableToFindDeposit
	}

	// now ensure we have enough confirmations
	if err := b.checkConfirmations(event.Raw.BlockNumber); err != nil {
		return "", "", "", 0, 0, err
	}

	return d.TargetPartyID, d.VegaAssetID, event.Raw.TxHash.Hex(), iter.Event.Amount.Uint64(), event.Raw.Index, nil
}

func (b *ERC20) checkConfirmations(txBlock uint64) error {
	curBlock, err := b.wallet.CurrentHeight(context.Background())
	if err != nil {
		return err
	}

	if curBlock < txBlock ||
		(curBlock-txBlock) < uint64(b.wallet.ConfirmationsRequired()) {
		return ErrMissingConfirmations
	}

	return nil
}

func (b *ERC20) String() string {
	return fmt.Sprintf("id(%v) name(%v) symbol(%v) totalSupply(%v) decimals(%v)",
		b.asset.Id, b.asset.Name, b.asset.Symbol, b.asset.TotalSupply,
		b.asset.Decimals)
}

func getMaybeHTTPStatus(err error) string {
	errstr := err.Error()
	if len(errstr) < 3 {
		return "tooshort"
	}
	i, err := strconv.Atoi(errstr[:3])
	if err != nil {
		return "nan"
	}
	if http.StatusText(i) == "" {
		return "unknown"
	}

	return errstr[:3]
}
