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

package pkg

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/commands"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	walletpb "code.vegaprotocol.io/vega/protos/vega/wallet/v1"
	"code.vegaprotocol.io/vega/wallet/api"
	wcommands "code.vegaprotocol.io/vega/wallet/commands"
	"code.vegaprotocol.io/vega/wallet/wallet"
)

type Node interface {
	SendTransaction(context.Context, *commandspb.Transaction, apipb.SubmitTransactionRequest_Type) (*apipb.SubmitTransactionResponse, error)
	LastBlock(context.Context) (*apipb.LastBlockHeightResponse, error)
}

func SendTransaction(ctx context.Context, w wallet.Wallet, pubKey string, request *walletpb.SubmitTransactionRequest, node Node) (*apipb.SubmitTransactionResponse, error) {
	request.PubKey = pubKey
	request.Propagate = true
	if errs := wcommands.CheckSubmitTransactionRequest(request); !errs.Empty() {
		return nil, errs
	}

	lastBlockData, err := node.LastBlock(ctx)
	if err != nil {
		return nil, api.ErrCouldNotGetLastBlockInformation
	}

	marshaledInputData, err := wcommands.ToMarshaledInputData(request, lastBlockData.Height)
	if err != nil {
		return nil, fmt.Errorf("could not marshal the input data: %w", err)
	}

	signature, err := w.SignTx(pubKey, commands.BundleInputDataForSigning(marshaledInputData, lastBlockData.ChainId))
	if err != nil {
		return nil, fmt.Errorf("could not sign the transaction: %w", err)
	}

	// Build the transaction.
	tx := commands.NewTransaction(pubKey, marshaledInputData, &commandspb.Signature{
		Value:   signature.Value,
		Algo:    signature.Algo,
		Version: signature.Version,
	})

	// Generate the proof of work for the transaction.
	txID := vgcrypto.RandomHash()
	powNonce, _, err := vgcrypto.PoW(lastBlockData.Hash, txID, uint(lastBlockData.SpamPowDifficulty), lastBlockData.SpamPowHashFunction)
	if err != nil {
		return nil, fmt.Errorf("could not compute the proof-of-work: %w", err)
	}

	tx.Pow = &commandspb.ProofOfWork{
		Nonce: powNonce,
		Tid:   txID,
	}

	result, err := node.SendTransaction(ctx, tx, apipb.SubmitTransactionRequest_TYPE_SYNC)
	if err != nil {
		return nil, err
	}

	return result, nil
}
