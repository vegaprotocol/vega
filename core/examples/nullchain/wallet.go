// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package nullchain

import (
	"context"
	"errors"
	"fmt"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
	api "code.vegaprotocol.io/vega/protos/vega/api/v1"
	walletpb "code.vegaprotocol.io/vega/protos/vega/wallet/v1"
	storev1 "code.vegaprotocol.io/vega/wallet/wallet/store/v1"
	"code.vegaprotocol.io/vega/wallet/wallets"
)

var ErrFailedSubmission = errors.New("failed to submit transaction")

type Party struct {
	wallet string
	pubkey string
}

type Wallet struct {
	handler    *wallets.Handler
	store      *storev1.FileStore
	passphrase string
}

func NewWallet(root, passphrase string) *Wallet {
	store, err := storev1.InitialiseStore(root)
	if err != nil {
		panic(fmt.Errorf("could not initialise the wallet store: %w", err))
	}

	return &Wallet{
		handler:    wallets.NewHandler(store),
		store:      store,
		passphrase: passphrase,
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
		if err := w.handler.LoginWallet(walletName, passphrase); err != nil {
			return nil, err
		}

		kp, err := w.handler.GenerateKeyPair(walletName, passphrase, nil)
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
	ctx := context.Background()
	for _, party := range party {
		_ = w.store.DeleteWallet(ctx, party.wallet)
	}
}

func (w *Wallet) Login(wallet string) {
	_ = w.handler.LoginWallet(wallet, w.passphrase)
}

func (w *Wallet) SubmitTransaction(conn *Connection, party *Party, txn *walletpb.SubmitTransactionRequest) error {
	blockHeight, _ := conn.LastBlockHeight()

	w.Login(party.wallet)

	// Add public key to the transaction
	txn.PubKey = party.pubkey

	chainID, err := conn.NetworkChainID()
	if err != nil {
		return err
	}

	signedTx, err := w.handler.SignTx(party.wallet, txn, blockHeight, chainID)
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

func (w *Wallet) Close() {
	w.store.Close()
}
