package main

import (
	"context"
	"errors"

	api "code.vegaprotocol.io/protos/vega/api/v1"
	walletpb "code.vegaprotocol.io/protos/vega/wallet/v1"
	storev1 "code.vegaprotocol.io/vegawallet/wallet/store/v1"
	wallets "code.vegaprotocol.io/vegawallet/wallets"
)

var ErrFailedSubmission = errors.New("failed to submit transaction")

type Party struct {
	wallet string
	pubkey string
}

type Wallets struct {
	handler    *wallets.Handler
	passphrase string
	parties    []*Party
}

func NewWallet(root, passphrase string) *Wallets {
	store, _ := storev1.InitialiseStore(root)
	handler := wallets.NewHandler(store)
	wallets, _ := handler.ListWallets()

	parties := make([]*Party, 0)
	for _, w := range wallets {
		handler.LoginWallet(w, passphrase)
		keys, _ := handler.ListPublicKeys(w)

		for _, k := range keys {
			parties = append(parties,
				&Party{
					wallet: w,
					pubkey: k.Key(),
				})
			break
		}
		handler.LogoutWallet(w)
	}

	return &Wallets{
		handler:    handler,
		passphrase: passphrase,
		parties:    parties,
	}
}

func (w *Wallets) Login(wallet string) {
	_ = w.handler.LoginWallet(wallet, w.passphrase)
}

func (w *Wallets) Logout(wallet string) {
	w.handler.LogoutWallet(wallet)
}

func (w *Wallets) GetParties() []*Party {
	return w.parties
}

func (w *Wallets) SubmitTransaction(
	conn *Connection,
	party *Party,
	txn *walletpb.SubmitTransactionRequest,
) error {
	blockHeight, _ := conn.LastBlockHeight()

	w.Login(party.wallet)
	defer w.Logout(party.wallet)

	// Add public key
	txn.PubKey = party.pubkey

	signedTx, err := w.handler.SignTx(party.wallet, txn, blockHeight)
	if err != nil {
		return err
	}

	submitReq := &api.SubmitTransactionRequest{
		Tx:   signedTx,
		Type: api.SubmitTransactionRequest_TYPE_SYNC,
	}
	submitResponse, err := conn.core.SubmitTransaction(context.Background(), submitReq)
	if err != nil {
		return err
	}
	if !submitResponse.Success {
		return ErrFailedSubmission
	}

	return nil
}
