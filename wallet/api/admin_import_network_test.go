package api_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminImportNetwork(t *testing.T) {
	t.Run("Documentation matches the code", testAdminImportNetworkSchemaCorrect)
	t.Run("Importing a network with invalid params fails", testImportingNetworkWithInvalidParamsFails)
	t.Run("Importing a network that already exists fails", testImportingNetworkThatAlreadyExistsFails)
	t.Run("Getting internal error during verification does not import the network", testGettingInternalErrorDuringVerificationDoesNotImportNetwork)
	t.Run("Importing a network from a file that doesn't exist fails", testImportingANetworkFromAFileThatDoesntExistFails)
	t.Run("Importing a network from a valid file saves", testImportingValidFileSaves)
	t.Run("Importing a network with no name fails", testImportingWithNoNameFails)
	t.Run("Importing a network from a valid file with name in config works", testImportingWithNameInConfig)
	t.Run("Importing a network with a github url suggests better alternative", testImportNetworkWithURL)
	t.Run("Importing a network with a content that is not TOML fails with a user friendly message", testImportNetworkWithNotTOMLContentFailsWithFriendlyMessage)
}

func testAdminImportNetworkSchemaCorrect(t *testing.T) {
	assertEqualSchema(t, "admin.import_network", api.AdminImportNetworkParams{}, api.AdminImportNetworkResult{})
}

func testImportingNetworkWithInvalidParamsFails(t *testing.T) {
	tcs := []struct {
		name          string
		params        interface{}
		expectedError error
	}{
		{
			name:          "with nil params",
			params:        nil,
			expectedError: api.ErrParamsRequired,
		}, {
			name:          "with wrong type of params",
			params:        "test",
			expectedError: api.ErrParamsDoNotMatch,
		}, {
			name: "with empty sources",
			params: api.AdminImportNetworkParams{
				Name: "fairground",
				URL:  "",
			},
			expectedError: api.ErrNetworkSourceIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newImportNetworkHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testImportingNetworkThatAlreadyExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)
	d := t.TempDir()
	filePath := filepath.Join(d + "tmp.toml")
	err := os.WriteFile(filePath, []byte("Name = \"local\""), 0o644)
	require.NoError(t, err)

	// setup
	handler := newImportNetworkHandler(t)

	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(gomock.Any()).Times(1).Return(true, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminImportNetworkParams{
		Name: name,
		URL:  api.FileSchemePrefix + filePath,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrNetworkAlreadyExists)
}

func testGettingInternalErrorDuringVerificationDoesNotImportNetwork(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)
	d := t.TempDir()
	filePath := filepath.Join(d + "tmp.toml")
	err := os.WriteFile(filePath, []byte("Name = \"local\""), 0o644)
	require.NoError(t, err)

	// setup
	handler := newImportNetworkHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(gomock.Any()).Times(1).Return(false, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminImportNetworkParams{
		Name: name,
		URL:  api.FileSchemePrefix + filePath,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the network existence: %w", assert.AnError))
}

func testImportingANetworkFromAFileThatDoesntExistFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newImportNetworkHandler(t)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminImportNetworkParams{
		Name: name,
		URL:  api.FileSchemePrefix + "some-file-path",
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, fmt.Errorf("the network source file does not exist: %w", api.ErrInvalidNetworkSource))
}

func testImportingValidFileSaves(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	d := t.TempDir()
	filePath := filepath.Join(d + "tmp.toml")
	err := os.WriteFile(filePath, []byte("Name = \"local\""), 0o644)
	require.NoError(t, err)

	// setup
	handler := newImportNetworkHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(name).Times(1).Return(false, nil)
	handler.networkStore.EXPECT().SaveNetwork(gomock.Any()).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminImportNetworkParams{
		Name: name,
		URL:  api.FileSchemePrefix + filePath,
	})

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, result.Name, name)
}

func testImportingWithNoNameFails(t *testing.T) {
	// given
	ctx := context.Background()

	d := t.TempDir()
	filePath := filepath.Join(d + "tmp.toml")
	err := os.WriteFile(filePath, []byte("Address = \"local\""), 0o644)
	require.NoError(t, err)

	// setup
	handler := newImportNetworkHandler(t)

	// when the config has no network name, and there is no network name specified in the params
	_, errorDetails := handler.handle(t, ctx, api.AdminImportNetworkParams{
		URL: api.FileSchemePrefix + filePath,
	})

	// then
	assertInvalidParams(t, errorDetails, api.ErrNetworkNameIsRequired)
}

