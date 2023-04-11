package connections_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	apimocks "code.vegaprotocol.io/vega/wallet/api/mocks"
	"code.vegaprotocol.io/vega/wallet/service/v2/connections"
	"code.vegaprotocol.io/vega/wallet/service/v2/connections/mocks"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager(t *testing.T) {
	t.Run("Start a new session from scratch succeeds", testStartNewSessionFromScratchSucceeds)
	t.Run("Loading long-living connections succeeds", testLoadingLongLivingConnectionsSucceeding)
	t.Run("Asynchronous updates on wallets update the connections", testAsynchronousUpdateOnWalletsUpdateTheConnection)
	t.Run("Asynchronous updates on long-living tokens update the connections", testAsynchronousUpdateOnLongLivingTokensUpdateTheConnection)
	t.Run("Reloading previous sessions succeeds", testReloadingPreviousSessionsSucceeds)
}

func testStartNewSessionFromScratchSucceeds(t *testing.T) {
	ctx, _ := randomTraceID(t)

	// given
	// The prefix with "b" will help to test the sorting while listing connections.
	hostnameB := "b" + vgrand.RandomStr(5)
	expectedWallet, expectedKeyPairs := randomWallet(t)
	// Tainting one key to prove it's not loaded as an allowed key.
	require.NoError(t, expectedWallet.TaintKey(expectedKeyPairs[1].PublicKey()))

	// setup
	manager := newTestManagerBuilder(t)
	manager.timeService.EXPECT().Now().AnyTimes().Return(time.Now())
	manager.walletStore.EXPECT().OnUpdate(gomock.Any()).Times(1)
	manager.tokenStore.EXPECT().OnUpdate(gomock.Any()).Times(1)
	manager.tokenStore.EXPECT().ListTokens().Times(1).Return(nil, nil)
	manager.sessionStore.EXPECT().ListSessions(gomock.Any()).Times(1).Return(nil, nil)
	manager.sessionStore.EXPECT().TrackSession(gomock.Any()).Times(1).Return(nil)
	manager.Build()

	// when initiating the session connection for the first time, we get a token back.
	firstSession, err := manager.StartSession(hostnameB, expectedWallet)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, firstSession)

	// when
	firstCw, err := manager.ConnectedWallet(ctx, hostnameB, firstSession)

	// then
	require.Nil(t, err)
	assert.Equal(t, expectedWallet.Name(), firstCw.Name())
	assert.Equal(t, hostnameB, firstCw.Hostname())
	// The used hostname is not permitted to list keys on the wallet.
	assert.Equal(t, expectedWallet.Permissions(hostnameB).CanListKeys(), firstCw.CanListKeys())
	// Since the hostname is not allowed to list keys, there is no allowed keys.
	assert.Empty(t, firstCw.AllowedKeys())
	// Regular sessions require user intervention, and this, interaction.
	assert.True(t, firstCw.RequireInteraction())

	// when using the token from different hostname, it fails.
	_, err = manager.ConnectedWallet(ctx, vgrand.RandomStr(5), firstSession)

	// then
	require.NotNil(t, err)
	assert.Equal(t, jsonrpc.NewServerError(api.ErrorCodeAuthenticationFailure, connections.ErrHostnamesMismatchForThisToken), err)

	// setup
	manager.sessionStore.EXPECT().TrackSession(gomock.Any()).Times(1).Return(nil)

	// when re-initiating a session connection on same hostname-wallet pair, the
	// original session is voided, and a new token is generated.
	secondSession, err := manager.StartSession(hostnameB, expectedWallet)

	// then the tokens are different
	require.NoError(t, err)
	assert.NotEmpty(t, secondSession)
	assert.NotEqual(t, firstSession, secondSession)

	// given
	// The prefix with "a" will help to test the sorting while listing the connections.
	hostnameA := "a" + vgrand.RandomStr(5)

	// setup
	manager.sessionStore.EXPECT().TrackSession(gomock.Any()).Times(1).Return(nil)
	// Allowing a hostname to list keys.
	require.NoError(t, expectedWallet.UpdatePermissions(hostnameA, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:      wallet.ReadAccess,
			AllowedKeys: nil,
		},
	}))

	// then
	thirdSession, err := manager.StartSession(hostnameA, expectedWallet)

	// then a brand-new token is issued
	require.NoError(t, err)
	assert.NotEmpty(t, thirdSession)
	assert.NotEqual(t, firstSession, thirdSession)
	assert.NotEqual(t, secondSession, thirdSession)

	// when
	secondCw, err := manager.ConnectedWallet(ctx, hostnameA, thirdSession)

	// then
	require.Nil(t, err)
	assert.Equal(t, expectedWallet.Name(), secondCw.Name())
	assert.Equal(t, hostnameA, secondCw.Hostname())
	// The used hostname is permitted to list keys on the wallet.
	assert.Equal(t, expectedWallet.Permissions(hostnameA).CanListKeys(), secondCw.CanListKeys())
	// Since the hostname is allowed to list all keys, it will allow all non-tainted keys.
	assertRightAllowedKeys(t, []wallet.KeyPair{expectedKeyPairs[0], expectedKeyPairs[2]}, secondCw.AllowedKeys())
	// Regular sessions require user intervention, and thus, interaction.
	assert.True(t, secondCw.RequireInteraction())

	// given
	// The prefix with "a" will help to test the sorting while listing the connections.
	hostnameC := "c" + vgrand.RandomStr(5)

	// setup
	manager.sessionStore.EXPECT().TrackSession(gomock.Any()).Times(1).Return(nil)
	// Allowing a hostname to list keys.
	require.NoError(t, expectedWallet.UpdatePermissions(hostnameC, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:      wallet.ReadAccess,
			AllowedKeys: []string{expectedKeyPairs[2].PublicKey()},
		},
	}))

	// when
	fourthSession, err := manager.StartSession(hostnameC, expectedWallet)

	// then a brand-new token is issued
	require.NoError(t, err)
	assert.NotEmpty(t, fourthSession)
	assert.NotEqual(t, firstSession, fourthSession)
	assert.NotEqual(t, secondSession, fourthSession)
	assert.NotEqual(t, thirdSession, fourthSession)

	// when
	thirdCw, err := manager.ConnectedWallet(ctx, hostnameC, fourthSession)

	// then
	require.Nil(t, err)
	assert.Equal(t, expectedWallet.Name(), thirdCw.Name())
	assert.Equal(t, hostnameC, thirdCw.Hostname())
	// The used hostname is permitted to list keys on the wallet.
	assert.Equal(t, expectedWallet.Permissions(hostnameC).CanListKeys(), thirdCw.CanListKeys())
	// Since the hostname is allowed to list all keys, it will allow all non-tainted keys.
	assertRightAllowedKeys(t, []wallet.KeyPair{expectedKeyPairs[2]}, thirdCw.AllowedKeys())
	// Regular sessions require user intervention, and thus, interaction.
	assert.True(t, thirdCw.RequireInteraction())

	// when
	liveConnections := manager.ListSessionConnections()

	// then
	assert.Equal(t, []api.Connection{
		{
			Hostname: hostnameA,
			Wallet:   expectedWallet.Name(),
		}, {
			Hostname: hostnameB,
			Wallet:   expectedWallet.Name(),
		}, {
			Hostname: hostnameC,
			Wallet:   expectedWallet.Name(),
		},
	}, liveConnections)

	// setup
	manager.EndSessionConnection(hostnameA, expectedWallet.Name())

	// when listing the connections after one as been ended, the ended one is not
	// listed.
	liveConnections = manager.ListSessionConnections()

	// then
	assert.Equal(t, []api.Connection{
		{
			Hostname: hostnameB,
			Wallet:   expectedWallet.Name(),
		}, {
			Hostname: hostnameC,
			Wallet:   expectedWallet.Name(),
		},
	}, liveConnections)

	// when trying to get the connection previously ended, it fails.
	secondCw, err = manager.ConnectedWallet(ctx, hostnameC, thirdSession)

	// then
	require.NotNil(t, err)
	assert.Equal(t, jsonrpc.NewServerError(api.ErrorCodeAuthenticationFailure, connections.ErrNoConnectionAssociatedThisAuthenticationToken), err)

	// setup
	manager.EndAllSessionConnections()

	// when listing the connections after they've been all closed, none of them
	// are listed.
	liveConnections = manager.ListSessionConnections()

	// then
	assert.Empty(t, liveConnections)
}

