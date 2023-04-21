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
	UnlockWallet(ctx context.Context, name, passphrase string) error
	WalletExists(ctx context.Context, name string) (bool, error)
	CreateWallet(ctx context.Context, w wallet.Wallet, passphrase string) error
	UpdateWallet(ctx context.Context, w wallet.Wallet) error
	GetWallet(ctx context.Context, name string) (wallet.Wallet, error)
	GetWalletPath(name string) string
	ListWallets(ctx context.Context) ([]string, error)
}

type Handler struct {
	store Store

	// just to make sure we do not access same file concurrently or the map
	mu sync.RWMutex
}

func NewHandler(store Store) *Handler {
	return &Handler{
		store: store,
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

	if err := h.store.CreateWallet(context.Background(), w, passphrase); err != nil {
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

	if err := h.store.CreateWallet(context.Background(), w, passphrase); err != nil {
		return err
	}

	return nil
}

func (h *Handler) LoginWallet(name, passphrase string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	ctx := context.Background()

	if exists, err := h.store.WalletExists(ctx, name); err != nil {
		return fmt.Errorf("couldn't verify wallet existence: %w", err)
	} else if !exists {
		return ErrWalletDoesNotExists
	}

	if err := h.store.UnlockWallet(ctx, name, passphrase); err != nil {
		if errors.Is(err, wallet.ErrWrongPassphrase) {
			return err
		}
		return fmt.Errorf("couldn't unlock wallet %q: %w", name, err)
	}

	if _, err := h.store.GetWallet(ctx, name); err != nil {
		return fmt.Errorf("couldn't get wallet %q: %w", name, err)
	}

	return nil
}

func (h *Handler) GenerateKeyPair(name, passphrase string, meta []wallet.Metadata) (wallet.KeyPair, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err := h.store.UnlockWallet(context.Background(), name, passphrase); err != nil {
		if errors.Is(err, wallet.ErrWrongPassphrase) {
			return nil, err
		}
		return nil, fmt.Errorf("couldn't unlock wallet %q: %w", name, err)
	}

	w, err := h.store.GetWallet(context.Background(), name)
	if err != nil {
		return nil, fmt.Errorf("couldn't get wallet %q: %w", name, err)
	}

	kp, err := w.GenerateKeyPair(meta)
	if err != nil {
		return nil, err
	}

	if err := h.store.UpdateWallet(context.Background(), w); err != nil {
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

	w, err := h.store.GetWallet(context.Background(), name)
	if err != nil {
		return nil, fmt.Errorf("couldn't get wallet %q: %w", name, err)
	}

	return w.DescribePublicKey(pubKey)
}

func (h *Handler) ListPublicKeys(name string) ([]wallet.PublicKey, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	w, err := h.store.GetWallet(context.Background(), name)
	if err != nil {
		return nil, fmt.Errorf("couldn't get wallet %q: %w", name, err)
	}

	return w.ListPublicKeys(), nil
}

func (h *Handler) ListKeyPairs(name string) ([]wallet.KeyPair, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	w, err := h.store.GetWallet(context.Background(), name)
	if err != nil {
		return nil, fmt.Errorf("couldn't get wallet %q: %w", name, err)
	}

	return w.ListKeyPairs(), nil
}

func (h *Handler) SignAny(name string, inputData []byte, pubKey string) ([]byte, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	w, err := h.store.GetWallet(context.Background(), name)
	if err != nil {
		return nil, fmt.Errorf("couldn't get wallet %q: %w", name, err)
	}

	return w.SignAny(pubKey, inputData)
}

func (h *Handler) SignTx(name string, req *walletpb.SubmitTransactionRequest, height uint64, chainID string) (*commandspb.Transaction, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	w, err := h.store.GetWallet(context.Background(), name)
	if err != nil {
		return nil, fmt.Errorf("couldn't get wallet %q: %w", name, err)
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

	if err := h.store.UnlockWallet(context.Background(), name, passphrase); err != nil {
		if errors.Is(err, wallet.ErrWrongPassphrase) {
			return err
		}
		return fmt.Errorf("couldn't unlock wallet %q: %w", name, err)
	}

	w, err := h.store.GetWallet(context.Background(), name)
	if err != nil {
		return fmt.Errorf("couldn't get wallet %q: %w", name, err)
	}

	err = w.TaintKey(pubKey)
	if err != nil {
		return err
	}

	if err := h.store.UpdateWallet(context.Background(), w); err != nil {
		return err
	}

	return nil
}

func (h *Handler) UntaintKey(name string, pubKey string, passphrase string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err := h.store.UnlockWallet(context.Background(), name, passphrase); err != nil {
		if errors.Is(err, wallet.ErrWrongPassphrase) {
			return err
		}
		return fmt.Errorf("couldn't unlock wallet %q: %w", name, err)
	}

	w, err := h.store.GetWallet(context.Background(), name)
	if err != nil {
		return fmt.Errorf("couldn't get wallet %q: %w", name, err)
	}

	err = w.UntaintKey(pubKey)
	if err != nil {
		return err
	}

	if err := h.store.UpdateWallet(context.Background(), w); err != nil {
		return err
	}

	return nil
}

func (h *Handler) UpdateMeta(name, pubKey, passphrase string, meta []wallet.Metadata) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err := h.store.UnlockWallet(context.Background(), name, passphrase); err != nil {
		if errors.Is(err, wallet.ErrWrongPassphrase) {
			return err
		}
		return fmt.Errorf("couldn't unlock wallet %q: %w", name, err)
	}

	w, err := h.store.GetWallet(context.Background(), name)
	if err != nil {
		return fmt.Errorf("couldn't get wallet %q: %w", name, err)
	}

	_, err = w.AnnotateKey(pubKey, meta)
	if err != nil {
		return err
	}

	if err := h.store.UpdateWallet(context.Background(), w); err != nil {
		return err
	}

	return nil
}

func (h *Handler) GetWalletPath(name string) (string, error) {
	return h.store.GetWalletPath(name), nil
}
