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
	types "code.vegaprotocol.io/vega/proto"

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
	ErrNotAnErc20Asset                = errors.New("not an erc20 asset")
)

type ERC20 struct {
	asset   *types.Asset
	address string
	ok      bool
	wallet  nodewallet.ETHWallet
}

func New(id string, asset *types.AssetDetails, w nodewallet.Wallet) (*ERC20, error) {
	wal, ok := w.(nodewallet.ETHWallet)
	if !ok {
		return nil, ErrMissingETHWalletFromNodeWallet
	}

	source := asset.GetErc20()
	if source == nil {
		return nil, ErrNotAnErc20Asset
	}

	return &ERC20{
		asset: &types.Asset{
			Id:      id,
			Details: asset,
		},
		address: source.ContractAddress,
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

	var carryErr error

	if name, err := t.Name(&bind.CallOpts{}); err != nil {
		carryErr = fmt.Errorf("couldn't get name %v: %w", err, carryErr)
	} else if name != b.asset.Details.Name {
		carryErr = maybeError(err, "invalid name, expected(%s), got(%s)", b.asset.Details.Name, name)
	}

	if symbol, err := t.Symbol(&bind.CallOpts{}); err != nil {
		carryErr = fmt.Errorf("couldn't get symbol %v: %w", err, carryErr)
	} else if symbol != b.asset.Details.Symbol {
		carryErr = maybeError(carryErr, "invalid symbol, expected(%s), got(%s)", b.asset.Details.Symbol, symbol)
	}

	if decimals, err := t.Decimals(&bind.CallOpts{}); err != nil {
		carryErr = fmt.Errorf("couldn't get decimals %v: %w", err, carryErr)
	} else if uint64(decimals) != b.asset.Details.Decimals {
		carryErr = maybeError(carryErr, "invalid decimals, expected(%d), got(%d)", b.asset.Details.Decimals, decimals)
	}

	// FIXME: We do not check the total supply for now.
	// It's for normal asset never really used, and will also vary
	// if new coins are minted...
	// if totalSupply, err := t.TotalSupply(&bind.CallOpts{}); err != nil {
	// 	carryErr = fmt.Errorf("couldn't get totalSupply %v: %w", err, carryErr)
	// } else if totalSupply.String() != b.asset.Details.TotalSupply {
	// 	carryErr = maybeError(carryErr, "invalid symbol, expected(%s), got(%s)", b.asset.Details.TotalSupply, totalSupply)
	// }

	if carryErr != nil {
		return carryErr
	}

	b.ok = true
	return nil
}

func maybeError(err error, format string, a ...interface{}) error {
	if err != nil {
		format = format + ": %w"
		args := []interface{}{}
		args = append(args, a...)
		args = append(args, err)
		return fmt.Errorf(format, args...)
	}
	return fmt.Errorf(format, a...)
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

func (b *ERC20) ValidateAssetList(w *types.ERC20AssetList, blockNumber, txIndex uint64) (hash string, logIndex uint, err error) {
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
	amount uint64,
	expiry int64,
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
	buf, err := args.Pack([]interface{}{addr, big.NewInt(int64(amount)), big.NewInt(expiry), hexEthPartyAddress, withdrawRef, withdrawContractName}...)
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

func (b *ERC20) ValidateWithdrawal(w *types.ERC20Withdrawal, blockNumber, txIndex uint64) (*big.Int, string, uint, error) {
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

	depamount, _ := new(big.Int).SetString(d.Amount, 10)
	defer iter.Close()
	var event *bridge.BridgeAssetDeposited
	for iter.Next() {
		// here the event queu send us a 0x... pubkey
		// we do the slice operation to remove it ([2:]
		if hex.EncodeToString(iter.Event.VegaPublicKey[:]) == d.TargetPartyId[2:] &&
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

	return d.TargetPartyId, d.VegaAssetId, event.Raw.TxHash.Hex(), iter.Event.Amount.Uint64(), event.Raw.Index, nil
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
		b.asset.Id, b.asset.Details.Name, b.asset.Details.Symbol, b.asset.Details.TotalSupply,
		b.asset.Details.Decimals)
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