func testLoadingLongLivingConnectionsSucceeding(t *testing.T) {
	ctx := context.Background()

	// given
	someHostname := vgrand.RandomStr(4)
	expectedWallet, expectedKeyPairs := randomWallet(t)
	// Tainting one key to prove it's not loaded as an allowed key.
	require.NoError(t, expectedWallet.TaintKey(expectedKeyPairs[1].PublicKey()))
	expectedToken := connections.TokenDescription{
		CreationDate: time.Now(),
		Token:        randomToken(t),
		Wallet: connections.WalletCredentials{
			Name:       expectedWallet.Name(),
			Passphrase: vgrand.RandomStr(5),
		},
	}
	expectedTokens := []connections.TokenSummary{
		{
			Token:        expectedToken.Token,
			CreationDate: expectedToken.CreationDate,
		},
	}

	// setup
	manager := newTestManagerBuilder(t)
	manager.timeService.EXPECT().Now().AnyTimes().Return(time.Now())
	manager.walletStore.EXPECT().OnUpdate(gomock.Any()).Times(1)
	manager.tokenStore.EXPECT().OnUpdate(gomock.Any()).Times(1)
	manager.tokenStore.EXPECT().ListTokens().Times(1).Return(expectedTokens, nil)
	manager.tokenStore.EXPECT().DescribeToken(expectedToken.Token).Times(1).Return(expectedToken, nil)
	manager.walletStore.EXPECT().UnlockWallet(gomock.Any(), expectedToken.Wallet.Name, expectedToken.Wallet.Passphrase).Times(1).Return(nil)
	manager.walletStore.EXPECT().GetWallet(gomock.Any(), expectedToken.Wallet.Name).Times(1).Return(expectedWallet, nil)
	manager.sessionStore.EXPECT().ListSessions(gomock.Any()).Times(1).Return(nil, nil)
	manager.Build()

	// when retrieving the connection associated to the long-living token.
	// Note: Long-living token are not tied to a hostname.
	cw, err := manager.ConnectedWallet(ctx, someHostname, expectedToken.Token)

	// then
	require.Nil(t, err)
	assert.Equal(t, expectedWallet.Name(), cw.Name())
	// There is no restriction on who can use a long-living token.
	assert.Empty(t, cw.Hostname())
	// This is always allowed on long-living tokens.
	assert.True(t, cw.CanListKeys())
	// Tainted keys are excluded from the allowed keys.
	assertRightAllowedKeys(t, []wallet.KeyPair{expectedKeyPairs[0], expectedKeyPairs[2]}, cw.AllowedKeys())
	// Long-living token should not require user interaction.
	assert.False(t, cw.RequireInteraction())

	// when trying to end the connection, it doesn't close it.
	manager.EndSessionConnectionWithToken(expectedToken.Token)
	unclosedCW, err := manager.ConnectedWallet(ctx, someHostname, expectedToken.Token)

	// then
	require.Nil(t, err)
	// We can't close a long-living token connection.
	assert.Equal(t, cw, unclosedCW)

	// when ending all session connections, it doesn't close the long-living connection.
	manager.EndAllSessionConnections()
	unclosedCW, err = manager.ConnectedWallet(ctx, someHostname, expectedToken.Token)

	// then
	require.Nil(t, err)
	// We can't close a long-living token connection.
	assert.Equal(t, cw, unclosedCW)
}

