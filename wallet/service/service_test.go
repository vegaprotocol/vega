package service_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"code.vegaprotocol.io/vega/wallet/api/mocks"
	"code.vegaprotocol.io/vega/wallet/network"
	"code.vegaprotocol.io/vega/wallet/service"
	v1 "code.vegaprotocol.io/vega/wallet/service/v1"
	v1mocks "code.vegaprotocol.io/vega/wallet/service/v1/mocks"
	v2 "code.vegaprotocol.io/vega/wallet/service/v2"
	"code.vegaprotocol.io/vega/wallet/service/v2/connections"
	v2connectionsmocks "code.vegaprotocol.io/vega/wallet/service/v2/connections/mocks"
	v2mocks "code.vegaprotocol.io/vega/wallet/service/v2/mocks"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/golang/mock/gomock"
	"go.uber.org/zap"
)

type testService struct {
	*service.Service

	ctrl              *gomock.Controller
	handler           *v1mocks.MockWalletHandler
	nodeForward       *v1mocks.MockNodeForward
	consentRequestsCh chan v1.ConsentRequest
	auth              *v1mocks.MockAuth

	clientAPI *v2mocks.MockClientAPI

	pow         *mocks.MockProofOfWork
	timeService *v2connectionsmocks.MockTimeService
	walletStore *v2connectionsmocks.MockWalletStore
	tokenStore  *v2connectionsmocks.MockTokenStore
}

func (s *testService) serveHTTP(t *testing.T, req *http.Request) (int, http.Header, []byte) {
	t.Helper()
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	resp := w.Result() //nolint:bodyclose
	defer func() {
		if err := w.Result().Body.Close(); err != nil {
			t.Logf("couldn't close response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("couldn't read body: %v", err)
	}

	return resp.StatusCode, resp.Header, body
}

func getTestServiceV1(t *testing.T, consentPolicy string) *testService {
	t.Helper()

	net := &network.Network{}

	ctrl := gomock.NewController(t)

	handler := v1mocks.NewMockWalletHandler(ctrl)
	auth := v1mocks.NewMockAuth(ctrl)
	nodeForward := v1mocks.NewMockNodeForward(ctrl)
	pow := mocks.NewMockProofOfWork(ctrl)

	consentRequestsCh := make(chan v1.ConsentRequest, 1)
	sentTxs := make(chan v1.SentTransaction, 1)

	ctx, cancelFn := context.WithCancel(context.Background())

	var policy v1.Policy
	switch consentPolicy {
	case "automatic":
		policy = v1.NewAutomaticConsentPolicy()
	case "manual":
		policy = v1.NewExplicitConsentPolicy(ctx, consentRequestsCh, sentTxs)
	default:
		t.Fatalf("unknown consent policy: %s", consentPolicy)
	}

	apiV1 := v1.NewAPI(zap.NewNop(), handler, auth, nodeForward, policy, net, pow)

	s := service.NewService(zap.NewNop(), net, apiV1, nil)

	t.Cleanup(func() {
		if err := s.Stop(context.Background()); err != nil {
			t.Log("The service couldn't stop properly")
		}
		cancelFn()
		close(consentRequestsCh)
		close(sentTxs)
	})

	return &testService{
		Service: s,
		ctrl:    ctrl,

		// V1
		handler:           handler,
		auth:              auth,
		nodeForward:       nodeForward,
		consentRequestsCh: consentRequestsCh,
		pow:               pow,
	}
}

type longLivingTokenSetupForTest struct {
	tokenDescription connections.TokenDescription
	wallet           wallet.Wallet
}

func getTestServiceV2(t *testing.T, tokenSetups ...longLivingTokenSetupForTest) *testService {
	t.Helper()

	net := &network.Network{}

	ctrl := gomock.NewController(t)

	clientAPI := v2mocks.NewMockClientAPI(ctrl)
	pow := mocks.NewMockProofOfWork(ctrl)
	timeService := v2connectionsmocks.NewMockTimeService(ctrl)
	walletStore := v2connectionsmocks.NewMockWalletStore(ctrl)
	tokenStore := v2connectionsmocks.NewMockTokenStore(ctrl)

	if len(tokenSetups) > 0 {
		tokenSummaries := make([]connections.TokenSummary, 0, len(tokenSetups))
		for _, tokenSetup := range tokenSetups {
			tokenSummaries = append(tokenSummaries, connections.TokenSummary{
				Description:    tokenSetup.tokenDescription.Description,
				Token:          tokenSetup.tokenDescription.Token,
				CreationDate:   tokenSetup.tokenDescription.CreationDate,
				ExpirationDate: tokenSetup.tokenDescription.ExpirationDate,
			})
			tokenStore.EXPECT().DescribeToken(tokenSetup.tokenDescription.Token).AnyTimes().Return(tokenSetup.tokenDescription, nil)
			walletStore.EXPECT().UnlockWallet(gomock.Any(), tokenSetup.tokenDescription.Wallet.Name, tokenSetup.tokenDescription.Wallet.Passphrase).Times(1).Return(nil)
			walletStore.EXPECT().GetWallet(gomock.Any(), tokenSetup.tokenDescription.Wallet.Name).Times(1).Return(tokenSetup.wallet, nil)
		}
		tokenStore.EXPECT().ListTokens().AnyTimes().Return(tokenSummaries, nil)
	} else {
		tokenStore.EXPECT().ListTokens().Times(1).Return(nil, nil)
	}

	walletStore.EXPECT().OnUpdate(gomock.Any()).Times(1)

	connectionsManager, err := connections.NewManager(timeService, walletStore, tokenStore)
	if err != nil {
		t.Fatalf("could not instantiate the connection manager for tests: %v", err)
	}

	apiV2 := v2.NewAPI(zap.NewNop(), clientAPI, connectionsManager)

	s := service.NewService(zap.NewNop(), net, nil, apiV2)

	t.Cleanup(func() {
		if err := s.Stop(context.Background()); err != nil {
			t.Log("The service couldn't stop properly")
		}
		connectionsManager.EndAllSessionConnections()
	})

	return &testService{
		Service: s,
		ctrl:    ctrl,

		clientAPI:   clientAPI,
		timeService: timeService,
		walletStore: walletStore,
		tokenStore:  tokenStore,

		pow: pow,
	}
}

func buildRequest(t *testing.T, method, path, payload string, headers map[string]string) *http.Request {
	t.Helper()

	ctx, cancelFn := context.WithTimeout(context.Background(), testRequestTimeout)
	t.Cleanup(func() {
		cancelFn()
	})

	req, _ := http.NewRequestWithContext(ctx, method, path, strings.NewReader(payload))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req
}
