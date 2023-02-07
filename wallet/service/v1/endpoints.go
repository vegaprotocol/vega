package v1

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/commands"
	vfmt "code.vegaprotocol.io/vega/libs/fmt"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	api "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	walletpb "code.vegaprotocol.io/vega/protos/vega/wallet/v1"
	"code.vegaprotocol.io/vega/version"
	nodetypes "code.vegaprotocol.io/vega/wallet/api/node/types"
	wcommands "code.vegaprotocol.io/vega/wallet/commands"
	"code.vegaprotocol.io/vega/wallet/network"
	"code.vegaprotocol.io/vega/wallet/wallet"

	"github.com/golang/protobuf/jsonpb"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

const (
	TxnValidationFailure   uint32 = 51
	TxnDecodingFailure     uint32 = 60
	TxnInternalError       uint32 = 70
	TxnUnknownCommandError uint32 = 80
	TxnSpamError           uint32 = 89
)

// CreateWalletRequest describes the request for CreateWallet.
type CreateWalletRequest struct {
	Wallet     string `json:"wallet"`
	Passphrase string `json:"passphrase"`
}

const TXIDLENGTH = 20

func ParseCreateWalletRequest(r *http.Request) (*CreateWalletRequest, commands.Errors) {
	errs := commands.NewErrors()

	req := &CreateWalletRequest{}
	if err := unmarshalBody(r, &req); err != nil {
		return nil, errs.FinalAdd(err)
	}

	if len(req.Wallet) == 0 {
		errs.AddForProperty("wallet", commands.ErrIsRequired)
	}

	if len(req.Passphrase) == 0 {
		errs.AddForProperty("passphrase", commands.ErrIsRequired)
	}

	if !errs.Empty() {
		return nil, errs
	}

	return req, errs
}

// CreateWalletResponse returns the authentication token and the auto-generated
// recovery phrase of the created wallet.
type CreateWalletResponse struct {
	RecoveryPhrase string `json:"recoveryPhrase"`
	Token          string `json:"token"`
}

// ImportWalletRequest describes the request for ImportWallet.
type ImportWalletRequest struct {
	Wallet         string `json:"wallet"`
	Passphrase     string `json:"passphrase"`
	RecoveryPhrase string `json:"recoveryPhrase"`
	Version        uint32 `json:"version"`
}

func ParseImportWalletRequest(r *http.Request) (*ImportWalletRequest, commands.Errors) {
	errs := commands.NewErrors()

	req := &ImportWalletRequest{}
	if err := unmarshalBody(r, &req); err != nil {
		return nil, errs.FinalAdd(err)
	}

	if len(req.Wallet) == 0 {
		errs.AddForProperty("wallet", commands.ErrIsRequired)
	}

	if len(req.Passphrase) == 0 {
		errs.AddForProperty("passphrase", commands.ErrIsRequired)
	}

	if len(req.RecoveryPhrase) == 0 {
		errs.AddForProperty("recoveryPhrase", commands.ErrIsRequired)
	}

	if req.Version == 0 {
		req.Version = wallet.LatestVersion
	}

	if !errs.Empty() {
		return nil, errs
	}

	return req, errs
}

// LoginWalletRequest describes the request for CreateWallet, LoginWallet.
type LoginWalletRequest struct {
	Wallet     string `json:"wallet"`
	Passphrase string `json:"passphrase"`
}

func ParseLoginWalletRequest(r *http.Request) (*LoginWalletRequest, commands.Errors) {
	errs := commands.NewErrors()

	req := &LoginWalletRequest{}
	if err := unmarshalBody(r, &req); err != nil {
		return nil, errs.FinalAdd(err)
	}

	if len(req.Wallet) == 0 {
		errs.AddForProperty("wallet", commands.ErrIsRequired)
	}

	if len(req.Passphrase) == 0 {
		errs.AddForProperty("passphrase", commands.ErrIsRequired)
	}

	if !errs.Empty() {
		return nil, errs
	}

	return req, errs
}

