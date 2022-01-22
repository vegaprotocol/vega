package erc20

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	typespb "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/assets/common"
	"code.vegaprotocol.io/vega/assets/erc20/bridge"
	"code.vegaprotocol.io/vega/bridges"
	"code.vegaprotocol.io/vega/metrics"
	ethnw "code.vegaprotocol.io/vega/nodewallets/eth"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

const MaxNonce = 100000000

var (
	ErrUnableToFindDeposit        = errors.New("unable to find erc20 deposit event")
	ErrUnableToFindWithdrawal     = errors.New("unable to find erc20 withdrawal event")
	ErrUnableToFindERC20AssetList = errors.New("unable to find erc20 asset list event")
	ErrMissingConfirmations       = errors.New("missing confirmation from ethereum")
	ErrNotAnErc20Asset            = errors.New("not an erc20 asset")
)

type ETHClient interface {
	bind.ContractBackend
	HeaderByNumber(context.Context, *big.Int) (*ethtypes.Header, error)
	CollateralBridgeAddress() ethcommon.Address
	CurrentHeight(context.Context) (uint64, error)
	ConfirmationsRequired() uint64
}

type ERC20 struct {
	asset     *types.Asset
	address   string
	ok        bool
	wallet    *ethnw.Wallet
	ethClient ETHClient
}

func New(
	id string,
	asset *types.AssetDetails,
	w *ethnw.Wallet,
	ethClient ETHClient,
) (*ERC20, error) {
	source := asset.GetErc20()
	if source == nil {
		return nil, ErrNotAnErc20Asset
	}

	return &ERC20{
		asset: &types.Asset{
			ID:      id,
			Details: asset,
		},
		address:   source.ContractAddress,
		wallet:    w,
		ethClient: ethClient,
	}, nil
}

func (e *ERC20) ProtoAsset() *typespb.Asset {
	return e.asset.IntoProto()
}

func (e ERC20) Type() *types.Asset {
	return e.asset.DeepClone()
}

func (e *ERC20) GetAssetClass() common.AssetClass {
	return common.ERC20
}

func (e *ERC20) IsValid() bool {
	return e.ok
}

// SetValidNonValidator this method is here temporarsy
// to avoid requiring ethclient for the non-validators
// will be removed once the eth client can be removed from this type.
func (e *ERC20) SetValidNonValidator() {
	e.ok = true
}

