package wallets

import (
	"errors"
	"sync"

	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"
)

var (
	ErrMarketInsufficientFunds = errors.New("insufficient funds in market general account")
	ErrTraderInsufficientFunds = errors.New("insufficient funds in trader general account")
	ErrMarketNoAccountForAsset = errors.New("market do not have a general account for th asset")
	ErrTraderNoAccountForAsset = errors.New("trader do not have a general account for the asset")
	ErrNoAccountForTrader      = errors.New("no accounts for trader")
	ErrNoAccountForMarket      = errors.New("no accounts for market")
	ErrInvalidWalletType       = errors.New("invalid wallet type")
)

type WalletType int8

const (
	TraderWalletType WalletType = iota
	MarketWalletType
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/buffer_mock.go -package mocks code.vegaprotocol.io/vega/internal/wallets Buffer
type Buffer interface {
	Add(owner, marketID, asset string, ty types.AccountType, balance int64)
}

type walletKey struct {
	ownerID string
	ty      WalletType
}

type Wallets struct {
	buf Buffer

	// {ownerID, ty} -> asset -> balance
	accs map[walletKey]map[string]int64
	mu   sync.Mutex
}

func New(log *logging.Logger, buf Buffer) *Wallets {
	return &Wallets{
		buf:  buf,
		accs: map[walletKey]map[string]int64{},
	}
}

// GetOrCreate will create a new market general account
// this account may already exists.
// return the balance for the account
func (w *Wallets) GetCreate(ty WalletType, ownerID, asset string) int64 {
	w.mu.Lock()

	accs, ok := w.accs[walletKey{ownerID, ty}]
	if !ok {
		// if not exist just make the balance = 0
		accs = map[string]int64{
			asset: 0,
		}
		w.accs[walletKey{ownerID, ty}] = accs
	}
	balance, ok := accs[asset]
	if !ok {
		accs[asset] = 0
	}

	w.addToBuf(ty, ownerID, asset, balance)
	w.mu.Unlock()
	return balance
}

func (w *Wallets) GetBalance(ty WalletType, ownerID, asset string) (int64, error) {
	w.mu.Lock()
	accs, ok := w.accs[walletKey{ownerID, ty}]
	if !ok {
		w.mu.Unlock()
		return 0, getNoAccountError(ty)
	}

	balance, ok := accs[asset]
	if !ok {
		w.mu.Unlock()
		return 0, getNoAccountForAssetError(ty)
	}

	w.mu.Unlock()
	return balance, nil
}

func (w *Wallets) Withdraw(ty WalletType, ownerID, asset string, amount int64) (int64, error) {
	w.mu.Lock()

	accs, ok := w.accs[walletKey{ownerID, ty}]
	if !ok {
		w.mu.Unlock()
		return 0, getNoAccountError(ty)
	}

	balance, ok := accs[asset]
	if !ok {
		w.mu.Unlock()
		return 0, getNoAccountForAssetError(ty)
	}

	if balance < amount {
		w.mu.Unlock()
		return balance, getUnsufficientFundsError(ty)
	}
	balance -= amount
	accs[asset] = balance

	w.addToBuf(ty, ownerID, asset, balance)
	w.mu.Unlock()
	return balance, nil
}

func (w *Wallets) Credit(ty WalletType, ownerID, asset string, amount int64) (int64, error) {
	w.mu.Lock()

	accs, ok := w.accs[walletKey{ownerID, ty}]
	if !ok {
		w.mu.Unlock()
		return 0, getNoAccountError(ty)
	}

	balance, ok := accs[asset]
	if !ok {
		w.mu.Unlock()
		return 0, getNoAccountForAssetError(ty)
	}

	balance += amount
	accs[asset] = balance

	w.addToBuf(ty, ownerID, asset, balance)
	w.mu.Unlock()
	return balance, nil
}

func (w *Wallets) Move(
	fromTy WalletType, fromOwnerID string,
	toTy WalletType, toOwnerID string,
	asset string, amount int64) (fromBalance int64, toBalance int64, err error) {
	w.mu.Lock()

	fromAccs, ok := w.accs[walletKey{fromOwnerID, fromTy}]
	if !ok {
		err = getNoAccountError(fromTy)
		w.mu.Unlock()
		return
	}

	fromBalance, ok = fromAccs[asset]
	if !ok {
		err = getNoAccountForAssetError(fromTy)
		w.mu.Unlock()
		return
	}

	if fromBalance < amount {
		err = getUnsufficientFundsError(fromTy)
		w.mu.Unlock()
		return
	}

	toAccs, ok := w.accs[walletKey{toOwnerID, toTy}]
	if !ok {
		err = getNoAccountError(toTy)
		w.mu.Unlock()
		return
	}

	toBalance, ok = toAccs[asset]
	if !ok {
		err = getNoAccountForAssetError(toTy)
		w.mu.Unlock()
		return
	}

	fromBalance -= amount
	toBalance += amount
	fromAccs[asset] = fromBalance
	toAccs[asset] = toBalance

	w.addToBuf(fromTy, fromOwnerID, asset, fromBalance)
	w.addToBuf(toTy, toOwnerID, asset, toBalance)
	w.mu.Unlock()
	return
}

func (w *Wallets) addToBuf(ty WalletType, ownerID, asset string, balance int64) {
	switch ty {
	case MarketWalletType:
		w.buf.Add("", ownerID, asset, types.AccountType_INSURANCE, balance)
	case TraderWalletType:
		w.buf.Add(ownerID, "", asset, types.AccountType_GENERAL, balance)
	}
}

func getNoAccountError(ty WalletType) error {
	switch ty {
	case MarketWalletType:
		return ErrNoAccountForMarket
	case TraderWalletType:
		return ErrNoAccountForTrader
	default:
		return ErrInvalidWalletType
	}
}

func getUnsufficientFundsError(ty WalletType) error {
	switch ty {
	case MarketWalletType:
		return ErrMarketInsufficientFunds
	case TraderWalletType:
		return ErrTraderInsufficientFunds
	default:
		return ErrInvalidWalletType
	}
}

func getNoAccountForAssetError(ty WalletType) error {
	switch ty {
	case MarketWalletType:
		return ErrMarketNoAccountForAsset
	case TraderWalletType:
		return ErrTraderNoAccountForAsset
	default:
		return ErrInvalidWalletType
	}
}