// TaintKeyRequest describes the request for TaintKey.
type TaintKeyRequest struct {
	Passphrase string `json:"passphrase"`
}

func ParseTaintKeyRequest(r *http.Request, keyID string) (*TaintKeyRequest, commands.Errors) {
	errs := commands.NewErrors()

	if len(keyID) == 0 {
		errs.AddForProperty("keyid", commands.ErrIsRequired)
	}

	req := &TaintKeyRequest{}
	if err := unmarshalBody(r, &req); err != nil {
		return nil, errs.FinalAdd(err)
	}

	if len(req.Passphrase) == 0 {
		errs.AddForProperty("passphrase", commands.ErrIsRequired)
	}

	if !errs.Empty() {
		return nil, errs
	}

	return req, errs
}

// GenKeyPairRequest describes the request for GenerateKeyPair.
type GenKeyPairRequest struct {
	Passphrase string            `json:"passphrase"`
	Meta       []wallet.Metadata `json:"meta"`
}

func ParseGenKeyPairRequest(r *http.Request) (*GenKeyPairRequest, commands.Errors) {
	errs := commands.NewErrors()

	req := &GenKeyPairRequest{}
	if err := unmarshalBody(r, &req); err != nil {
		return nil, errs.FinalAdd(err)
	}

	if len(req.Passphrase) == 0 {
		errs.AddForProperty("passphrase", commands.ErrIsRequired)
	}

	if !errs.Empty() {
		return nil, errs
	}

	return req, errs
}

// UpdateMetaRequest describes the request for UpdateMetadata.
type UpdateMetaRequest struct {
	Passphrase string            `json:"passphrase"`
	Meta       []wallet.Metadata `json:"meta"`
}

func ParseUpdateMetaRequest(r *http.Request, keyID string) (*UpdateMetaRequest, commands.Errors) {
	errs := commands.NewErrors()

	if len(keyID) == 0 {
		errs.AddForProperty("keyid", commands.ErrIsRequired)
	}

	req := &UpdateMetaRequest{}
	if err := unmarshalBody(r, &req); err != nil {
		return nil, errs.FinalAdd(err)
	}

	if len(req.Passphrase) == 0 {
		errs.AddForProperty("passphrase", commands.ErrIsRequired)
	}

	if !errs.Empty() {
		return nil, errs
	}

	return req, errs
}

// SignAnyRequest describes the request for SignAny.
type SignAnyRequest struct {
	// InputData is the payload to generate a signature from. I should be
	// base 64 encoded.
	InputData string `json:"inputData"`
	// PubKey is used to retrieve the private key to sign the InputDate.
	PubKey string `json:"pubKey"`

	decodedInputData []byte
}

func ParseSignAnyRequest(r *http.Request) (*SignAnyRequest, commands.Errors) {
	errs := commands.NewErrors()

	req := &SignAnyRequest{}
	if err := unmarshalBody(r, &req); err != nil {
		return nil, errs.FinalAdd(err)
	}

	if len(req.InputData) == 0 {
		errs.AddForProperty("inputData", commands.ErrIsRequired)
	}
	decodedInputData, err := base64.StdEncoding.DecodeString(req.InputData)
	if err != nil {
		errs.AddForProperty("inputData", ErrShouldBeBase64Encoded)
	} else {
		req.decodedInputData = decodedInputData
	}

	if len(req.PubKey) == 0 {
		errs.AddForProperty("pubKey", commands.ErrIsRequired)
	}

	if !errs.Empty() {
		return nil, errs
	}

	return req, errs
}

// VerifyAnyRequest describes the request for VerifyAny.
type VerifyAnyRequest struct {
	// InputData is the payload to be verified. It should be base64 encoded.
	InputData string `json:"inputData"`
	// Signature is the signature to check against the InputData. It should be
	// base64 encoded.
	Signature string `json:"signature"`
	// PubKey is the public key used along the signature to check the InputData.
	PubKey string `json:"pubKey"`

	decodedInputData []byte
	decodedSignature []byte
}

