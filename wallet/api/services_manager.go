package api

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/wallet/api/session"
)

var ErrNoServiceIsRunningForThisNetwork = errors.New("no service is running for this network")

type RunningService struct {
	url            string
	shutdownSwitch *ServiceShutdownSwitch
	sessions       *session.Sessions
}

// ServicesManager keeps track of all running services. It is used by the
// admin API to control the lifecycle of the services and query their state.
type ServicesManager struct {
	servicesByNetwork map[string]*RunningService

	mu          sync.Mutex
	tokenStore  TokenStore
	walletStore WalletStore
}

func (ns *ServicesManager) RegisterService(network, url string, sessions *session.Sessions, shutdownSwitch *ServiceShutdownSwitch) error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if svc, ok := ns.servicesByNetwork[network]; ok {
		return fmt.Errorf("a service is already running for this network at %q", svc.url)
	}

	tokens, err := ns.tokenStore.ListTokens()
	if err != nil {
		return fmt.Errorf("could not retrieve the list of registered tokens: %w", err)
	}

	for _, token := range tokens {
		tokenInfo, err := ns.tokenStore.GetToken(token.Token)
		if err != nil {
			return fmt.Errorf("could not retrieve the information linked to the token %q: %w", token.Token, err)
		}
		w, err := ns.walletStore.GetWallet(context.Background(), tokenInfo.Wallet.Name, tokenInfo.Wallet.Passphrase)
		if err != nil {
			return fmt.Errorf("could not retrieve the wallet %q associated to the token %q: %w", tokenInfo.Wallet.Name, token.Token, err)
		}

		var expiry *time.Time
		if tokenInfo.Expiry != nil {
			expiry = ptr.From(time.Unix(*tokenInfo.Expiry, 0))
		}

		if err := sessions.ConnectWalletForLongLivingConnection(token.Token, w, time.Now(), expiry); err != nil {
			return fmt.Errorf("could not connect the wallet %q: %w", tokenInfo.Wallet.Name, err)
		}
	}

	ns.servicesByNetwork[network] = &RunningService{
		url:            url,
		shutdownSwitch: shutdownSwitch,
		sessions:       sessions,
	}

	return nil
}

func (ns *ServicesManager) Sessions(network string) (*session.Sessions, error) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	service, ok := ns.servicesByNetwork[network]
	if !ok {
		return nil, ErrNoServiceIsRunningForThisNetwork
	}

	return service.sessions, nil
}

func (ns *ServicesManager) AbortService(network string, err error) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	service, ok := ns.servicesByNetwork[network]
	if !ok {
		return
	}

	service.shutdownSwitch.Flip(err)
	service.sessions.DisconnectAllWallets()

	delete(ns.servicesByNetwork, network)
}

func (ns *ServicesManager) StopService(network string) {
	ns.AbortService(network, nil)
}

func NewServicesManager(tokenStore TokenStore, walletStore WalletStore) *ServicesManager {
	return &ServicesManager{
		servicesByNetwork: map[string]*RunningService{},
		tokenStore:        tokenStore,
		walletStore:       walletStore,
	}
}

type ServiceShutdownSwitch struct {
	ctx          context.Context
	cFunc        context.CancelFunc
	wg           *sync.WaitGroup
	onErrorFunc  func(error)
	beforeCancel func()
}

func (s *ServiceShutdownSwitch) Ctx() context.Context {
	return s.ctx
}

func (s *ServiceShutdownSwitch) Flip(err error) {
	if err != nil && s.onErrorFunc != nil {
		s.onErrorFunc(err)
	}
	if s.beforeCancel != nil {
		s.beforeCancel()
	}
	s.cFunc()
}

func (s *ServiceShutdownSwitch) Flipped() <-chan struct{} {
	return s.ctx.Done()
}

func (s *ServiceShutdownSwitch) BindToProcess() ProcessStoppedNotifier {
	s.wg.Add(1)
	return func() {
		s.wg.Done()
	}
}

func (s *ServiceShutdownSwitch) WaitForShutdown() {
	s.wg.Wait()
}

func (s *ServiceShutdownSwitch) BeforeCancelFunc(f func()) {
	s.beforeCancel = f
}

func NewServiceShutdownSwitch(onErrorFunc func(error)) *ServiceShutdownSwitch {
	ctx, cFunc := context.WithCancel(context.Background())
	return &ServiceShutdownSwitch{
		ctx:          ctx,
		cFunc:        cFunc,
		wg:           &sync.WaitGroup{},
		beforeCancel: nil,
		onErrorFunc:  onErrorFunc,
	}
}