func testAsynchronousUpdateOnWalletsUpdateTheConnection(t *testing.T) {
	ctx, _ := randomTraceID(t)

	// given
	var onWalletUpdateCb func(context.Context, wallet.Event)
	hostname := vgrand.RandomStr(5)
	expectedWallet, expectedKeyPairs := randomWallet(t)

	// setup
	manager := newTestManagerBuilder(t)
	manager.timeService.EXPECT().Now().AnyTimes().Return(time.Now())
	manager.walletStore.EXPECT().OnUpdate(gomock.Any()).Times(1).Do(func(cb func(context.Context, wallet.Event)) {
		// Capturing the callback, so we can use it like the wallet store would.
		onWalletUpdateCb = cb
	})
	manager.tokenStore.EXPECT().OnUpdate(gomock.Any()).Times(1)
	manager.tokenStore.EXPECT().ListTokens().Times(1).Return(nil, nil)
	manager.sessionStore.EXPECT().ListSessions(gomock.Any()).Times(1).Return(nil, nil)
	manager.sessionStore.EXPECT().TrackSession(gomock.Any()).Times(1).Return(nil)
	manager.Build()

	// when initiating the session connection for the first time, we get a token back.
	session, err := manager.StartSession(hostname, expectedWallet)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, session)

	// when
	cw, err := manager.ConnectedWallet(ctx, hostname, session)

	// then
	require.Nil(t, err)
	assert.Equal(t, expectedWallet.Name(), cw.Name())
	assert.Equal(t, hostname, cw.Hostname())
	assert.False(t, cw.CanListKeys())
	assert.Empty(t, cw.AllowedKeys())
	assert.True(t, cw.RequireInteraction())

	// setup
	// Add some permissions to the wallet to see if the connection is updated
	// accordingly.
	require.NoError(t, expectedWallet.UpdatePermissions(hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:      wallet.ReadAccess,
			AllowedKeys: []string{expectedKeyPairs[2].PublicKey()},
		},
	}))

	// when
	// Simulating an external wallet update.
	onWalletUpdateCb(ctx, wallet.NewUnlockedWalletUpdatedEvent(expectedWallet))
	cw, err = manager.ConnectedWallet(ctx, hostname, session)

	// then
	require.Nil(t, err)
	assert.Equal(t, expectedWallet.Name(), cw.Name())
	assert.Equal(t, hostname, cw.Hostname())
	assert.True(t, cw.CanListKeys())
	assertRightAllowedKeys(t, []wallet.KeyPair{expectedKeyPairs[2]}, cw.AllowedKeys())
	assert.True(t, cw.RequireInteraction())

	// when
	// Simulating an external wallet update.
	onWalletUpdateCb(ctx, wallet.NewUnlockedWalletUpdatedEvent(expectedWallet))
	cw, err = manager.ConnectedWallet(ctx, hostname, session)

	// then
	require.Nil(t, err)
	assert.Equal(t, expectedWallet.Name(), cw.Name())
	assert.Equal(t, hostname, cw.Hostname())
	assert.True(t, cw.CanListKeys())
	assertRightAllowedKeys(t, []wallet.KeyPair{expectedKeyPairs[2]}, cw.AllowedKeys())
	assert.True(t, cw.RequireInteraction())

	// setup
	previousName := expectedWallet.Name()
	expectedWallet.SetName(vgrand.RandomStr(5))
	manager.walletStore.EXPECT().GetWallet(gomock.Any(), expectedWallet.Name()).Times(1).Return(expectedWallet, nil)

	// when
	// Simulating an external wallet rename.
	onWalletUpdateCb(ctx, wallet.NewWalletRenamedEvent(previousName, expectedWallet.Name()))
	cw, err = manager.ConnectedWallet(ctx, hostname, session)

	// then
	require.Nil(t, err)
	assert.Equal(t, expectedWallet.Name(), cw.Name())
	assert.Equal(t, hostname, cw.Hostname())
	assert.True(t, cw.CanListKeys())
	assertRightAllowedKeys(t, []wallet.KeyPair{expectedKeyPairs[2]}, cw.AllowedKeys())
	assert.True(t, cw.RequireInteraction())

	// setup
	manager.sessionStore.EXPECT().DeleteSession(gomock.Any(), session).Times(1).Return(nil)

	// when
	// Simulating an external wallet removal.
	onWalletUpdateCb(ctx, wallet.NewWalletRemovedEvent(expectedWallet.Name()))
	cw, err = manager.ConnectedWallet(ctx, hostname, session)

	// then
	require.NotNil(t, err)
	assert.Equal(t, jsonrpc.NewServerError(api.ErrorCodeAuthenticationFailure, connections.ErrNoConnectionAssociatedThisAuthenticationToken), err)
	require.Empty(t, cw)
}