func ParseVerifyAnyRequest(r *http.Request) (*VerifyAnyRequest, commands.Errors) {
	errs := commands.NewErrors()

	req := &VerifyAnyRequest{}
	if err := unmarshalBody(r, &req); err != nil {
		return nil, errs.FinalAdd(err)
	}

	if len(req.InputData) == 0 {
		errs.AddForProperty("inputData", commands.ErrIsRequired)
	} else {
		decodedInputData, err := base64.StdEncoding.DecodeString(req.InputData)
		if err != nil {
			errs.AddForProperty("inputData", ErrShouldBeBase64Encoded)
		} else {
			req.decodedInputData = decodedInputData
		}
	}

	if len(req.Signature) == 0 {
		errs.AddForProperty("signature", commands.ErrIsRequired)
	} else {
		decodedSignature, err := base64.StdEncoding.DecodeString(req.Signature)
		if err != nil {
			errs.AddForProperty("signature", ErrShouldBeBase64Encoded)
		} else {
			req.decodedSignature = decodedSignature
		}
	}

	if len(req.PubKey) == 0 {
		errs.AddForProperty("pubKey", commands.ErrIsRequired)
	}

	if !errs.Empty() {
		return nil, errs
	}

	return req, nil
}

func ParseSubmitTransactionRequest(r *http.Request) (*walletpb.SubmitTransactionRequest, commands.Errors) {
	errs := commands.NewErrors()

	req := &walletpb.SubmitTransactionRequest{
		Propagate: true,
	}
	if err := jsonpb.Unmarshal(r.Body, req); err != nil {
		return nil, errs.FinalAdd(err)
	}

	if errs = wcommands.CheckSubmitTransactionRequest(req); !errs.Empty() {
		return nil, errs
	}

	return req, nil
}

// KeyResponse describes the response to a request that returns a single key.
type KeyResponse struct {
	Key KeyKeyResponse `json:"key"`
}

type KeyKeyResponse struct {
	Idx          uint32            `json:"index"`
	PublicKey    string            `json:"pub"`
	KeyName      string            `json:"name"`
	Algorithm    wallet.Algorithm  `json:"algorithm"`
	Tainted      bool              `json:"tainted"`
	MetadataList []wallet.Metadata `json:"meta"`
}

// KeysResponse describes the response to a request that returns a list of keys.
type KeysResponse struct {
	Keys []KeyKeyResponse `json:"keys"`
}

// SignAnyResponse describes the response for SignAny.
type SignAnyResponse struct {
	HexSignature    string `json:"hexSignature"`
	Base64Signature string `json:"base64Signature"`
}

// VerifyAnyResponse describes the response for VerifyAny.
type VerifyAnyResponse struct {
	Valid bool `json:"success"`
}

// SuccessResponse describes the response to a request that returns a simple true/false answer.
type SuccessResponse struct {
	Success bool `json:"success"`
}

// TokenResponse describes the response to a request that returns a token.
type TokenResponse struct {
	Token string `json:"token"`
}

// VersionResponse describes the response to a request that returns app version info.
type VersionResponse struct {
	Version     string `json:"version"`
	VersionHash string `json:"versionHash"`
}

// NetworkResponse describes the response to a request that returns app hosts info.
type NetworkResponse struct {
	Network network.Network `json:"network"`
}

func (s *API) CreateWallet(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	req, errs := ParseCreateWalletRequest(r)
	if !errs.Empty() {
		s.writeBadRequest(w, errs)
		return
	}

	recoveryPhrase, err := s.handler.CreateWallet(req.Wallet, req.Passphrase)
	if err != nil {
		s.writeBadRequestErr(w, err)
		return
	}

	token, err := s.auth.NewSession(req.Wallet)
	if err != nil {
		s.writeInternalError(w, err)
		return
	}

	s.writeSuccess(w, CreateWalletResponse{
		RecoveryPhrase: recoveryPhrase,
		Token:          token,
	})
}