func (e *ERC20) Validate() error {
	t, err := NewErc20(ethcommon.HexToAddress(e.address), e.ethClient)
	if err != nil {
		return err
	}

	var carryErr error

	if name, err := t.Name(&bind.CallOpts{}); err != nil {
		carryErr = fmt.Errorf("couldn't get name %v: %w", err, carryErr)
	} else if name != e.asset.Details.Name {
		carryErr = maybeError(err, "invalid name, expected(%s), got(%s)", e.asset.Details.Name, name)
	}

	if symbol, err := t.Symbol(&bind.CallOpts{}); err != nil {
		carryErr = fmt.Errorf("couldn't get symbol %v: %w", err, carryErr)
	} else if symbol != e.asset.Details.Symbol {
		carryErr = maybeError(carryErr, "invalid symbol, expected(%s), got(%s)", e.asset.Details.Symbol, symbol)
	}

	if decimals, err := t.Decimals(&bind.CallOpts{}); err != nil {
		carryErr = fmt.Errorf("couldn't get decimals %v: %w", err, carryErr)
	} else if uint64(decimals) != e.asset.Details.Decimals {
		carryErr = maybeError(carryErr, "invalid decimals, expected(%d), got(%d)", e.asset.Details.Decimals, decimals)
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

	e.ok = true
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
// return the generated message and the signature for this message.
func (e *ERC20) SignBridgeListing() (msg []byte, sig []byte, err error) {
	bridgeAddress := e.ethClient.CollateralBridgeAddress().Hex()
	// use the asset ID converted into a uint256
	nonce, err := num.UintFromHex("0x" + e.asset.ID)
	if err != nil {
		return nil, nil, err
	}
	bundle, err := bridges.NewERC20Logic(e.wallet, bridgeAddress).
		ListAsset(e.address, e.asset.ID, nonce)
	if err != nil {
		return nil, nil, err
	}

	return bundle.Message, bundle.Signature, nil
}

func (e *ERC20) ValidateAssetList(w *types.ERC20AssetList, blockNumber, txIndex uint64) error {
	bf, err := bridge.NewBridgeFilterer(e.ethClient.CollateralBridgeAddress(), e.ethClient)
	if err != nil {
		return err
	}

	resp := "ok"
	defer func() {
		metrics.EthCallInc("validate_allowlist", e.asset.ID, resp)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	iter, err := bf.FilterAssetListed(
		&bind.FilterOpts{
			Start:   blockNumber - 1,
			Context: ctx,
		},
		[]ethcommon.Address{ethcommon.HexToAddress(e.address)},
		[][32]byte{},
	)
	if err != nil {
		resp = getMaybeHTTPStatus(err)
		return err
	}

	defer iter.Close()
	var event *bridge.BridgeAssetListed

	assetID := strings.TrimPrefix(w.VegaAssetID, "0x")
	for iter.Next() {
		if hex.EncodeToString(iter.Event.VegaAssetId[:]) == assetID {
			event = iter.Event

			break
		}
	}

	if event == nil {
		return ErrUnableToFindERC20AssetList
	}

	// now ensure we have enough confirmations
	if err := e.checkConfirmations(event.Raw.BlockNumber); err != nil {
		return err
	}

	return nil
}

func (e *ERC20) SignWithdrawal(
	amount *num.Uint,
	ethPartyAddress string,
	withdrawRef *big.Int,
) (msg []byte, sig []byte, err error) {
	nonce, _ := num.UintFromBig(withdrawRef)
	bridgeAddress := e.ethClient.CollateralBridgeAddress().Hex()
	bundle, err := bridges.NewERC20Logic(e.wallet, bridgeAddress).
		WithdrawAsset(e.address, amount, ethPartyAddress, nonce)
	if err != nil {
		return nil, nil, err
	}

	return bundle.Message, bundle.Signature, nil
}

func (e *ERC20) ValidateWithdrawal(w *types.ERC20Withdrawal, blockNumber, txIndex uint64) (*big.Int, string, uint, error) {
	bf, err := bridge.NewBridgeFilterer(e.ethClient.CollateralBridgeAddress(), e.ethClient)
	if err != nil {
		return nil, "", 0, err
	}

	resp := "ok"
	defer func() {
		metrics.EthCallInc("validate_withdrawal", e.asset.ID, resp)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	iter, err := bf.FilterAssetWithdrawn(
		&bind.FilterOpts{
			Start:   blockNumber - 1,
			Context: ctx,
		},
		// user_address
		[]ethcommon.Address{ethcommon.HexToAddress(w.TargetEthereumAddress)},
		// asset_source
		[]ethcommon.Address{ethcommon.HexToAddress(e.address)})
	if err != nil {
		resp = getMaybeHTTPStatus(err)
		return nil, "", 0, err
	}

	defer iter.Close()
	var event *bridge.BridgeAssetWithdrawn
	nonce := &big.Int{}
	_, ok := nonce.SetString(w.ReferenceNonce, 10)
	if !ok {
		return nil, "", 0, fmt.Errorf("could not use reference nonce, expected base 10 integer: %v", w.ReferenceNonce)
	}
	for iter.Next() {
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
	if err := e.checkConfirmations(event.Raw.BlockNumber); err != nil {
		return nil, "", 0, err
	}

	return nonce, event.Raw.TxHash.Hex(), event.Raw.Index, nil
}

func (e *ERC20) ValidateDeposit(d *types.ERC20Deposit, blockNumber, txIndex uint64) error {
	bf, err := bridge.NewBridgeFilterer(e.ethClient.CollateralBridgeAddress(), e.ethClient)
	if err != nil {
		return err
	}

	resp := "ok"
	defer func() {
		metrics.EthCallInc("validate_deposit", e.asset.ID, resp)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	iter, err := bf.FilterAssetDeposited(
		&bind.FilterOpts{
			Start:   blockNumber - 1,
			Context: ctx,
		},
		// user_address
		[]ethcommon.Address{ethcommon.HexToAddress(d.SourceEthereumAddress)},
		// asset_source
		[]ethcommon.Address{ethcommon.HexToAddress(e.address)})
	if err != nil {
		resp = getMaybeHTTPStatus(err)
		return err
	}

	depamount := d.Amount.BigInt()
	defer iter.Close()
	var event *bridge.BridgeAssetDeposited
	targetPartyID := strings.TrimPrefix(d.TargetPartyID, "0x")
	for iter.Next() {
		if hex.EncodeToString(iter.Event.VegaPublicKey[:]) == targetPartyID &&
			iter.Event.Amount.Cmp(depamount) == 0 &&
			iter.Event.Raw.BlockNumber == blockNumber &&
			uint64(iter.Event.Raw.Index) == txIndex {
			event = iter.Event
			break
		}
	}

	if event == nil {
		return ErrUnableToFindDeposit
	}

	// now ensure we have enough confirmations
	if err := e.checkConfirmations(event.Raw.BlockNumber); err != nil {
		return err
	}

	return nil
}

func (e *ERC20) checkConfirmations(txBlock uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	curBlock, err := e.ethClient.CurrentHeight(ctx)
	if err != nil {
		return err
	}

	if curBlock < txBlock || (curBlock-txBlock) < e.ethClient.ConfirmationsRequired() {
		return ErrMissingConfirmations
	}

	return nil
}

func (e *ERC20) String() string {
	return fmt.Sprintf("id(%v) name(%v) symbol(%v) totalSupply(%v) decimals(%v)",
		e.asset.ID, e.asset.Details.Name, e.asset.Details.Symbol, e.asset.Details.TotalSupply,
		e.asset.Details.Decimals)
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
