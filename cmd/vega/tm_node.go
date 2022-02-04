package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/genesis"
	"github.com/spf13/cobra"
	tmabciclient "github.com/tendermint/tendermint/abci/client"
	tmcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	tmcfg "github.com/tendermint/tendermint/config"
	tmlog "github.com/tendermint/tendermint/libs/log"
	tmservice "github.com/tendermint/tendermint/libs/service"
	tmnode "github.com/tendermint/tendermint/node"
	tmtypes "github.com/tendermint/tendermint/types"
)

var (
	networkSelect        string
	networkSelectFromURL string
)

func NewRunNodeCmd(config *tmcfg.Config, logger tmlog.Logger) *cobra.Command {
	cmd := tmcmd.NewRunNodeCmd(customNewNode, config, logger)

	cmd.Flags().StringVar(
		&networkSelectFromURL,
		"network-url",
		"",
		"The URL to a genesis file to start this node with")
	cmd.Flags().StringVar(
		&networkSelect,
		"network",
		"",
		"The network to start this node with",
	)

	return cmd
}

func customNewNode(ctx context.Context, config *tmcfg.Config, logger tmlog.Logger) (tmservice.Service, error) {
	doc, err := getGenesisDoc(config)
	if err != nil {
		return nil, fmt.Errorf("couldn't get genesis document: %w", err)
	}
	// We are using tendermint as an external app, so remote create it is.
	remoteCreator := tmabciclient.NewRemoteCreator(logger, config.ProxyApp, config.ABCI, false)
	return tmnode.New(ctx, config, logger, remoteCreator, doc)
}

func getGenesisDoc(config *tmcfg.Config) (*tmtypes.GenesisDoc, error) {
	if len(networkSelect) > 0 {
		return genesisDocFromHTTP()
	} else if len(networkSelectFromURL) > 0 {
		return genesisDocHTTPFromURL()
	}

	return tmtypes.GenesisDocFromFile(config.GenesisFile())
}

func genesisDocFromHTTP() (*tmtypes.GenesisDoc, error) {
	genesisFilesRootPath := fmt.Sprintf("https://raw.githubusercontent.com/vegaprotocol/networks/master/%s", networkSelect)

	doc, state, err := getGenesisFromRemote(genesisFilesRootPath)
	if err != nil {
		return nil, err
	}

	sig, err := getSignatureFromRemote(genesisFilesRootPath)
	if err != nil {
		return nil, err
	}

	validSignature, err := genesis.VerifyGenesisStateSignature(state, sig)
	if err != nil {
		return nil, fmt.Errorf("couldn't verify the genesis state signature: %s", err)
	}
	if !validSignature {
		return nil, fmt.Errorf("genesis state doesn't match the signature: %s", sig)
	}

	return doc, nil
}

func genesisDocHTTPFromURL() (*tmtypes.GenesisDoc, error) {
	genesisFilePath := networkSelectFromURL

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", genesisFilePath, nil)
	if err != nil {
		return nil, fmt.Errorf("couldn't load genesis file from %s: %w", genesisFilePath, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("couldn't load genesis file from %s: %w", genesisFilePath, err)
	}
	defer resp.Body.Close()
	jsonGenesis, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	doc, _, err := genesis.GenesisFromJSON(jsonGenesis)
	if err != nil {
		return nil, fmt.Errorf("invalid genesis file from %s: %w", genesisFilePath, err)
	}

	return doc, nil
}

func getGenesisFromRemote(genesisFilesRootPath string) (*tmtypes.GenesisDoc, *genesis.GenesisState, error) {
	jsonGenesis, err := fetchData(fmt.Sprintf("%s/genesis.json", genesisFilesRootPath))
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't get remote genesis file: %w", err)
	}
	doc, state, err := genesis.GenesisFromJSON(jsonGenesis)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't parse genesis file: %w", err)
	}
	return doc, state, nil
}

func getSignatureFromRemote(genesisFilesRootPath string) (string, error) {
	sig, err := fetchData(fmt.Sprintf("%s/signature.txt", genesisFilesRootPath))
	if err != nil {
		return "", fmt.Errorf("couldn't get remote signature: %w", err)
	}
	return strings.Trim(string(sig), "\n"), nil
}

func fetchData(path string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("couldn't build request for %s: %w", path, err)
	}
	sigResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("couldn't get response for %s: %w", path, err)
	}
	defer sigResp.Body.Close()
	data, err := ioutil.ReadAll(sigResp.Body)
	if err != nil {
		return nil, fmt.Errorf("couldn't read response body: %w", err)
	}
	return data, nil
}