func (s *API) ImportWallet(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	req, errs := ParseImportWalletRequest(r)
	if !errs.Empty() {
		s.writeBadRequest(w, errs)
		return
	}

	err := s.handler.ImportWallet(req.Wallet, req.Passphrase, req.RecoveryPhrase, req.Version)
	if err != nil {
		s.writeBadRequestErr(w, err)
		return
	}

	token, err := s.auth.NewSession(req.Wallet)
	if err != nil {
		s.writeInternalError(w, err)
		return
	}

	s.writeSuccess(w, TokenResponse{Token: token})
}

func (s *API) Login(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	req, errs := ParseLoginWalletRequest(r)
	if !errs.Empty() {
		s.writeBadRequest(w, errs)
		return
	}

	err := s.handler.LoginWallet(req.Wallet, req.Passphrase)
	if err != nil {
		s.writeForbiddenError(w, err)
		return
	}

	token, err := s.auth.NewSession(req.Wallet)
	if err != nil {
		s.writeInternalError(w, err)
		return
	}

	s.writeSuccess(w, TokenResponse{Token: token})
}

func (s *API) Revoke(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	token, err := extractToken(r)
	if err != nil {
		writeError(w, err)
		return
	}

	if _, err := s.auth.Revoke(token); err != nil {
		s.writeForbiddenError(w, err)
		return
	}

	s.writeSuccess(w, nil)
}

func (s *API) GenerateKeyPair(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	token, err := extractToken(r)
	if err != nil {
		writeError(w, err)
		return
	}

	req, errs := ParseGenKeyPairRequest(r)
	if !errs.Empty() {
		s.writeBadRequest(w, errs)
		return
	}

	name, err := s.auth.VerifyToken(token)
	if err != nil {
		s.writeForbiddenError(w, err)
		return
	}

	pubKey, err := s.handler.SecureGenerateKeyPair(name, req.Passphrase, req.Meta)
	if err != nil {
		if errors.Is(err, wallet.ErrWrongPassphrase) {
			s.writeForbiddenError(w, err)
		} else {
			s.writeInternalError(w, err)
		}
		return
	}

	key, err := s.handler.GetPublicKey(name, pubKey)
	if err != nil {
		s.writeInternalError(w, err)
		return
	}

	s.writeSuccess(w, KeyResponse{
		Key: KeyKeyResponse{
			Idx:       key.Index(),
			PublicKey: key.Key(),
			KeyName:   key.Name(),
			Algorithm: wallet.Algorithm{
				Name:    key.AlgorithmName(),
				Version: key.AlgorithmVersion(),
			},
			Tainted:      key.IsTainted(),
			MetadataList: key.Metadata(),
		},
	})
}

func (s *API) GetPublicKey(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	token, err := extractToken(r)
	if err != nil {
		writeError(w, err)
		return
	}

	name, err := s.auth.VerifyToken(token)
	if err != nil {
		s.writeForbiddenError(w, err)
		return
	}

	key, err := s.handler.GetPublicKey(name, ps.ByName("keyid"))
	if err != nil {
		var statusCode int
		if errors.Is(err, wallet.ErrPubKeyDoesNotExist) {
			statusCode = http.StatusNotFound
		} else {
			statusCode = http.StatusInternalServerError
		}
		s.writeError(w, newErrorResponse(err.Error()), statusCode)
		return
	}

	s.writeSuccess(w, KeyResponse{
		Key: KeyKeyResponse{
			Idx:       key.Index(),
			PublicKey: key.Key(),
			KeyName:   key.Name(),
			Algorithm: wallet.Algorithm{
				Name:    key.AlgorithmName(),
				Version: key.AlgorithmVersion(),
			},
			Tainted:      key.IsTainted(),
			MetadataList: key.Metadata(),
		},
	})
}

