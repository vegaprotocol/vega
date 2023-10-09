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

package v1

import (
	"context"

	api "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	walletpb "code.vegaprotocol.io/vega/protos/vega/wallet/v1"
	nodetypes "code.vegaprotocol.io/vega/wallet/api/node/types"
	"code.vegaprotocol.io/vega/wallet/wallet"
)

// Generates mocks
//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/wallet/service/v1 WalletHandler,Auth,NodeForward,RSAStore,SpamHandler

//nolint:interfacebloat
type WalletHandler interface {
	CreateWallet(name, passphrase string) (string, error)
	ImportWallet(name, passphrase, recoveryPhrase string, version uint32) error
	LoginWallet(name, passphrase string) error
	SecureGenerateKeyPair(name, passphrase string, meta []wallet.Metadata) (string, error)
	GetPublicKey(name, pubKey string) (wallet.PublicKey, error)
	ListPublicKeys(name string) ([]wallet.PublicKey, error)
	SignTx(name string, req *walletpb.SubmitTransactionRequest, height uint64, chainID string) (*commandspb.Transaction, error)
	SignAny(name string, inputData []byte, pubKey string) ([]byte, error)
	VerifyAny(inputData, sig []byte, pubKey string) (bool, error)
	TaintKey(name, pubKey, passphrase string) error
	UpdateMeta(name, pubKey, passphrase string, meta []wallet.Metadata) error
}

type Auth interface {
	NewSession(name string) (string, error)
	VerifyToken(token string) (string, error)
	Revoke(token string) (string, error)
	RevokeAllToken()
}

type NodeForward interface {
	SendTx(context.Context, *commandspb.Transaction, api.SubmitTransactionRequest_Type, int) (*api.SubmitTransactionResponse, error)
	CheckTx(context.Context, *commandspb.Transaction, int) (*api.CheckTransactionResponse, error)
	HealthCheck(context.Context) error
	LastBlockHeightAndHash(context.Context) (*api.LastBlockHeightResponse, int, error)
	SpamStatistics(context.Context, string) (*api.GetSpamStatisticsResponse, int, error)
	Stop()
}

type RSAStore interface {
	GetRsaKeys() (*RSAKeys, error)
}

type SpamHandler interface {
	GenerateProofOfWork(pubKey string, stats *nodetypes.SpamStatistics) (*commandspb.ProofOfWork, error)
	CheckSubmission(req *walletpb.SubmitTransactionRequest, stats *nodetypes.SpamStatistics) error
}
