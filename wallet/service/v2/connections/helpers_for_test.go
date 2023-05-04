package connections_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/service/v2/connections"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertRightAllowedKeys(t *testing.T, expectedKeys []wallet.KeyPair, resultKeys []api.AllowedKey) {
	t.Helper()

	require.Len(t, resultKeys, len(expectedKeys))

	for i := 0; i < len(expectedKeys); i++ {
		assertRightAllowedKey(t, expectedKeys[i], resultKeys[i])
	}
}

func assertRightAllowedKey(t *testing.T, expectedKey wallet.KeyPair, resultKey api.AllowedKey) {
	t.Helper()

	assert.Equal(t, expectedKey.Name(), resultKey.Name())
	assert.Equal(t, expectedKey.PublicKey(), resultKey.PublicKey())
}

func randomTraceID(t *testing.T) (context.Context, string) {
	t.Helper()

	traceID := vgrand.RandomStr(64)
	return context.WithValue(context.Background(), jsonrpc.TraceIDKey, traceID), traceID
}

func randomWallet(t *testing.T) (wallet.Wallet, []wallet.KeyPair) {
	t.Helper()

	return randomWalletWithName(t, vgrand.RandomStr(5))
}

func randomWalletWithName(t *testing.T, walletName string) (wallet.Wallet, []wallet.KeyPair) {
	t.Helper()

	w, _, err := wallet.NewHDWallet(walletName)
	if err != nil {
		t.Fatalf("could not create wallet for test: %v", err)
	}

	kps := make([]wallet.KeyPair, 0, 3)
	for i := 0; i < 3; i++ {
		kp, err := w.GenerateKeyPair(nil)
		if err != nil {
			t.Fatalf("could not generate keys on wallet for test: %v", err)
		}
		kps = append(kps, kp)
	}

	return w, kps
}

func randomToken(t *testing.T) connections.Token {
	t.Helper()

	token, err := connections.AsToken(vgrand.RandomStr(64))
	if err != nil {
		panic(fmt.Errorf("could not create a random connection token: %w", err))
	}
	return token
}