func (s *API) ListPublicKeys(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	token, err := extractToken(r)
	if err != nil {
		writeError(w, err)
		return
	}

	name, err := s.auth.VerifyToken(token)
	if err != nil {
		s.writeForbiddenError(w, err)
		return
	}

	keys, err := s.handler.ListPublicKeys(name)
	if err != nil {
		s.writeInternalError(w, err)
		return
	}

	res := make([]KeyKeyResponse, 0, len(keys))
	for _, key := range keys {
		res = append(res, KeyKeyResponse{
			Idx:       key.Index(),
			PublicKey: key.Key(),
			KeyName:   key.Name(),
			Algorithm: wallet.Algorithm{
				Name:    key.AlgorithmName(),
				Version: key.AlgorithmVersion(),
			},
			Tainted:      key.IsTainted(),
			MetadataList: key.Metadata(),
		})
	}

	s.writeSuccess(w, KeysResponse{Keys: res})
}

func (s *API) TaintKey(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	token, err := extractToken(r)
	if err != nil {
		writeError(w, err)
		return
	}

	keyID := ps.ByName("keyid")
	req, errs := ParseTaintKeyRequest(r, keyID)
	if !errs.Empty() {
		s.writeBadRequest(w, errs)
		return
	}

	name, err := s.auth.VerifyToken(token)
	if err != nil {
		s.writeForbiddenError(w, err)
		return
	}

	if err = s.handler.TaintKey(name, keyID, req.Passphrase); err != nil {
		if errors.Is(err, wallet.ErrWrongPassphrase) {
			s.writeForbiddenError(w, err)
		} else {
			s.writeInternalError(w, err)
		}
		return
	}

	s.writeSuccess(w, nil)
}

func (s *API) UpdateMeta(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	token, err := extractToken(r)
	if err != nil {
		writeError(w, err)
		return
	}

	keyID := ps.ByName("keyid")
	req, errs := ParseUpdateMetaRequest(r, keyID)
	if !errs.Empty() {
		s.writeBadRequest(w, errs)
		return
	}

	name, err := s.auth.VerifyToken(token)
	if err != nil {
		s.writeForbiddenError(w, err)
		return
	}

	if err = s.handler.UpdateMeta(name, keyID, req.Passphrase, req.Meta); err != nil {
		if errors.Is(err, wallet.ErrWrongPassphrase) {
			s.writeForbiddenError(w, err)
		} else {
			s.writeInternalError(w, err)
		}
		return
	}

	s.writeSuccess(w, nil)
}

func (s *API) SignAny(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	token, err := extractToken(r)
	if err != nil {
		writeError(w, err)
		return
	}

	req, errs := ParseSignAnyRequest(r)
	if !errs.Empty() {
		s.writeBadRequest(w, errs)
		return
	}

	name, err := s.auth.VerifyToken(token)
	if err != nil {
		s.writeForbiddenError(w, err)
		return
	}

	signature, err := s.handler.SignAny(name, req.decodedInputData, req.PubKey)
	if err != nil {
		s.writeInternalError(w, err)
		return
	}

	res := SignAnyResponse{
		HexSignature:    hex.EncodeToString(signature),
		Base64Signature: base64.StdEncoding.EncodeToString(signature),
	}

	s.writeSuccess(w, res)
}

func (s *API) VerifyAny(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	req, errs := ParseVerifyAnyRequest(r)
	if !errs.Empty() {
		s.writeBadRequest(w, errs)
		return
	}

	verified, err := s.handler.VerifyAny(req.decodedInputData, req.decodedSignature, req.PubKey)
	if err != nil {
		s.writeInternalError(w, err)
		return
	}

	s.writeSuccess(w, VerifyAnyResponse{Valid: verified})
}

