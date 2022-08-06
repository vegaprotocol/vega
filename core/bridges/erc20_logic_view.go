package bridges

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

	bridgecontract "code.vegaprotocol.io/vega/core/contracts/erc20_bridge_logic_restricted"
	"code.vegaprotocol.io/vega/core/metrics"
	"code.vegaprotocol.io/vega/core/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

var (
	ErrUnableToFindERC20AssetList     = errors.New("unable to find erc20 asset list event")
	ErrUnableToFindERC20BridgeStopped = errors.New("unable to find erc20 bridge stopped event")
	ErrUnableToFindERC20BridgeResumed = errors.New("unable to find erc20 bridge resumed event")
	ErrUnableToFindERC20Deposit       = errors.New("unable to find erc20 asset deposit")
	ErrUnableToFindERC20Withdrawal    = errors.New("unabled to find erc20 asset withdrawal")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/eth_client_mock.go -package mocks code.vegaprotocol.io/vega/core/bridges ETHClient
type ETHClient interface {
	bind.ContractFilterer
	HeaderByNumber(context.Context, *big.Int) (*ethtypes.Header, error)
	CollateralBridgeAddress() ethcommon.Address
	CurrentHeight(context.Context) (uint64, error)
	ConfirmationsRequired() uint64
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/eth_confirmations_mock.go -package mocks code.vegaprotocol.io/vega/core/bridges EthConfirmations
type EthConfirmations interface {
	Check(uint64) error
}

// ERC20Logic yea that's a weird name but
// it just matches the name of the contract.
type ERC20LogicView struct {
	clt      ETHClient
	ethConfs EthConfirmations
}

func NewERC20LogicView(
	clt ETHClient,
	ethConfs EthConfirmations,
) *ERC20LogicView {
	return &ERC20LogicView{
		clt:      clt,
		ethConfs: ethConfs,
	}
}

// FindAssetList will look at the ethereum logs and try to find the
// given transaction.
func (e *ERC20LogicView) FindAssetList(
	al *types.ERC20AssetList,
	blockNumber,
	logIndex uint64,
) error {
	bf, err := bridgecontract.NewErc20BridgeLogicRestrictedFilterer(
		e.clt.CollateralBridgeAddress(), e.clt)
	if err != nil {
		return err
	}

	resp := "ok"
	defer func() {
		metrics.EthCallInc("find_asset_list", al.VegaAssetID, resp)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	iter, err := bf.FilterAssetListed(
		&bind.FilterOpts{
			Start:   blockNumber - 1,
			Context: ctx,
		},
		[]ethcommon.Address{ethcommon.HexToAddress(al.AssetSource)},
		[][32]byte{},
	)
	if err != nil {
		resp = getMaybeHTTPStatus(err)
		return err
	}
	defer iter.Close()

	var event *bridgecontract.Erc20BridgeLogicRestrictedAssetListed
	assetID := strings.TrimPrefix(al.VegaAssetID, "0x")

	for iter.Next() {
		if hex.EncodeToString(iter.Event.VegaAssetId[:]) == assetID &&
			iter.Event.Raw.BlockNumber == blockNumber &&
			uint64(iter.Event.Raw.Index) == logIndex {
			event = iter.Event

			break
		}
	}

	if event == nil {
		return ErrUnableToFindERC20AssetList
	}

	// now ensure we have enough confirmations
	if err := e.ethConfs.Check(event.Raw.BlockNumber); err != nil {
		return err
	}

	return nil
}

// FindBridgeStopped will look at the ethereum logs and try to find the
// given transaction.
func (e *ERC20LogicView) FindBridgeStopped(
	al *types.ERC20EventBridgeStopped,
	blockNumber,
	logIndex uint64,
) error {
	bf, err := bridgecontract.NewErc20BridgeLogicRestrictedFilterer(
		e.clt.CollateralBridgeAddress(), e.clt)
	if err != nil {
		return err
	}

	resp := "ok"
	defer func() {
		metrics.EthCallInc("find_bridge_stopped", resp)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	iter, err := bf.FilterBridgeStopped(
		&bind.FilterOpts{
			Start:   blockNumber - 1,
			Context: ctx,
		},
	)
	if err != nil {
		resp = getMaybeHTTPStatus(err)
		return err
	}
	defer iter.Close()

	var event *bridgecontract.Erc20BridgeLogicRestrictedBridgeStopped

	for iter.Next() {
		if iter.Event.Raw.BlockNumber == blockNumber &&
			uint64(iter.Event.Raw.Index) == logIndex {
			event = iter.Event

			break
		}
	}

	if event == nil {
		return ErrUnableToFindERC20BridgeStopped
	}

	// now ensure we have enough confirmations
	if err := e.ethConfs.Check(event.Raw.BlockNumber); err != nil {
		return err
	}

	return nil
}

// FindBridgeResumed will look at the ethereum logs and try to find the
// given transaction.
func (e *ERC20LogicView) FindBridgeResumed(
	al *types.ERC20EventBridgeResumed,
	blockNumber,
	logIndex uint64,
) error {
	bf, err := bridgecontract.NewErc20BridgeLogicRestrictedFilterer(
		e.clt.CollateralBridgeAddress(), e.clt)
	if err != nil {
		return err
	}

	resp := "ok"
	defer func() {
		metrics.EthCallInc("find_bridge_stopped", resp)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	iter, err := bf.FilterBridgeResumed(
		&bind.FilterOpts{
			Start:   blockNumber - 1,
			Context: ctx,
		},
	)
	if err != nil {
		resp = getMaybeHTTPStatus(err)
		return err
	}
	defer iter.Close()

	var event *bridgecontract.Erc20BridgeLogicRestrictedBridgeResumed

	for iter.Next() {
		if iter.Event.Raw.BlockNumber == blockNumber &&
			uint64(iter.Event.Raw.Index) == logIndex {
			event = iter.Event

			break
		}
	}

	if event == nil {
		return ErrUnableToFindERC20BridgeStopped
	}

	// now ensure we have enough confirmations
	if err := e.ethConfs.Check(event.Raw.BlockNumber); err != nil {
		return err
	}

	return nil
}

func (e *ERC20LogicView) FindDeposit(
	d *types.ERC20Deposit,
	blockNumber, logIndex uint64,
	ethAssetAddress string,
) error {
	bf, err := bridgecontract.NewErc20BridgeLogicRestrictedFilterer(
		e.clt.CollateralBridgeAddress(), e.clt)
	if err != nil {
		return err
	}

	resp := "ok"
	defer func() {
		metrics.EthCallInc("find_deposit", d.VegaAssetID, resp)
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
		[]ethcommon.Address{ethcommon.HexToAddress(ethAssetAddress)})
	if err != nil {
		resp = getMaybeHTTPStatus(err)
		return err
	}
	defer iter.Close()

	depamount := d.Amount.BigInt()
	var event *bridgecontract.Erc20BridgeLogicRestrictedAssetDeposited
	targetPartyID := strings.TrimPrefix(d.TargetPartyID, "0x")

	for iter.Next() {
		if hex.EncodeToString(iter.Event.VegaPublicKey[:]) == targetPartyID &&
			iter.Event.Amount.Cmp(depamount) == 0 &&
			iter.Event.Raw.BlockNumber == blockNumber &&
			uint64(iter.Event.Raw.Index) == logIndex {
			event = iter.Event
			break
		}
	}

	if event == nil {
		return ErrUnableToFindERC20Deposit
	}

	// now ensure we have enough confirmations
	if err := e.ethConfs.Check(event.Raw.BlockNumber); err != nil {
		return err
	}

	return nil
}

func (e *ERC20LogicView) FindWithdrawal(
	w *types.ERC20Withdrawal,
	blockNumber, logIndex uint64,
	ethAssetAddress string,
) (*big.Int, string, uint, error) {
	bf, err := bridgecontract.NewErc20BridgeLogicRestrictedFilterer(
		e.clt.CollateralBridgeAddress(), e.clt)
	if err != nil {
		return nil, "", 0, err
	}

	resp := "ok"
	defer func() {
		metrics.EthCallInc("find_withdrawal", w.VegaAssetID, resp)
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
		[]ethcommon.Address{ethcommon.HexToAddress(ethAssetAddress)})
	if err != nil {
		resp = getMaybeHTTPStatus(err)
		return nil, "", 0, err
	}
	defer iter.Close()

	var event *bridgecontract.Erc20BridgeLogicRestrictedAssetWithdrawn
	nonce := &big.Int{}
	_, ok := nonce.SetString(w.ReferenceNonce, 10)
	if !ok {
		return nil, "", 0, fmt.Errorf("could not use reference nonce, expected base 10 integer: %v", w.ReferenceNonce)
	}

	for iter.Next() {
		if nonce.Cmp(iter.Event.Nonce) == 0 &&
			iter.Event.Raw.BlockNumber == blockNumber &&
			uint64(iter.Event.Raw.Index) == logIndex {
			event = iter.Event

			break
		}
	}

	if event == nil {
		return nil, "", 0, ErrUnableToFindERC20Withdrawal
	}

	// now ensure we have enough confirmations
	if err := e.ethConfs.Check(event.Raw.BlockNumber); err != nil {
		return nil, "", 0, err
	}

	return nonce, event.Raw.TxHash.Hex(), event.Raw.Index, nil
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
