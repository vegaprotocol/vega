package wallet

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/commands"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	walletpb "code.vegaprotocol.io/vega/protos/vega/wallet/v1"
	wcommands "code.vegaprotocol.io/vega/wallet/commands"

	"github.com/golang/protobuf/proto"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/store_mock.go -package mocks code.vegaprotocol.io/vega/wallet/wallet Store
type Store interface {
	WalletExists(ctx context.Context, name string) (bool, error)
	SaveWallet(ctx context.Context, w Wallet, passphrase string) error
	GetWallet(ctx context.Context, name, passphrase string) (Wallet, error)
	GetWalletPath(name string) string
	ListWallets(ctx context.Context) ([]string, error)
}

type GenerateKeyRequest struct {
	Wallet     string `json:"wallet"`
	Metadata   []Meta `json:"metadata"`
	Passphrase string `json:"passphrase"`
}

type GenerateKeyResponse struct {
	PublicKey string    `json:"publicKey"`
	Algorithm Algorithm `json:"algorithm"`
	Meta      []Meta    `json:"meta"`
}

func GenerateKey(store Store, req *GenerateKeyRequest) (*GenerateKeyResponse, error) {
	resp := &GenerateKeyResponse{}

	w, err := getWallet(store, req.Wallet, req.Passphrase)
	if err != nil {
		return nil, err
	}

	req.Metadata = addDefaultKeyName(w, req.Metadata)

	kp, err := w.GenerateKeyPair(req.Metadata)
	if err != nil {
		return nil, err
	}

	if err := store.SaveWallet(context.Background(), w, req.Passphrase); err != nil {
		return nil, fmt.Errorf("couldn't save wallet: %w", err)
	}

	resp.PublicKey = kp.PublicKey()
	resp.Algorithm.Name = kp.AlgorithmName()
	resp.Algorithm.Version = kp.AlgorithmVersion()
	resp.Meta = kp.Meta()

	return resp, nil
}

type AnnotateKeyRequest struct {
	Wallet     string `json:"wallet"`
	PubKey     string `json:"pubKey"`
	Metadata   []Meta `json:"metadata"`
	Passphrase string `json:"passphrase"`
}

func AnnotateKey(store Store, req *AnnotateKeyRequest) error {
	w, err := getWallet(store, req.Wallet, req.Passphrase)
	if err != nil {
		return err
	}

	if err = w.UpdateMeta(req.PubKey, req.Metadata); err != nil {
		return fmt.Errorf("couldn't update metadata: %w", err)
	}

	if err := store.SaveWallet(context.Background(), w, req.Passphrase); err != nil {
		return fmt.Errorf("couldn't save wallet: %w", err)
	}

	return nil
}

type TaintKeyRequest struct {
	Wallet     string `json:"wallet"`
	PubKey     string `json:"pubKey"`
	Passphrase string `json:"passphrase"`
}

func TaintKey(store Store, req *TaintKeyRequest) error {
	w, err := getWallet(store, req.Wallet, req.Passphrase)
	if err != nil {
		return err
	}

	if err = w.TaintKey(req.PubKey); err != nil {
		return fmt.Errorf("couldn't taint key: %w", err)
	}

	if err := store.SaveWallet(context.Background(), w, req.Passphrase); err != nil {
		return fmt.Errorf("couldn't save wallet: %w", err)
	}

	return nil
}

type UntaintKeyRequest struct {
	Wallet     string `json:"wallet"`
	PubKey     string `json:"pubKey"`
	Passphrase string `json:"passphrase"`
}

func UntaintKey(store Store, req *UntaintKeyRequest) error {
	w, err := getWallet(store, req.Wallet, req.Passphrase)
	if err != nil {
		return err
	}

	if err = w.UntaintKey(req.PubKey); err != nil {
		return fmt.Errorf("couldn't untaint key: %w", err)
	}

	if err := store.SaveWallet(context.Background(), w, req.Passphrase); err != nil {
		return fmt.Errorf("couldn't save wallet: %w", err)
	}

	return nil
}

type IsolateKeyRequest struct {
	Wallet     string `json:"wallet"`
	PubKey     string `json:"pubKey"`
	Passphrase string `json:"passphrase"`
}