func (s *API) CheckTx(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()

	token, err := extractToken(r)
	if err != nil {
		writeError(w, err)
		return
	}

	name, err := s.auth.VerifyToken(token)
	if err != nil {
		s.writeForbiddenError(w, err)
		return
	}

	req, errs := ParseSubmitTransactionRequest(r)
	if !errs.Empty() {
		s.writeBadRequest(w, errs)
		return
	}

	stats, cltIdx, err := s.nodeForward.SpamStatistics(r.Context(), req.PubKey)
	if err != nil {
		s.writeInternalError(w, ErrCouldNotGetBlockHeight)
		return
	}

	if stats.ChainId == "" {
		s.writeInternalError(w, ErrCouldNotGetChainID)
		return
	}
	st := convertSpamStatistics(stats)
	tx, err := s.handler.SignTx(name, req, st.LastBlockHeight, stats.ChainId)
	if err != nil {
		s.writeInternalError(w, err)
		return
	}

	// generate proof of work for the transaction
	tx.Pow, err = s.spam.GenerateProofOfWork(req.PubKey, st)
	if err != nil {
		s.writeInternalError(w, err)
		return
	}

	result, err := s.nodeForward.CheckTx(r.Context(), tx, cltIdx)
	if err != nil {
		s.writeInternalError(w, err)
		return
	}

	s.writeSuccess(w, struct {
		Success   bool                    `json:"success"`
		Code      uint32                  `json:"code"`
		GasWanted int64                   `json:"gas_wanted"`
		GasUsed   int64                   `json:"gas_used"`
		Tx        *commandspb.Transaction `json:"tx"`
	}{
		Success:   result.Success,
		Code:      result.Code,
		GasWanted: result.GasWanted,
		GasUsed:   result.GasUsed,
		Tx:        tx,
	})
}

func (s *API) SignTxSync(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	s.signTx(w, r, p, api.SubmitTransactionRequest_TYPE_SYNC)
}

func (s *API) SignTxCommit(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	s.signTx(w, r, p, api.SubmitTransactionRequest_TYPE_COMMIT)
}

func (s *API) SignTx(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	s.signTx(w, r, p, api.SubmitTransactionRequest_TYPE_ASYNC)
}

func (s *API) signTx(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ty api.SubmitTransactionRequest_Type) {
	defer r.Body.Close()

	token, err := extractToken(r)
	if err != nil {
		writeError(w, err)
		return
	}

	name, err := s.auth.VerifyToken(token)
	if err != nil {
		s.writeForbiddenError(w, err)
		return
	}

	req, errs := ParseSubmitTransactionRequest(r)
	if !errs.Empty() {
		s.writeBadRequest(w, errs)
		return
	}

	txID := vgrand.RandomStr(TXIDLENGTH)
	receivedAt := time.Now()
	approved, err := s.policy.Ask(req, txID, receivedAt)
	if err != nil {
		s.log.Error("couldn't get user consent", zap.Error(err))
		s.writeError(w, err, http.StatusServiceUnavailable)
		return
	}

	if !approved {
		s.log.Info("user rejected transaction signing request", zap.Any("request", req))
		s.writeError(w, ErrRejectedSignRequest, http.StatusUnauthorized)
		return
	}
	s.log.Info("user approved transaction signing request", zap.Any("request", req))

	stats, cltIdx, err := s.nodeForward.SpamStatistics(r.Context(), req.PubKey)
	ss := convertSpamStatistics(stats)
	if err != nil || ss.ChainID == "" {
		s.policy.Report(SentTransaction{
			TxID:  txID,
			Error: ErrCouldNotGetChainID,
		})
		s.writeInternalError(w, ErrCouldNotGetChainID)
		return
	}
	tx, err := s.handler.SignTx(name, req, ss.LastBlockHeight, ss.ChainID)
	if err != nil {
		s.policy.Report(SentTransaction{
			TxID:  txID,
			Error: err,
		})
		s.writeInternalError(w, err)
		return
	}

	// generate proof of work for the transaction
	tx.Pow, err = s.spam.GenerateProofOfWork(req.PubKey, ss)
	if err != nil {
		s.policy.Report(SentTransaction{
			Tx:    tx,
			TxID:  txID,
			Error: err,
		})
		s.writeInternalError(w, err)
		return
	}
	sentAt := time.Now()
	resp, err := s.nodeForward.SendTx(r.Context(), tx, ty, cltIdx)
	if err != nil {
		s.policy.Report(SentTransaction{
			Tx:     tx,
			TxID:   txID,
			Error:  err,
			SentAt: sentAt,
		})
		s.writeInternalError(w, err)
		return
	}
	if !resp.Success {
		s.policy.Report(SentTransaction{
			Tx:     tx,
			TxID:   txID,
			Error:  errors.New(resp.Data),
			SentAt: sentAt,
		})
		s.writeTxError(w, resp)
		return
	}

	s.policy.Report(SentTransaction{
		TxHash: resp.TxHash,
		TxID:   txID,
		Tx:     tx,
		SentAt: sentAt,
	})

	s.writeSuccess(w, struct {
		TxHash     string                  `json:"txHash"`
		ReceivedAt time.Time               `json:"receivedAt"`
		SentAt     time.Time               `json:"sentAt"`
		TxID       string                  `json:"txId"`
		Tx         *commandspb.Transaction `json:"tx"`
	}{
		TxHash:     resp.TxHash,
		ReceivedAt: receivedAt,
		SentAt:     sentAt,
		TxID:       txID,
		Tx:         tx,
	})
}

