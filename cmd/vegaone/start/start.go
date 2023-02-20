package start

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"code.vegaprotocol.io/vega/cmd/vegaone/config"
	"code.vegaprotocol.io/vega/cmd/vegaone/start/dnode"
	"code.vegaprotocol.io/vega/core/genesis"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	tmtypes "github.com/tendermint/tendermint/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Run(
	vegaPaths paths.Paths,
	tendermintHome string,
	networkURL, network string,
	passphraseFile string,
	c *config.Config,
) error {
	log := configureLogger()
	defer func() {
		log.AtExit()
	}()

	core, err := newCore(log, vegaPaths, tendermintHome, networkURL, network, passphraseFile)
	if err != nil {
		return err
	}

	var dn *dnode.DN

	// datanode is enabled, so start it up
	if c.WithDatanode {
		dn, err = dnode.New(log, vegaPaths)
		if err != nil {
			return fmt.Errorf("could not start datanode: %w", err)
		}
	}

	go dn.Start()
	go core.Start()

	// wait for an error or user input to exit
	wait(log, core, dn)

	core.Stop()
	if dn != nil {
		dn.Stop()
	}

	return nil
}

func wait(log *logging.Logger, core *Core, dn *dnode.DN) error {
	gracefulStop := make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT)
	if dn != nil {
		for {
			select {
			case sig := <-gracefulStop:
				log.Info("caught signal", logging.String("name", fmt.Sprintf("%+v", sig)))
				return nil
			case e := <-core.Err():
				log.Error("problem starting blockchain", logging.Error(e))
				return e
			case <-core.Done():
				// nothing to do
				return nil
			case <-dn.Done():
				// nothing to do
				return nil
			}
		}
	}

	for {
		select {
		case sig := <-gracefulStop:
			log.Info("caught signal", logging.String("name", fmt.Sprintf("%+v", sig)))
			return nil
		case e := <-core.Err():
			log.Error("problem starting blockchain", logging.Error(e))
			return e
		case <-core.ctx.Done():
			// nothing to do
			return nil
		}
	}
}

func configureLogger() *logging.Logger {
	cfg := zap.Config{
		Encoding:         "json",
		Level:            zap.NewAtomicLevelAt(zapcore.InfoLevel),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:  "message",
			LevelKey:    "level",
			NameKey:     "scope",
			EncodeLevel: zapcore.CapitalLevelEncoder,
			TimeKey:     "time",
			EncodeTime:  zapcore.RFC3339TimeEncoder,

			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}
	return logging.NewLoggerFromZapConfig(cfg)
}

func genesisDocHTTPFromURL(genesisFilePath string) (*tmtypes.GenesisDoc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, genesisFilePath, nil)
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

	doc, _, err := genesis.FromJSON(jsonGenesis)
	if err != nil {
		return nil, fmt.Errorf("invalid genesis file from %s: %w", genesisFilePath, err)
	}

	return doc, nil
}

func httpGenesisDocURLFromNetwork(networkSelect string) string {
	return fmt.Sprintf(
		"https://raw.githubusercontent.com/vegaprotocol/networks/master/%s",
		networkSelect,
	)
}