type IsolateKeyResponse struct {
	Wallet   string `json:"wallet"`
	FilePath string `json:"filePath"`
}

func IsolateKey(store Store, req *IsolateKeyRequest) (*IsolateKeyResponse, error) {
	w, err := getWallet(store, req.Wallet, req.Passphrase)
	if err != nil {
		return nil, err
	}

	isolatedWallet, err := w.IsolateWithKey(req.PubKey)
	if err != nil {
		return nil, fmt.Errorf("couldn't isolate wallet %s: %w", req.Wallet, err)
	}

	if err := store.SaveWallet(context.Background(), isolatedWallet, req.Passphrase); err != nil {
		return nil, fmt.Errorf("couldn't save isolated wallet %s: %w", isolatedWallet.Name(), err)
	}

	return &IsolateKeyResponse{
		Wallet:   isolatedWallet.Name(),
		FilePath: store.GetWalletPath(isolatedWallet.Name()),
	}, nil
}

type ListKeysRequest struct {
	Wallet     string `json:"wallet"`
	Passphrase string `json:"passphrase"`
}

type ListKeysResponse struct {
	Keys []NamedPubKey `json:"keys"`
}

type DescribeKeyRequest struct {
	Wallet     string `json:"wallet"`
	Passphrase string `json:"passphrase"`
	PubKey     string `json:"pubKey"`
}

type DescribeKeyResponse struct {
	PublicKey string    `json:"publicKey"`
	Algorithm Algorithm `json:"algorithm"`
	Meta      []Meta    `json:"meta"`
	IsTainted bool      `json:"isTainted"`
}

type NamedPubKey struct {
	Name      string `json:"name"`
	PublicKey string `json:"publicKey"`
}

func ListKeys(store Store, req *ListKeysRequest) (*ListKeysResponse, error) {
	w, err := getWallet(store, req.Wallet, req.Passphrase)
	if err != nil {
		return nil, err
	}

	kps := w.ListKeyPairs()
	keys := make([]NamedPubKey, 0, len(kps))
	for _, kp := range kps {
		keys = append(keys, NamedPubKey{
			Name:      GetKeyName(kp.Meta()),
			PublicKey: kp.PublicKey(),
		})
	}

	return &ListKeysResponse{
		Keys: keys,
	}, nil
}

func DescribeKey(store Store, req *DescribeKeyRequest) (*DescribeKeyResponse, error) {
	w, err := getWallet(store, req.Wallet, req.Passphrase)
	if err != nil {
		return nil, err
	}

	resp := &DescribeKeyResponse{}

	kp, err := w.DescribeKeyPair(req.PubKey)
	if err != nil {
		return nil, err
	}
	resp.PublicKey = kp.PublicKey()
	resp.Algorithm.Name = kp.AlgorithmName()
	resp.Algorithm.Version = kp.AlgorithmVersion()
	resp.Meta = kp.Meta()
	resp.IsTainted = kp.IsTainted()
	return resp, nil
}

type RotateKeyRequest struct {
	Wallet            string `json:"wallet"`
	Passphrase        string `json:"passphrase"`
	NewPublicKey      string `json:"newPublicKey"`
	ChainID           string `json:"chainId"`
	CurrentPublicKey  string `json:"currentPublicKey"`
	TxBlockHeight     uint64 `json:"txBlockHeight"`
	TargetBlockHeight uint64 `json:"targetBlockHeight"`
}

type RotateKeyResponse struct {
	MasterPublicKey   string `json:"masterPublicKey"`
	Base64Transaction string `json:"base64Transaction"`
}