func (s *API) Version(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	res := VersionResponse{
		Version:     version.Get(),
		VersionHash: version.GetCommitHash(),
	}

	s.writeSuccess(w, res)
}

func (s *API) GetNetwork(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	res := NetworkResponse{
		Network: *s.network,
	}
	s.writeSuccess(w, res)
}

func (s *API) Health(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if err := s.nodeForward.HealthCheck(r.Context()); err != nil {
		s.writeError(w, newErrorResponse(err.Error()), http.StatusFailedDependency)
		return
	}
	s.writeSuccess(w, nil)
}

func (s *API) GetNetworkChainID(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	lastBlock, _, err := s.nodeForward.LastBlockHeightAndHash(r.Context())
	if err != nil {
		s.writeError(w, newErrorResponse(err.Error()), http.StatusFailedDependency)
		return
	}
	s.writeSuccess(w, struct {
		ChainID string `json:"chainID"`
	}{
		ChainID: lastBlock.ChainId,
	})
}

func (s *API) writeBadRequestErr(w http.ResponseWriter, err error) {
	errs := commands.NewErrors()
	s.writeErrors(w, http.StatusBadRequest, errs.FinalAdd(err))
}

func (s *API) writeBadRequest(w http.ResponseWriter, errs commands.Errors) {
	s.writeErrors(w, http.StatusBadRequest, errs)
}

func (s *API) writeErrors(w http.ResponseWriter, statusCode int, errs commands.Errors) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	buf, err := json.Marshal(ErrorsResponse{Errors: errs})
	if err != nil {
		s.log.Error("couldn't marshal errors", zap.String("error", vfmt.Escape(errs.Error())))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(buf); err != nil {
		s.log.Error("couldn't write errors", zap.String("error", vfmt.Escape(errs.Error())))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.log.Info(fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)))
}

func unmarshalBody(r *http.Request, into interface{}) error {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return ErrCouldNotReadRequest
	}
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, into)
}

func (s *API) writeForbiddenError(w http.ResponseWriter, e error) {
	s.writeError(w, newErrorResponse(e.Error()), http.StatusForbidden)
}

func (s *API) writeInternalError(w http.ResponseWriter, e error) {
	s.writeError(w, newErrorResponse(e.Error()), http.StatusInternalServerError)
}

func (s *API) writeTxError(w http.ResponseWriter, r *api.SubmitTransactionResponse) {
	var code int
	switch r.Code {
	case TxnSpamError:
		code = http.StatusTooManyRequests
	case TxnUnknownCommandError, TxnValidationFailure, TxnDecodingFailure:
		code = http.StatusBadRequest
	case TxnInternalError:
		code = http.StatusInternalServerError
	default:
		s.log.Error("unknown transaction code", zap.Uint32("code", r.Code))
		code = http.StatusInternalServerError
	}
	s.writeError(w, newErrorResponse(r.Data), code)
}

