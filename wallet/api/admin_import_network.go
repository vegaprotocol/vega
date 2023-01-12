package api

import (
	"context"
	"errors"
	"fmt"

	vgfs "code.vegaprotocol.io/vega/libs/fs"
	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/network"

	"github.com/mitchellh/mapstructure"
)

var ErrInvalidNetworkSource = errors.New("invalid network source")

type AdminImportNetworkParams struct {
	Name      string `json:"name"`
	FilePath  string `json:"filePath"`
	URL       string `json:"url"`
	Overwrite bool   `json:"overwrite"`
}

type AdminImportNetworkResult struct {
	Name     string `json:"name"`
	FilePath string `json:"filePath"`
}

type AdminImportNetwork struct {
	networkStore NetworkStore
}

type Reader func(uri string, net interface{}) error

type Readers struct {
	ReadFromFile Reader
	ReadFromURL  Reader
}

func NewReaders() Readers {
	return Readers{
		ReadFromFile: paths.ReadStructuredFile,
		ReadFromURL:  paths.FetchStructuredFile,
	}
}

func (h *AdminImportNetwork) Handle(_ context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateImportNetworkParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	net, err := readImportNetworkSource(params)
	if errors.Is(err, ErrInvalidNetworkSource) {
		return nil, invalidParams(err)
	}
	if err != nil {
		return nil, internalError(err)
	}

	if len(params.Name) != 0 {
		net.Name = params.Name
	}

	if len(net.Name) == 0 {
		return nil, invalidParams(ErrNetworkNameIsRequired)
	}

	if exist, err := h.networkStore.NetworkExists(net.Name); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the network existence: %w", err))
	} else if exist && !params.Overwrite {
		return nil, invalidParams(ErrNetworkAlreadyExists)
	}

	if err := h.networkStore.SaveNetwork(net); err != nil {
		return nil, internalError(err)
	}

	return AdminImportNetworkResult{
		Name:     net.Name,
		FilePath: h.networkStore.GetNetworkPath(net.Name),
	}, nil
}

func validateImportNetworkParams(rawParams jsonrpc.Params) (AdminImportNetworkParams, error) {
	if rawParams == nil {
		return AdminImportNetworkParams{}, ErrParamsRequired
	}

	params := AdminImportNetworkParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminImportNetworkParams{}, ErrParamsDoNotMatch
	}

	if params.FilePath == "" && params.URL == "" {
		return AdminImportNetworkParams{}, ErrNetworkSourceIsRequired
	}

	if params.FilePath != "" && params.URL != "" {
		return AdminImportNetworkParams{}, ErrMultipleNetworkSources
	}

	return params, nil
}

// readImportNetworkSource parse the network file given by the source in the params
// into a `Network` which can then be saved to disk.
func readImportNetworkSource(params AdminImportNetworkParams) (*network.Network, error) {
	net := &network.Network{}
	rs := NewReaders()
	if len(params.FilePath) != 0 {
		exists, err := vgfs.FileExists(params.FilePath)
		if err != nil {
			return nil, fmt.Errorf("could not check file's existence at %q: %w", params.FilePath, err)
		}
		if !exists {
			return nil, fmt.Errorf("the network source file does not exist: %w", ErrInvalidNetworkSource)
		}

		err = rs.ReadFromFile(params.FilePath, net)
		if err == paths.ErrEmptyFile {
			return nil, fmt.Errorf("network source file is empty: %w", ErrInvalidNetworkSource)
		}
		if err != nil {
			return nil, fmt.Errorf("could not read the network configuration at %q: %w", params.FilePath, err)
		}
		return net, nil
	}

	if len(params.URL) != 0 {
		err := rs.ReadFromURL(params.URL, net)
		if err == paths.ErrEmptyResponse {
			return nil, fmt.Errorf("network source url points to an empty file: %w", ErrInvalidNetworkSource)
		}
		if err != nil {
			return nil, fmt.Errorf("could not fetch the network configuration from %q: %w", params.URL, err)
		}
		return net, nil
	}

	return net, nil
}

func NewAdminImportNetwork(
	networkStore NetworkStore,
) *AdminImportNetwork {
	return &AdminImportNetwork{
		networkStore: networkStore,
	}
}