func RotateKey(store Store, req *RotateKeyRequest) (*RotateKeyResponse, error) {
	w, err := getWallet(store, req.Wallet, req.Passphrase)
	if err != nil {
		return nil, err
	}

	mKeyPair, err := w.GetMasterKeyPair()
	if errors.Is(err, ErrIsolatedWalletDoesNotHaveMasterKey) {
		return nil, ErrCantRotateKeyInIsolatedWallet
	}
	if err != nil {
		return nil, err
	}

	pubKey, err := w.DescribePublicKey(req.NewPublicKey)
	if err != nil {
		return nil, fmt.Errorf("couldn't get the public key: %w", err)
	}

	currentPubKey, err := w.DescribePublicKey(req.CurrentPublicKey)
	if err != nil {
		return nil, fmt.Errorf("couldn't get the current public key: %w", err)
	}

	if pubKey.IsTainted() {
		return nil, ErrPubKeyIsTainted
	}

	currentPubKeyHash, err := currentPubKey.Hash()
	if err != nil {
		return nil, fmt.Errorf("couldn't hash the current public key: %w", err)
	}

	inputData := commands.NewInputData(req.TxBlockHeight)
	inputData.Command = &commandspb.InputData_KeyRotateSubmission{
		KeyRotateSubmission: &commandspb.KeyRotateSubmission{
			NewPubKeyIndex:    pubKey.Index(),
			TargetBlock:       req.TargetBlockHeight,
			NewPubKey:         pubKey.Key(),
			CurrentPubKeyHash: currentPubKeyHash,
		},
	}

	data, err := commands.MarshalInputData(req.ChainID, inputData)
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal key rotate submission input data: %w", err)
	}

	sign, err := mKeyPair.Sign(data)
	if err != nil {
		return nil, fmt.Errorf("couldn't sign key rotate submission input data: %w", err)
	}

	protoSignature := &commandspb.Signature{
		Value:   sign.Value,
		Algo:    sign.Algo,
		Version: sign.Version,
	}

	transaction := commands.NewTransaction(mKeyPair.PublicKey(), data, protoSignature)
	transactionRaw, err := proto.Marshal(transaction)
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal transaction: %w", err)
	}

	return &RotateKeyResponse{
		MasterPublicKey:   mKeyPair.PublicKey(),
		Base64Transaction: base64.StdEncoding.EncodeToString(transactionRaw),
	}, nil
}

type GetWalletInfoRequest struct {
	Wallet     string `json:"wallet"`
	Passphrase string `json:"passphrase"`
}

type GetWalletInfoResponse struct {
	Type    string `json:"type"`
	Version uint32 `json:"version"`
	ID      string `json:"id"`
}

func GetWalletInfo(store Store, req *GetWalletInfoRequest) (*GetWalletInfoResponse, error) {
	w, err := getWallet(store, req.Wallet, req.Passphrase)
	if err != nil {
		return nil, err
	}

	return &GetWalletInfoResponse{
		Type:    w.Type(),
		Version: w.Version(),
		ID:      w.ID(),
	}, nil
}

type CreateWalletRequest struct {
	Wallet     string `json:"wallet"`
	Passphrase string `json:"passphrase"`
}

type CreateWalletResponse struct {
	Wallet CreatedWallet  `json:"wallet"`
	Key    FirstPublicKey `json:"key"`
}

type CreatedWallet struct {
	Name           string `json:"name"`
	Version        uint32 `json:"version"`
	FilePath       string `json:"filePath"`
	RecoveryPhrase string `json:"recoveryPhrase"`
}

type FirstPublicKey struct {
	PublicKey string    `json:"publicKey"`
	Algorithm Algorithm `json:"algorithm"`
	Meta      []Meta    `json:"meta"`
}

func CreateWallet(store Store, req *CreateWalletRequest) (*CreateWalletResponse, error) {
	resp := &CreateWalletResponse{}

	if exist, err := store.WalletExists(context.Background(), req.Wallet); err != nil {
		return nil, fmt.Errorf("couldn't verify wallet existence: %w", err)
	} else if exist {
		return nil, ErrWalletAlreadyExists
	}

	w, recoveryPhrase, err := NewHDWallet(req.Wallet)
	if err != nil {
		return nil, fmt.Errorf("couldn't create HD wallet: %w", err)
	}

	kp, err := w.GenerateKeyPair(addDefaultKeyName(w, nil))
	if err != nil {
		return nil, err
	}

	if err := store.SaveWallet(context.Background(), w, req.Passphrase); err != nil {
		return nil, fmt.Errorf("couldn't save wallet: %w", err)
	}

	resp.Wallet.Name = req.Wallet
	resp.Wallet.RecoveryPhrase = recoveryPhrase
	resp.Wallet.Version = w.Version()
	resp.Wallet.FilePath = store.GetWalletPath(req.Wallet)
	resp.Key.PublicKey = kp.PublicKey()
	resp.Key.Algorithm.Name = kp.AlgorithmName()
	resp.Key.Algorithm.Version = kp.AlgorithmVersion()
	resp.Key.Meta = kp.Meta()

	return resp, nil
}

