package genesis

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	vgproto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/validators"
	"github.com/golang/protobuf/proto"

	"github.com/ethereum/go-ethereum/ethclient"
	tmconfig "github.com/tendermint/tendermint/config"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmos "github.com/tendermint/tendermint/libs/os"
	"github.com/tendermint/tendermint/privval"
	tmtypes "github.com/tendermint/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"
)

type generateCmd struct {
	DryRun  bool   `long:"dry-run" description:"Display the genesis file without writing it"`
	Network string `short:"n" long:"network" choice:"mainnet" choice:"testnet"`
	TmRoot  string `short:"t" long:"tm-root" description:"The root path of tendermint"`
}

func (opts *generateCmd) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	pass, err := genesisCmd.PassphraseFile.Get("node wallet")
	if err != nil {
		return err
	}

	vegaKey, ethAddress, err := loadNodeWalletPubKey(log, genesisCmd.RootPath, pass)
	if err != nil {
		return err
	}

	// genesis file
	tmConfig := tmconfig.DefaultConfig()
	tmConfig.SetRoot(os.ExpandEnv(opts.TmRoot))

	pubKey, err := loadTendermintPrivateValidatorKey(tmConfig)
	if err != nil {
		return err
	}
	genesisState := genesis.DefaultGenesisState()
	genesisState.Validators[base64.StdEncoding.EncodeToString(pubKey.Bytes())] = validators.ValidatorData{
		VegaPubKey:      vegaKey,
		EthereumAddress: ethAddress,
	}

	if len(opts.Network) != 0 {
		var ethConfig []byte
		switch opts.Network {
		case "mainnet":
			delete(genesisState.Assets, "VOTE")
			genesisState.Assets[assets.VegaTokenTestNet.Symbol] = assets.VegaTokenMainNet
			genesisState.NetParams[netparams.GovernanceVoteAsset] = assets.VegaTokenTestNet.Symbol
			ethConfig, err = proto.Marshal(&vgproto.EthereumConfig{
				NetworkId:     "1",
				ChainId:       "1",
				BridgeAddress: "0x4149257d844Ef09f11b02f2e73CbDfaB4c911a73",
				Confirmations: 50,
				StakingBridgeAddresses: []string{
					"0x195064D33f09e0c42cF98E665D9506e0dC17de68",
					"0x23d1bFE8fA50a167816fBD79D7932577c06011f4",
				},
			})
			if err != nil {
				return fmt.Errorf("couldn't marshal EthereumConfig: %w", err)
			}

		case "testnet":
			delete(genesisState.Assets, "VOTE")
			genesisState.Assets[assets.VegaTokenTestNet.Symbol] = assets.VegaTokenTestNet
			genesisState.NetParams[netparams.GovernanceVoteAsset] = assets.VegaTokenTestNet.Symbol
			ethConfig, err = proto.Marshal(&vgproto.EthereumConfig{
				NetworkId:              "3",
				ChainId:                "3",
				BridgeAddress:          "0x898b9F9f9Cab971d9Ceb809F93799109Abbe2D10",
				Confirmations:          3,
				StakingBridgeAddresses: []string{},
			})
			if err != nil {
				return fmt.Errorf("couldn't marshal EthereumConfig: %w", err)
			}
		default:
			return fmt.Errorf("network %s is not supported", opts.Network)
		}
		genesisState.NetParams[netparams.BlockchainsEthereumConfig] = string(ethConfig)
	}

	rawGenesisState, err := json.Marshal(genesisState)
	if err != nil {
		return err
	}

	genesisDoc := tmtypes.GenesisDoc{
		ChainID:         fmt.Sprintf("test-chain-%v", crypto.RandomStr(6)),
		GenesisTime:     tmtime.Now(),
		ConsensusParams: tmtypes.DefaultConsensusParams(),
		AppState:        rawGenesisState,
		Validators: []tmtypes.GenesisValidator{
			{
				Address: pubKey.Address(),
				PubKey:  pubKey,
				Power:   10,
			},
		},
	}

	if !opts.DryRun {
		genesisFilePath := tmConfig.GenesisFile()
		log.Infof("Saving genesis doc at %s", genesisFilePath)
		if err := genesisDoc.SaveAs(genesisFilePath); err != nil {
			return err
		}
	}

	marshalledGenesisDoc, err := tmjson.MarshalIndent(genesisDoc, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(marshalledGenesisDoc))
	return err
}

func loadTendermintPrivateValidatorKey(tmConfig *tmconfig.Config) (tmcrypto.PubKey, error) {
	privValKeyFile := tmConfig.PrivValidatorKeyFile()
	privValStateFile := tmConfig.PrivValidatorStateFile()
	if !tmos.FileExists(privValKeyFile) {
		return nil, fmt.Errorf("file \"%s\" not found", privValKeyFile)
	}

	pv := privval.LoadFilePV(privValKeyFile, privValStateFile)

	pubKey, err := pv.GetPubKey()
	if err != nil {
		return nil, fmt.Errorf("can't get pubkey: %w", err)
	}

	return pubKey, nil
}

func loadNodeWalletPubKey(log *logging.Logger, rootPath, pass string) (string, string, error) {
	conf, err := config.Read(rootPath)
	if err != nil {
		return "", "", err
	}

	ethClient, err := ethclient.Dial(conf.NodeWallet.ETH.Address)
	if err != nil {
		return "", "", err
	}

	nw, err := nodewallet.New(log, conf.NodeWallet, pass, ethClient, rootPath)
	if err != nil {
		return "", "", err
	}

	wVega, ok := nw.Get("vega")
	if !ok {
		return "", "", errors.New("no vega wallet stored in node wallet")
	}

	wEth, ok := nw.Get("ethereum")
	if !ok {
		return "", "", errors.New("no ethereum wallet stored in node wallet")
	}

	return wVega.PubKeyOrAddress().Hex(), wEth.PubKeyOrAddress().Hex(), nil
}
