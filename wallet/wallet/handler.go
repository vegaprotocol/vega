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
		return nil, fmt.Errorf("could not marshal the input data: %w", err)
	}

	pubKey := req.Request.GetPubKey()
	signature, err := w.SignTx(pubKey, data)
	if err != nil {
		return nil, fmt.Errorf("could not sign the transaction: %w", err)
	}

	protoSignature := &commandspb.Signature{
		Value:   signature.Value,
		Algo:    signature.Algo,
		Version: signature.Version,
	}

	tx := commands.NewTransaction(pubKey, data, protoSignature)

	rawTx, err := proto.Marshal(tx)
	if err != nil {
		return nil, fmt.Errorf("could not marshal the transaction: %w", err)
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
		return nil, fmt.Errorf("could not sign the message: %w", err)
	}

	return &SignMessageResponse{
		Base64: base64.StdEncoding.EncodeToString(sig),
		Bytes:  sig,
	}, nil
}

func getWallet(store Store, wallet, passphrase string) (Wallet, error) {
	if exist, err := store.WalletExists(context.Background(), wallet); err != nil {
		return nil, fmt.Errorf("could not verify the wallet existence: %w", err)
	} else if !exist {
		return nil, ErrWalletDoesNotExist
	}

	w, err := store.GetWallet(context.Background(), wallet, passphrase)
	if err != nil {
		if errors.Is(err, ErrWrongPassphrase) {
			return nil, err
		}
		return nil, fmt.Errorf("could not retrieve the wallet %s: %w", wallet, err)
	}

	return w, nil
}
