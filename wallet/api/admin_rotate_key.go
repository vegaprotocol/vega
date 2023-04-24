package api

import (
	"context"
	"encoding/base64"
	"fmt"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/libs/jsonrpc"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/golang/protobuf/proto"
	"github.com/mitchellh/mapstructure"
)

type AdminRotateKeyParams struct {
	Wallet                string `json:"wallet"`
	FromPublicKey         string `json:"fromPublicKey"`
	ToPublicKey           string `json:"toPublicKey"`
	ChainID               string `json:"chainID"`
	SubmissionBlockHeight uint64 `json:"submissionBlockHeight"`
	EnactmentBlockHeight  uint64 `json:"enactmentBlockHeight"`
}

type AdminRotateKeyResult struct {
	MasterPublicKey    string `json:"masterPublicKey"`
	EncodedTransaction string `json:"encodedTransaction"`
}

type AdminRotateKey struct {
	walletStore WalletStore
}

// Handle create a transaction to rotate the keys.
func (h *AdminRotateKey) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAdminRotateKeyParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the wallet exists: %w", err))
	} else if !exist {
		return nil, invalidParams(ErrWalletDoesNotExist)
	}

	alreadyUnlocked, err := h.walletStore.IsWalletAlreadyUnlocked(ctx, params.Wallet)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not verify whether the wallet is already unlock or not: %w", err))
	}
	if !alreadyUnlocked {
		return nil, requestNotPermittedError(ErrWalletIsLocked)
	}

	w, err := h.walletStore.GetWallet(ctx, params.Wallet)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not retrieve the wallet: %w", err))
	}

	if w.IsIsolated() {
		return nil, invalidParams(ErrCannotRotateKeysOnIsolatedWallet)
	}

	if !w.HasPublicKey(params.FromPublicKey) {
		return nil, invalidParams(ErrCurrentPublicKeyDoesNotExist)
	}

	if !w.HasPublicKey(params.ToPublicKey) {
		return nil, invalidParams(ErrNextPublicKeyDoesNotExist)
	}

	currentPublicKey, err := w.DescribePublicKey(params.FromPublicKey)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not retrieve the current public key: %w", err))
	}

	nextPublicKey, err := w.DescribePublicKey(params.ToPublicKey)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not retrieve the next public key: %w", err))
	}

	if nextPublicKey.IsTainted() {
		return nil, invalidParams(ErrNextPublicKeyIsTainted)
	}

	currentPubKeyHash, err := currentPublicKey.Hash()
	if err != nil {
		return nil, internalError(fmt.Errorf("could not hash the current public key: %w", err))
	}

	inputData := commands.NewInputData(params.SubmissionBlockHeight)
	inputData.Command = &commandspb.InputData_KeyRotateSubmission{
		KeyRotateSubmission: &commandspb.KeyRotateSubmission{
			NewPubKeyIndex:    nextPublicKey.Index(),
			NewPubKey:         nextPublicKey.Key(),
			TargetBlock:       params.EnactmentBlockHeight,
			CurrentPubKeyHash: currentPubKeyHash,
		},
	}

	marshaledInputData, err := commands.MarshalInputData(inputData)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not build the key rotation transaction: %w", err))
	}

	masterKey, err := w.MasterKey()
	if err != nil {
		return nil, internalError(fmt.Errorf("could not retrieve master key to sign the key rotation transaction: %w", err))
	}

	rotationSignature, err := masterKey.Sign(commands.BundleInputDataForSigning(marshaledInputData, params.ChainID))
	if err != nil {
		return nil, internalError(fmt.Errorf("could not sign the key rotation transaction: %w", err))
	}

	protoSignature := &commandspb.Signature{
		Value:   rotationSignature.Value,
		Algo:    rotationSignature.Algo,
		Version: rotationSignature.Version,
	}

	transaction := commands.NewTransaction(masterKey.PublicKey(), marshaledInputData, protoSignature)
	rawTransaction, err := proto.Marshal(transaction)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not bundle the key rotation transaction: %w", err))
	}

	return AdminRotateKeyResult{
		MasterPublicKey:    masterKey.PublicKey(),
		EncodedTransaction: base64.StdEncoding.EncodeToString(rawTransaction),
	}, nil
}

func validateAdminRotateKeyParams(rawParams jsonrpc.Params) (AdminRotateKeyParams, error) {
	if rawParams == nil {
		return AdminRotateKeyParams{}, ErrParamsRequired
	}

	params := AdminRotateKeyParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminRotateKeyParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminRotateKeyParams{}, ErrWalletIsRequired
	}

	if params.ChainID == "" {
		return AdminRotateKeyParams{}, ErrChainIDIsRequired
	}

	if params.FromPublicKey == "" {
		return AdminRotateKeyParams{}, ErrCurrentPublicKeyIsRequired
	}

	if params.ToPublicKey == "" {
		return AdminRotateKeyParams{}, ErrNextPublicKeyIsRequired
	}

	if params.ToPublicKey == params.FromPublicKey {
		return AdminRotateKeyParams{}, ErrNextAndCurrentPublicKeysCannotBeTheSame
	}

	if params.SubmissionBlockHeight == 0 {
		return AdminRotateKeyParams{}, ErrSubmissionBlockHeightIsRequired
	}

	if params.EnactmentBlockHeight == 0 {
		return AdminRotateKeyParams{}, ErrEnactmentBlockHeightIsRequired
	}

	if params.EnactmentBlockHeight <= params.SubmissionBlockHeight {
		return AdminRotateKeyParams{}, ErrEnactmentBlockHeightMustBeGreaterThanSubmissionOne
	}

	return params, nil
}

func NewAdminRotateKey(
	walletStore WalletStore,
) *AdminRotateKey {
	return &AdminRotateKey{
		walletStore: walletStore,
	}
}
