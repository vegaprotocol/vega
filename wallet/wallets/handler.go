package wallets

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/commands"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	walletpb "code.vegaprotocol.io/vega/protos/vega/wallet/v1"
	wcommands "code.vegaprotocol.io/vega/wallet/commands"
	wcrypto "code.vegaprotocol.io/vega/wallet/crypto"
	"code.vegaprotocol.io/vega/wallet/wallet"
)

var ErrWalletDoesNotExists = errors.New("wallet does not exist")

// Store abstracts the underlying storage for wallet data.
type Store interface {
	WalletExists(ctx context.Context, name string) (bool, error)
	SaveWallet(ctx context.Context, w wallet.Wallet, passphrase string) error
	GetWallet(ctx context.Context, name, passphrase string) (wallet.Wallet, error)
	GetWalletPath(name string) string
	ListWallets(ctx context.Context) ([]string, error)
}

type Handler struct {
	store         Store
	loggedWallets wallets

	// just to make sure we do not access same file concurrently or the map
	mu sync.RWMutex
}

func NewHandler(store Store) *Handler {
	return &Handler{
		store:         store,
		loggedWallets: newWallets(),
	}
}

func (h *Handler) WalletExists(name string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	exist, _ := h.store.WalletExists(context.Background(), name)
	return exist
}

func (h *Handler) ListWallets() ([]string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.store.ListWallets(context.Background())
}

func (h *Handler) CreateWallet(name, passphrase string) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if exists, err := h.store.WalletExists(context.Background(), name); err != nil {
		return "", fmt.Errorf("couldn't verify the wallet existence: %w", err)
	} else if exists {
		return "", wallet.ErrWalletAlreadyExists
	}

	w, recoveryPhrase, err := wallet.NewHDWallet(name)
	if err != nil {
		return "", err
	}

	err = h.saveWallet(w, passphrase)
	if err != nil {
		return "", err
	}

	return recoveryPhrase, nil
}

func (h *Handler) ImportWallet(name, passphrase, recoveryPhrase string, keyDerivationVersion uint32) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if exists, err := h.store.WalletExists(context.Background(), name); err != nil {
		return fmt.Errorf("couldn't verify wallet existence: %w", err)
	} else if exists {
		return wallet.ErrWalletAlreadyExists
	}

	w, err := wallet.ImportHDWallet(name, recoveryPhrase, keyDerivationVersion)
	if err != nil {
		return err
	}

	return h.saveWallet(w, passphrase)
}

func (h *Handler) LoginWallet(name, passphrase string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if exists, err := h.store.WalletExists(context.Background(), name); err != nil {
		return fmt.Errorf("couldn't verify wallet existence: %w", err)
	} else if !exists {
		return ErrWalletDoesNotExists
	}

	w, err := h.store.GetWallet(context.Background(), name, passphrase)
	if err != nil {
		if errors.Is(err, wallet.ErrWrongPassphrase) {
			return err
		}
		return fmt.Errorf("couldn't get wallet %s: %w", name, err)
	}

	h.loggedWallets.Add(w)

	return nil
}

func (h *Handler) LogoutWallet(name string) {
	h.loggedWallets.Remove(name)
}

func (h *Handler) GenerateKeyPair(name, passphrase string, meta []wallet.Metadata) (wallet.KeyPair, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	w, err := h.store.GetWallet(context.Background(), name, passphrase)
	if err != nil {
		if errors.Is(err, wallet.ErrWrongPassphrase) {
			return nil, err
		}
		return nil, fmt.Errorf("couldn't get wallet %s: %w", name, err)
	}

	kp, err := w.GenerateKeyPair(meta)
	if err != nil {
		return nil, err
	}

	err = h.saveWallet(w, passphrase)
	if err != nil {
		return nil, err
	}

	return kp, nil
}

func (h *Handler) SecureGenerateKeyPair(name, passphrase string, meta []wallet.Metadata) (string, error) {
	kp, err := h.GenerateKeyPair(name, passphrase, meta)
	if err != nil {
		return "", err
	}

	return kp.PublicKey(), nil
}

func (h *Handler) GetPublicKey(name, pubKey string) (wallet.PublicKey, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	w, err := h.getLoggedWallet(name)
	if err != nil {
		return nil, err
	}

	return w.DescribePublicKey(pubKey)
}

func (h *Handler) ListPublicKeys(name string) ([]wallet.PublicKey, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	w, err := h.getLoggedWallet(name)
	if err != nil {
		return nil, err
	}

	return w.ListPublicKeys(), nil
}