type ImportWalletRequest struct {
	Wallet         string `json:"wallet"`
	RecoveryPhrase string `json:"recoveryPhrase"`
	Version        uint32 `json:"version"`
	Passphrase     string `json:"passphrase"`
}

type ImportWalletResponse struct {
	Wallet ImportedWallet `json:"wallet"`
	Key    FirstPublicKey `json:"key"`
}

type ImportedWallet struct {
	Name     string `json:"name"`
	Version  uint32 `json:"version"`
	FilePath string `json:"filePath"`
}

func ImportWallet(store Store, req *ImportWalletRequest) (*ImportWalletResponse, error) {
	ctx := context.Background()

	if exist, err := store.WalletExists(ctx, req.Wallet); err != nil {
		return nil, fmt.Errorf("couldn't verify wallet existence: %w", err)
	} else if exist {
		return nil, ErrWalletAlreadyExists
	}

	w, err := ImportHDWallet(req.Wallet, req.RecoveryPhrase, req.Version)
	if err != nil {
		return nil, fmt.Errorf("couldn't import the wallet: %w", err)
	}

	kp, err := w.GenerateKeyPair(addDefaultKeyName(w, nil))
	if err != nil {
		return nil, err
	}

	if err := store.SaveWallet(ctx, w, req.Passphrase); err != nil {
		return nil, fmt.Errorf("couldn't save wallet: %w", err)
	}

	resp := &ImportWalletResponse{}
	resp.Wallet.Name = req.Wallet
	resp.Wallet.Version = w.Version()
	resp.Wallet.FilePath = store.GetWalletPath(req.Wallet)
	resp.Key.PublicKey = kp.PublicKey()
	resp.Key.Algorithm.Name = kp.AlgorithmName()
	resp.Key.Algorithm.Version = kp.AlgorithmVersion()
	resp.Key.Meta = kp.Meta()

	return resp, nil
}

type ListWalletsResponse struct {
	Wallets []string `json:"wallets"`
}

func ListWallets(store Store) (*ListWalletsResponse, error) {
	ws, err := store.ListWallets(context.Background())
	if err != nil {
		return nil, err
	}

	resp := &ListWalletsResponse{}
	resp.Wallets = make([]string, 0, len(ws))
	resp.Wallets = append(resp.Wallets, ws...)

	return resp, nil
}

type SignCommandRequest struct {
	Wallet        string `json:"wallet"`
	Passphrase    string `json:"passphrase"`
	TxBlockHeight uint64 `json:"txBlockHeight"`
	ChainID       string `json:"chainID"`

	Request *walletpb.SubmitTransactionRequest `json:"request"`
}

type SignCommandResponse struct {
	Base64Transaction string `json:"base64Transaction"`
}

func SignCommand(store Store, req *SignCommandRequest) (*SignCommandResponse, error) {
	w, err := getWallet(store, req.Wallet, req.Passphrase)
	if err != nil {
		return nil, err
	}

	data, err := wcommands.ToMarshaledInputData(req.Request, req.TxBlockHeight, req.ChainID)
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal input data: %w", err)
	}

	pubKey := req.Request.GetPubKey()
	signature, err := w.SignTx(pubKey, data)
	if err != nil {
		return nil, fmt.Errorf("couldn't sign transaction: %w", err)
	}

	protoSignature := &commandspb.Signature{
		Value:   signature.Value,
		Algo:    signature.Algo,
		Version: signature.Version,
	}

	tx := commands.NewTransaction(pubKey, data, protoSignature)

	rawTx, err := proto.Marshal(tx)
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal transaction: %w", err)
	}

	return &SignCommandResponse{
		Base64Transaction: base64.StdEncoding.EncodeToString(rawTx),
	}, nil
}

