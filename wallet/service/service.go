package service

import (
	"context"
	"fmt"
	"net/http"

	v1 "code.vegaprotocol.io/vega/wallet/service/v1"
	v2 "code.vegaprotocol.io/vega/wallet/service/v2"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

type Service struct {
	*httprouter.Router

	log *zap.Logger

	server *http.Server

	apiV1 *v1.API
	apiV2 *v2.API
}

func NewService(log *zap.Logger, cfg *Config, apiV1 *v1.API, apiV2 *v2.API) *Service {
	s := &Service{
		Router: httprouter.New(),
		log:    log,
		apiV1:  apiV1,
		apiV2:  apiV2,
	}

	s.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%v", cfg.Server.Host, cfg.Server.Port),
		Handler: cors.AllowAll().Handler(s),
	}

	// V1
	s.handleV1(http.MethodPost, "/api/v1/auth/token", s.apiV1.Login)
	s.handleV1(http.MethodDelete, "/api/v1/auth/token", s.apiV1.Revoke)

	s.handleV1(http.MethodGet, "/api/v1/network", s.apiV1.GetNetwork)
	s.handleV1(http.MethodGet, "/api/v1/network/chainid", s.apiV1.GetNetworkChainID)

	s.handleV1(http.MethodPost, "/api/v1/wallets", s.apiV1.CreateWallet)
	s.handleV1(http.MethodPost, "/api/v1/wallets/import", s.apiV1.ImportWallet)

	s.handleV1(http.MethodGet, "/api/v1/keys", s.apiV1.ListPublicKeys)
	s.handleV1(http.MethodPost, "/api/v1/keys", s.apiV1.GenerateKeyPair)
	s.handleV1(http.MethodGet, "/api/v1/keys/:keyid", s.apiV1.GetPublicKey)
	s.handleV1(http.MethodPut, "/api/v1/keys/:keyid/taint", s.apiV1.TaintKey)
	s.handleV1(http.MethodPut, "/api/v1/keys/:keyid/metadata", s.apiV1.UpdateMeta)

	s.handleV1(http.MethodPost, "/api/v1/command", s.apiV1.SignTx)
	s.handleV1(http.MethodPost, "/api/v1/command/sync", s.apiV1.SignTxSync)
	s.handleV1(http.MethodPost, "/api/v1/command/check", s.apiV1.CheckTx)
	s.handleV1(http.MethodPost, "/api/v1/command/commit", s.apiV1.SignTxCommit)
	s.handleV1(http.MethodPost, "/api/v1/sign", s.apiV1.SignAny)
	s.handleV1(http.MethodPost, "/api/v1/verify", s.apiV1.VerifyAny)

	s.handleV1(http.MethodGet, "/api/v1/version", s.apiV1.Version)
	s.handleV1(http.MethodGet, "/api/v1/status", s.apiV1.Health)

	// V2
	s.Handle(http.MethodGet, "/api/v2/health", s.apiV2.CheckHealth)
	s.Handle(http.MethodGet, "/api/v2/methods", s.apiV2.ListMethods)
	s.Handle(http.MethodPost, "/api/v2/requests", s.apiV2.HandleRequest)

	return s
}

func (s *Service) Start() error {
	return s.server.ListenAndServe()
}

func (s *Service) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Service) handleV1(method string, path string, handle httprouter.Handle) {
	loggedEndpoint := func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		s.log.Info(fmt.Sprintf("Entering %s %s", method, path))
		handle(w, r, p)
		s.log.Info(fmt.Sprintf("Leaving %s %s", method, path))
	}
	s.Handle(method, path, loggedEndpoint)
}