func testImportingWithNameInConfig(t *testing.T) {
	// given
	ctx := context.Background()

	d := t.TempDir()
	filePath := filepath.Join(d + "tmp.toml")
	err := os.WriteFile(filePath, []byte("Name = \"local\""), 0o644)
	require.NoError(t, err)

	// setup
	handler := newImportNetworkHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists("local").Times(1).Return(false, nil)
	handler.networkStore.EXPECT().SaveNetwork(gomock.Any()).Times(1)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminImportNetworkParams{
		URL: api.FileSchemePrefix + filePath,
	})

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, result.Name, "local")
}

func testImportNetworkWithURL(t *testing.T) {
	// given
	ctx := context.Background()

	d := t.TempDir()
	filePath := filepath.Join(d + "tmp.toml")
	err := os.WriteFile(filePath, []byte("Name = \"local\""), 0o644)
	require.NoError(t, err)

	// setup
	_ = "network-path/local.toml"
	handler := newImportNetworkHandler(t)

	testCases := []struct {
		name       string
		url        string
		suggestion string
		jsonrpcErr jsonrpc.ErrorCode
	}{
		{
			name:       "real-url",
			url:        "https://github.com/vegaprotocol/networks-internal/blob/main/fairground/vegawallet-fairground.toml",
			suggestion: "https://raw.githubusercontent.com/vegaprotocol/networks-internal/main/fairground/vegawallet-fairground.toml",
			jsonrpcErr: jsonrpc.ErrorCodeInvalidParams,
		},
		{
			name:       "without s in http",
			url:        "http://github.com/blah/blob/main/fairground/network.toml",
			suggestion: "http://raw.githubusercontent.com/blah/main/fairground/network.toml",
			jsonrpcErr: jsonrpc.ErrorCodeInvalidParams,
		},
		{
			name:       "non-github url tries to fetch",
			url:        "https://example.com",
			jsonrpcErr: jsonrpc.ErrorCodeInternalError,
		},
		{
			name:       "missing .toml tries to fetch",
			url:        "https://github.com/vegaprotocol/vega/issues",
			jsonrpcErr: jsonrpc.ErrorCodeInternalError,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, errorDetails := handler.handle(t, ctx, api.AdminImportNetworkParams{
				URL: tc.url,
			})
			// then
			require.NotNil(t, errorDetails)
			assert.Equal(t, tc.jsonrpcErr, errorDetails.Code)
			if tc.suggestion != "" {
				require.Contains(t, errorDetails.Data, tc.suggestion)
			}
		})
	}
}

func testImportNetworkWithNotTOMLContentFailsWithFriendlyMessage(t *testing.T) {
	// given
	ctx := context.Background()
	d := t.TempDir()

	// setup
	handler := newImportNetworkHandler(t)

	tcs := []struct {
		name           string
		content        []byte
		identifiedType string
	}{
		{
			name:           "when HTML",
			content:        []byte("<!DOCTYPE html><html></html>"),
			identifiedType: "HTML",
		}, {
			name:           "when JSON",
			content:        []byte("{\"type\":\"JSON\"}"),
			identifiedType: "JSON",
		}, {
			name:           "when JSON",
			content:        []byte("{\"type\":\"JSON\"}"),
			identifiedType: "JSON",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			filePath := filepath.Join(d + "tmp.toml")
			err := os.WriteFile(filePath, tc.content, 0o644)
			require.NoError(tt, err)

			// when
			result, errorDetails := handler.handle(tt, ctx, api.AdminImportNetworkParams{
				URL: api.FileSchemePrefix + filePath,
			})

			// then
			require.NotNil(tt, errorDetails)
			assert.Equal(tt, fmt.Sprintf("could not read the network configuration at %q: the content looks like it contains %s, be sure your file has TOML formatting", filePath, tc.identifiedType), errorDetails.Data)
			assert.Empty(tt, result)
		})
	}
}

type importNetworkHandler struct {
	*api.AdminImportNetwork
	ctrl         *gomock.Controller
	networkStore *mocks.MockNetworkStore
}

func (h *importNetworkHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) (api.AdminImportNetworkResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.AdminImportNetworkResult)
		if !ok {
			t.Fatal("AdminImportWallet handler result is not a AdminImportWalletResult")
		}
		return result, err
	}
	return api.AdminImportNetworkResult{}, err
}

func newImportNetworkHandler(t *testing.T) *importNetworkHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	networkStore := mocks.NewMockNetworkStore(ctrl)

	return &importNetworkHandler{
		AdminImportNetwork: api.NewAdminImportNetwork(networkStore),
		ctrl:               ctrl,
		networkStore:       networkStore,
	}
}
