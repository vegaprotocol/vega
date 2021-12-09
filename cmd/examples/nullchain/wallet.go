package nullchain

import (
	"context"
	"errors"

	api "code.vegaprotocol.io/protos/vega/api/v1"
	walletpb "code.vegaprotocol.io/protos/vega/wallet/v1"
	vgrand "code.vegaprotocol.io/shared/libs/rand"
	storev1 "code.vegaprotocol.io/vegawallet/wallet/store/v1"
	wallets "code.vegaprotocol.io/vegawallet/wallets"
)

var ErrFailedSubmission = errors.New("failed to submit transaction")

type Party struct {
	wallet string
	pubkey string
}

type Wallet struct {
	handler    *wallets.Handler
	store      *storev1.Store
	passphrase string
	parties    []*Party
}

func NewWallet(root, passphrase string) *Wallet {
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

	return &Wallet{
		handler:    handler,
		store:      store,
		passphrase: passphrase,
		parties:    parties,
	}
}

func (w *Wallet) MakeParties(n uint64) ([]*Party, error) {
	parties := make([]*Party, 0, n)

	var err error
	defer func() {
		if err != nil {
			w.DeleteParties(parties)
		}
	}()
	// make n wallet's each with a single key
	passphrase := "pin"

	for i := uint64(0); i < n; i++ {
		walletName := vgrand.RandomStr(10)
		if _, err = w.handler.CreateWallet(walletName, passphrase); err != nil {
			return nil, err
		}
		w.handler.LoginWallet(walletName, passphrase)
		kp, err := w.handler.GenerateKeyPair(walletName, passphrase, nil)
		w.handler.LogoutWallet(walletName)

		if err != nil {
			return nil, err
		}

		parties = append(parties, &Party{
			wallet: walletName,
			pubkey: kp.PublicKey(),
		})
	}

	return parties, nil
}

func (w *Wallet) DeleteParties(party []*Party) {
	for _, party := range party {
		w.store.DeleteWallet(party.wallet)
	}
}

func (w *Wallet) Login(wallet string) {
	_ = w.handler.LoginWallet(wallet, w.passphrase)
}

func (w *Wallet) Logout(wallet string) {
	w.handler.LogoutWallet(wallet)
}

func (w *Wallet) GetParties() []*Party {
	return w.parties
}

func (w *Wallet) SubmitTransaction(conn *Connection, party *Party, txn *walletpb.SubmitTransactionRequest) error {
	blockHeight, _ := conn.LastBlockHeight()

	w.Login(party.wallet)
	defer w.Logout(party.wallet)

	// Add public key to the transaction
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