func (s *API) writeError(w http.ResponseWriter, e error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	buf, err := json.Marshal(e)
	if err != nil {
		s.log.Error("couldn't marshal error", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = w.Write(buf)
	if err != nil {
		s.log.Error("couldn't write error to HTTP response", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.log.Info(fmt.Sprintf("%d %s", status, http.StatusText(status)))
}

func (s *API) writeSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if data == nil {
		s.log.Info(fmt.Sprintf("%d %s", http.StatusOK, http.StatusText(http.StatusOK)))
		return
	}

	buf, err := json.Marshal(data)
	if err != nil {
		s.log.Error("couldn't marshal error", zap.Error(err))
		s.writeInternalError(w, fmt.Errorf("couldn't marshal error: %w", err))
		return
	}

	_, err = w.Write(buf)
	if err != nil {
		s.log.Error("couldn't write error to HTTP response", zap.Error(err))
		s.writeInternalError(w, fmt.Errorf("couldn't write error to HTTP response: %w", err))
		return
	}
	s.log.Info(fmt.Sprintf("%d %s", http.StatusOK, http.StatusText(http.StatusOK)))
}

// convertSpamStatistics takes us from the protos to the nodetypes version
// the code is copied from the V2 API but is not worth the pain of trying to share
// because V1 is disappearing soon anyway.
func convertSpamStatistics(r *api.GetSpamStatisticsResponse) *nodetypes.SpamStatistics {
	proposals := map[string]uint64{}
	for _, st := range r.Statistics.Votes.Statistics {
		proposals[st.Proposal] = st.CountForEpoch
	}

	blockStates := []nodetypes.PoWBlockState{}
	for _, b := range r.Statistics.Pow.BlockStates {
		blockStates = append(blockStates, nodetypes.PoWBlockState{
			BlockHeight:          b.BlockHeight,
			BlockHash:            b.BlockHash,
			TransactionsSeen:     b.TransactionsSeen,
			ExpectedDifficulty:   b.ExpectedDifficulty,
			HashFunction:         b.HashFunction,
			TxPerBlock:           b.TxPerBlock,
			IncreasingDifficulty: b.IncreasingDifficulty,
			Difficulty:           b.Difficulty,
		})
	}

	// sort by block-height so latest block is first
	sort.Slice(blockStates, func(i int, j int) bool {
		return blockStates[i].BlockHeight > blockStates[j].BlockHeight
	})

	var lastBlockHeight uint64
	if len(blockStates) > 0 {
		lastBlockHeight = blockStates[0].BlockHeight
	}
	return &nodetypes.SpamStatistics{
		Proposals:         toSpamStatistic(r.Statistics.Proposals),
		Delegations:       toSpamStatistic(r.Statistics.Delegations),
		Transfers:         toSpamStatistic(r.Statistics.Transfers),
		NodeAnnouncements: toSpamStatistic(r.Statistics.NodeAnnouncements),
		Votes: &nodetypes.VoteSpamStatistics{
			Proposals:   proposals,
			MaxForEpoch: r.Statistics.Votes.MaxForEpoch,
			BannedUntil: r.Statistics.Votes.BannedUntil,
		},
		PoW: &nodetypes.PoWStatistics{
			PowBlockStates: blockStates,
			BannedUntil:    r.Statistics.Pow.BannedUntil,
			PastBlocks:     r.Statistics.Pow.NumberOfPastBlocks,
		},
		ChainID:         r.ChainId,
		EpochSeq:        r.Statistics.EpochSeq,
		LastBlockHeight: lastBlockHeight,
	}
}

func toSpamStatistic(st *api.SpamStatistic) *nodetypes.SpamStatistic {
	return &nodetypes.SpamStatistic{
		CountForEpoch: st.CountForEpoch,
		MaxForEpoch:   st.MaxForEpoch,
		BannedUntil:   st.BannedUntil,
	}
}