func (h *Handler) ListKeyPairs(name string) ([]wallet.KeyPair, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	w, err := h.getLoggedWallet(name)
	if err != nil {
		return nil, err
	}

	return w.ListKeyPairs(), nil
}

func (h *Handler) SignAny(name string, inputData []byte, pubKey string) ([]byte, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	w, err := h.getLoggedWallet(name)
	if err != nil {
		return nil, err
	}

	return w.SignAny(pubKey, inputData)
}

func (h *Handler) SignTx(name string, req *walletpb.SubmitTransactionRequest, height uint64, chainID string) (*commandspb.Transaction, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	w, err := h.getLoggedWallet(name)
	if err != nil {
		return nil, err
	}

	marshaledInputData, err := wcommands.ToMarshaledInputData(req, height)
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal input data: %w", err)
	}

	pubKey := req.GetPubKey()
	signature, err := w.SignTx(pubKey, commands.BundleInputDataForSigning(marshaledInputData, chainID))
	if err != nil {
		return nil, fmt.Errorf("couldn't sign transaction: %w", err)
	}

	protoSignature := &commandspb.Signature{
		Value:   signature.Value,
		Algo:    signature.Algo,
		Version: signature.Version,
	}
	return commands.NewTransaction(pubKey, marshaledInputData, protoSignature), nil
}

func (h *Handler) VerifyAny(inputData, sig []byte, pubKey string) (bool, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return wcrypto.VerifyMessage(&wcrypto.VerifyMessageRequest{
		Message:   inputData,
		Signature: sig,
		PubKey:    pubKey,
	})
}

func (h *Handler) TaintKey(name, pubKey, passphrase string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	w, err := h.store.GetWallet(context.Background(), name, passphrase)
	if err != nil {
		if errors.Is(err, wallet.ErrWrongPassphrase) {
			return err
		}
		return fmt.Errorf("couldn't get wallet %s: %w", name, err)
	}

	err = w.TaintKey(pubKey)
	if err != nil {
		return err
	}

	return h.saveWallet(w, passphrase)
}

func (h *Handler) UntaintKey(name string, pubKey string, passphrase string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	w, err := h.store.GetWallet(context.Background(), name, passphrase)
	if err != nil {
		if errors.Is(err, wallet.ErrWrongPassphrase) {
			return err
		}
		return fmt.Errorf("couldn't get wallet %s: %w", name, err)
	}

	err = w.UntaintKey(pubKey)
	if err != nil {
		return err
	}

	return h.saveWallet(w, passphrase)
}

func (h *Handler) UpdateMeta(name, pubKey, passphrase string, meta []wallet.Metadata) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	w, err := h.store.GetWallet(context.Background(), name, passphrase)
	if err != nil {
		if errors.Is(err, wallet.ErrWrongPassphrase) {
			return err
		}
		return fmt.Errorf("couldn't get wallet %s: %w", name, err)
	}

	_, err = w.AnnotateKey(pubKey, meta)
	if err != nil {
		return err
	}

	return h.saveWallet(w, passphrase)
}

func (h *Handler) GetWalletPath(name string) (string, error) {
	return h.store.GetWalletPath(name), nil
}

func (h *Handler) saveWallet(w wallet.Wallet, passphrase string) error {
	err := h.store.SaveWallet(context.Background(), w, passphrase)
	if err != nil {
		return err
	}

	h.loggedWallets.Add(w)

	return nil
}

func (h *Handler) getLoggedWallet(name string) (wallet.Wallet, error) {
	if exists, err := h.store.WalletExists(context.Background(), name); err != nil {
		return nil, fmt.Errorf("couldn't verify wallet existence: %w", err)
	} else if !exists {
		return nil, ErrWalletDoesNotExists
	}

	w, loggedIn := h.loggedWallets.Get(name)
	if !loggedIn {
		return nil, wallet.ErrWalletNotLoggedIn
	}
	return w, nil
}

type wallets map[string]wallet.Wallet

func newWallets() wallets {
	return map[string]wallet.Wallet{}
}

func (w wallets) Add(wallet wallet.Wallet) {
	w[wallet.Name()] = wallet
}

func (w wallets) Get(name string) (wallet.Wallet, bool) {
	wal, ok := w[name]
	return wal, ok
}

func (w wallets) Remove(name string) {
	delete(w, name)
}
