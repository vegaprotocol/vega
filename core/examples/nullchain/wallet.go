// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
	store, err := storev1.InitialiseStore(root, false)
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