type SignMessageRequest struct {
	Wallet     string `json:"wallet"`
	PubKey     string `json:"pubKey"`
	Message    []byte `json:"message"`
	Passphrase string `json:"passphrase"`
}

type SignMessageResponse struct {
	Base64 string `json:"hexSignature"`
	Bytes  []byte `json:"bytesSignature"`
}

func SignMessage(store Store, req *SignMessageRequest) (*SignMessageResponse, error) {
	w, err := getWallet(store, req.Wallet, req.Passphrase)
	if err != nil {
		return nil, err
	}

	sig, err := w.SignAny(req.PubKey, req.Message)
	if err != nil {
		return nil, fmt.Errorf("couldn't sign message: %w", err)
	}

	return &SignMessageResponse{
		Base64: base64.StdEncoding.EncodeToString(sig),
		Bytes:  sig,
	}, nil
}

type ListPermissionsRequest struct {
	Wallet     string `json:"wallet"`
	Passphrase string `json:"passphrase"`
}

type ListPermissionsResponse struct {
	Hostnames []string `json:"hostnames"`
}

func ListPermissions(store Store, req *ListPermissionsRequest) (*ListPermissionsResponse, error) {
	w, err := getWallet(store, req.Wallet, req.Passphrase)
	if err != nil {
		return nil, err
	}

	return &ListPermissionsResponse{
		Hostnames: w.PermittedHostnames(),
	}, nil
}

type DescribePermissionsRequest struct {
	Wallet     string `json:"wallet"`
	Passphrase string `json:"passphrase"`
	Hostname   string `json:"hostname"`
}

type DescribePermissionsResponse struct {
	Permissions Permissions `json:"permissions"`
}

func DescribePermissions(store Store, req *DescribePermissionsRequest) (*DescribePermissionsResponse, error) {
	w, err := getWallet(store, req.Wallet, req.Passphrase)
	if err != nil {
		return nil, err
	}

	return &DescribePermissionsResponse{
		Permissions: w.Permissions(req.Hostname),
	}, nil
}

type RevokePermissionsRequest struct {
	Wallet     string `json:"wallet"`
	Passphrase string `json:"passphrase"`
	Hostname   string `json:"hostname"`
}

func RevokePermissions(store Store, req *RevokePermissionsRequest) error {
	w, err := getWallet(store, req.Wallet, req.Passphrase)
	if err != nil {
		return err
	}

	w.RevokePermissions(req.Hostname)

	if err := store.SaveWallet(context.Background(), w, req.Passphrase); err != nil {
		return fmt.Errorf("couldn't save wallet: %w", err)
	}
	return nil
}

type PurgePermissionsRequest struct {
	Wallet     string `json:"wallet"`
	Passphrase string `json:"passphrase"`
}

func PurgePermissions(store Store, req *PurgePermissionsRequest) error {
	w, err := getWallet(store, req.Wallet, req.Passphrase)
	if err != nil {
		return err
	}

	w.PurgePermissions()

	if err := store.SaveWallet(context.Background(), w, req.Passphrase); err != nil {
		return fmt.Errorf("couldn't save wallet: %w", err)
	}
	return nil
}

func getWallet(store Store, wallet, passphrase string) (Wallet, error) {
	if exist, err := store.WalletExists(context.Background(), wallet); err != nil {
		return nil, fmt.Errorf("couldn't verify wallet existence: %w", err)
	} else if !exist {
		return nil, ErrWalletDoesNotExists
	}

	w, err := store.GetWallet(context.Background(), wallet, passphrase)
	if err != nil {
		if errors.Is(err, ErrWrongPassphrase) {
			return nil, err
		}
		return nil, fmt.Errorf("couldn't get wallet %s: %w", wallet, err)
	}

	return w, nil
}

func addDefaultKeyName(w Wallet, meta []Meta) []Meta {
	for _, m := range meta {
		if m.Key == KeyNameMeta {
			return meta
		}
	}

	if len(meta) == 0 {
		meta = []Meta{}
	}

	nextID := len(w.ListKeyPairs()) + 1

	meta = append(meta, Meta{
		Key:   KeyNameMeta,
		Value: fmt.Sprintf("%s key %d", w.Name(), nextID),
	})
	return meta
}