func testAsynchronousUpdateOnLongLivingTokensUpdateTheConnection(t *testing.T) {
	ctx, _ := randomTraceID(t)

	// given
	var onTokenUpdateCb func(context.Context, ...connections.TokenDescription)
	passphrase := vgrand.RandomStr(5)
	expectedWallet, expectedKeyPairs := randomWallet(t)
	firstToken := connections.TokenDescription{
		CreationDate: time.Now(),
		Token:        randomToken(t),
		Wallet: connections.WalletCredentials{
			Name:       expectedWallet.Name(),
			Passphrase: passphrase,
		},
	}
	expectedTokens := []connections.TokenSummary{
		{
			Token:        firstToken.Token,
			CreationDate: firstToken.CreationDate,
		},
	}

	// setup
	manager := newTestManagerBuilder(t)
	manager.timeService.EXPECT().Now().AnyTimes().Return(time.Now())
	manager.walletStore.EXPECT().OnUpdate(gomock.Any()).Times(1)
	manager.tokenStore.EXPECT().OnUpdate(gomock.Any()).Times(1).Do(func(cb func(context.Context, ...connections.TokenDescription)) {
		// Capturing the callback, so we can use it like the token store would.
		onTokenUpdateCb = cb
	})
	manager.tokenStore.EXPECT().ListTokens().Times(1).Return(expectedTokens, nil)
	manager.tokenStore.EXPECT().DescribeToken(firstToken.Token).Times(1).Return(firstToken, nil)
	manager.walletStore.EXPECT().UnlockWallet(gomock.Any(), firstToken.Wallet.Name, firstToken.Wallet.Passphrase).Times(1).Return(nil)
	manager.walletStore.EXPECT().GetWallet(gomock.Any(), firstToken.Wallet.Name).Times(1).Return(expectedWallet, nil)
	manager.sessionStore.EXPECT().ListSessions(gomock.Any()).Times(1).Return(nil, nil)
	manager.Build()

	// when retrieving the connection associated to the long-living token.
	// Note: Long-living token are not tied to a hostname.
	cw, err := manager.ConnectedWallet(ctx, "", firstToken.Token)

	// then
	require.Nil(t, err)
	assert.Equal(t, expectedWallet.Name(), cw.Name())
	assert.Empty(t, cw.Hostname())
	assert.True(t, cw.CanListKeys())
	assertRightAllowedKeys(t, []wallet.KeyPair{expectedKeyPairs[0], expectedKeyPairs[1], expectedKeyPairs[2]}, cw.AllowedKeys())
	assert.False(t, cw.RequireInteraction())

	// given
	secondToken := connections.TokenDescription{
		CreationDate: time.Now(),
		Token:        randomToken(t),
		Wallet: connections.WalletCredentials{
			Name:       expectedWallet.Name(),
			Passphrase: passphrase,
		},
	}

	// setup
	manager.walletStore.EXPECT().UnlockWallet(ctx, firstToken.Wallet.Name, firstToken.Wallet.Passphrase).Times(1).Return(nil)
	manager.walletStore.EXPECT().GetWallet(ctx, firstToken.Wallet.Name).Times(1).Return(expectedWallet, nil)
	manager.walletStore.EXPECT().UnlockWallet(ctx, secondToken.Wallet.Name, secondToken.Wallet.Passphrase).Times(1).Return(nil)
	manager.walletStore.EXPECT().GetWallet(ctx, secondToken.Wallet.Name).Times(1).Return(expectedWallet, nil)

	// when simulating the creation of a second token.
	onTokenUpdateCb(ctx, firstToken, secondToken)

	// when
	cw, err = manager.ConnectedWallet(ctx, "", secondToken.Token)

	// then
	require.Nil(t, err)
	assert.Equal(t, expectedWallet.Name(), cw.Name())
	assert.Empty(t, cw.Hostname())
	assert.True(t, cw.CanListKeys())
	assertRightAllowedKeys(t, []wallet.KeyPair{expectedKeyPairs[0], expectedKeyPairs[1], expectedKeyPairs[2]}, cw.AllowedKeys())
	assert.False(t, cw.RequireInteraction())

	// setup
	manager.walletStore.EXPECT().UnlockWallet(ctx, secondToken.Wallet.Name, secondToken.Wallet.Passphrase).Times(1).Return(nil)
	manager.walletStore.EXPECT().GetWallet(ctx, secondToken.Wallet.Name).Times(1).Return(expectedWallet, nil)

	// when simulating the deletion of the first token
	onTokenUpdateCb(ctx, secondToken)

	// when
	cw, err = manager.ConnectedWallet(ctx, "", firstToken.Token)

	// then
	require.NotNil(t, err)
	assert.Equal(t, jsonrpc.NewServerError(api.ErrorCodeAuthenticationFailure, connections.ErrNoConnectionAssociatedThisAuthenticationToken), err)
	assert.Empty(t, cw)
}

func testReloadingPreviousSessionsSucceeds(t *testing.T) {
	ctx, traceID := randomTraceID(t)

	// given
	hostnameA := "a" + vgrand.RandomStr(5)
	hostnameB := "b" + vgrand.RandomStr(5)
	walletA, _ := randomWalletWithName(t, "a"+vgrand.RandomStr(5))
	walletB, _ := randomWalletWithName(t, "b"+vgrand.RandomStr(5))
	walletAPassphrase := vgrand.RandomStr(5)
	nonExistingWallet := vgrand.RandomStr(5)
	tokenOnNonExistingWallet := randomToken(t)
	token1 := randomToken(t)
	token2 := randomToken(t)
	token3 := randomToken(t)
	previousSessions := []connections.Session{
		{
			Token:    token1,
			Hostname: hostnameA,
			Wallet:   walletA.Name(),
		}, {
			Token:    token2,
			Hostname: hostnameB,
			Wallet:   walletA.Name(),
		}, {
			Token:    token3,
			Hostname: hostnameB,
			Wallet:   walletB.Name(),
		}, {
			Token:    tokenOnNonExistingWallet,
			Hostname: hostnameB,
			Wallet:   nonExistingWallet, // Emulate a non-existing wallet.
		},
	}

	// setup
	manager := newTestManagerBuilder(t)
	manager.timeService.EXPECT().Now().AnyTimes().Return(time.Now())
	manager.walletStore.EXPECT().OnUpdate(gomock.Any()).Times(1)
	manager.tokenStore.EXPECT().OnUpdate(gomock.Any()).Times(1)
	manager.tokenStore.EXPECT().ListTokens().Times(1).Return(nil, nil)
	manager.sessionStore.EXPECT().ListSessions(gomock.Any()).Times(1).Return(previousSessions, nil)
	gomock.InOrder(
		manager.walletStore.EXPECT().WalletExists(gomock.Any(), walletA.Name()).Times(1).Return(true, nil),
		manager.walletStore.EXPECT().WalletExists(gomock.Any(), walletA.Name()).Times(1).Return(true, nil),
		manager.walletStore.EXPECT().WalletExists(gomock.Any(), walletB.Name()).Times(1).Return(true, nil),
		manager.walletStore.EXPECT().WalletExists(gomock.Any(), nonExistingWallet).Times(1).Return(false, nil),
	)
	manager.sessionStore.EXPECT().DeleteSession(gomock.Any(), tokenOnNonExistingWallet).Times(1).Return(nil)
	gomock.InOrder(
		manager.walletStore.EXPECT().IsWalletAlreadyUnlocked(gomock.Any(), walletA.Name()).Times(1).Return(false, nil),
		manager.walletStore.EXPECT().IsWalletAlreadyUnlocked(gomock.Any(), walletA.Name()).Times(1).Return(false, nil),
		manager.walletStore.EXPECT().IsWalletAlreadyUnlocked(gomock.Any(), walletB.Name()).Times(1).Return(true, nil),
	)
	manager.walletStore.EXPECT().GetWallet(gomock.Any(), walletB.Name()).Times(1).Return(walletB, nil)
	manager.Build()

	// when listing the session connections, all but the one with a non-existing
	// wallet should be returned.
	connectionList := manager.ListSessionConnections()

	// then
	assert.Equal(t, []api.Connection{
		{
			Hostname: hostnameA,
			Wallet:   walletA.Name(),
		}, {
			Hostname: hostnameB,
			Wallet:   walletA.Name(),
		}, {
			Hostname: hostnameB,
			Wallet:   walletB.Name(),
		},
	}, connectionList)

	// when verifying connections to walletB are full restored
	cw1, err := manager.ConnectedWallet(ctx, hostnameB, token3)

	// then
	require.Nil(t, err)
	assert.Equal(t, walletB.Name(), cw1.Name())
	assert.Equal(t, hostnameB, cw1.Hostname())

	// setup verifying connecting a closed connection trigger the passphrase
	// pipeline only once, and restore all connections associated to that wallet.
	manager.interactor.EXPECT().NotifyInteractionSessionBegan(ctx, traceID, api.WalletUnlockingWorkflow, uint8(2)).Times(1).Return(nil)
	manager.interactor.EXPECT().NotifyInteractionSessionEnded(ctx, traceID).Times(1)
	manager.interactor.EXPECT().RequestPassphrase(ctx, traceID, uint8(1), walletA.Name(), gomock.Any()).Times(1).Return(walletAPassphrase, nil)
	manager.walletStore.EXPECT().UnlockWallet(ctx, walletA.Name(), walletAPassphrase).Times(1).Return(nil)
	manager.walletStore.EXPECT().GetWallet(ctx, walletA.Name()).Times(1).Return(walletA, nil)
	manager.interactor.EXPECT().NotifySuccessfulRequest(ctx, traceID, uint8(2), gomock.Any()).Times(1)

	// when
	cw2, err := manager.ConnectedWallet(ctx, hostnameA, token1)

	// then
	require.Nil(t, err)
	assert.Equal(t, walletA.Name(), cw2.Name())
	assert.Equal(t, hostnameA, cw2.Hostname())

	// when
	cw3, err := manager.ConnectedWallet(ctx, hostnameB, token2)

	// then
	require.Nil(t, err)
	assert.Equal(t, walletA.Name(), cw3.Name())
	assert.Equal(t, hostnameB, cw3.Hostname())
}

type testManager struct {
	*connections.Manager
	timeService  *mocks.MockTimeService
	walletStore  *mocks.MockWalletStore
	tokenStore   *mocks.MockTokenStore
	sessionStore *mocks.MockSessionStore
	interactor   *apimocks.MockInteractor
}

func (tm *testManager) Build() {
	manager, err := connections.NewManager(
		tm.timeService,
		tm.walletStore,
		tm.tokenStore,
		tm.sessionStore,
		tm.interactor,
	)
	if err != nil {
		panic(fmt.Errorf("could not initialise the manager: %w", err))
	}

	tm.Manager = manager
}

func newTestManagerBuilder(t *testing.T) *testManager {
	t.Helper()

	ctrl := gomock.NewController(t)
	timeService := mocks.NewMockTimeService(ctrl)
	walletStore := mocks.NewMockWalletStore(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)
	sessionStore := mocks.NewMockSessionStore(ctrl)
	interactor := apimocks.NewMockInteractor(ctrl)

	return &testManager{
		timeService:  timeService,
		walletStore:  walletStore,
		tokenStore:   tokenStore,
		sessionStore: sessionStore,
		interactor:   interactor,
	}
}
