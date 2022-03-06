package storage

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"code.vegaprotocol.io/data-node/logging"
	vgfs "code.vegaprotocol.io/shared/libs/fs"
)

type ChainInfo struct {
	config          Config
	jsonFile        string
	log             *logging.Logger
	onCriticalError func()
	storedInfo      *storedInfo
}

type storedInfo struct {
	ChainID string
}

func NewChainInfo(log *logging.Logger, home string, c Config, onCriticalError func()) (*ChainInfo, error) {
	log = log.Named(namedLogger)
	log.SetLevel(c.Level.Get())
	jsonFile := filepath.Join(home, "info.json")

	chainInfo := ChainInfo{
		jsonFile:        jsonFile,
		log:             log,
		onCriticalError: onCriticalError,
	}

	// If the json file doesn't exist yet, create one with some default values
	if exists, _ := vgfs.FileExists(jsonFile); !exists {
		chainInfo.SetChainID("")
	}

	return &chainInfo, nil
}

func (e *ChainInfo) ReloadConf(cfg Config) {
	e.log.Info("reloading configuration")
	if e.log.GetLevel() != cfg.Level.Get() {
		e.log.Info("updating log level",
			logging.String("old", e.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		e.log.SetLevel(cfg.Level.Get())
	}

	e.config = cfg
}

func (c *ChainInfo) SetChainID(chainID string) error {
	data := storedInfo{ChainID: chainID}
	jsonData, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		c.log.Error("Unable to serialize chain info", logging.Error(err))
		c.onCriticalError()
	}

	err = ioutil.WriteFile(c.jsonFile, jsonData, 0644)
	if err != nil {
		c.log.Error("Unable to write chain info file: ",
			logging.String("file", c.jsonFile),
			logging.Error(err))
		c.onCriticalError()
	}
	// save the stored chain ID
	c.storedInfo = &data
	return err
}

func (c *ChainInfo) GetChainID() (string, error) {
	// if the chain ID is non nil and not empty return the cached value
	if c.storedInfo != nil && len(c.storedInfo.ChainID) > 0 {
		return c.storedInfo.ChainID, nil
	}

	jsonData, err := ioutil.ReadFile(c.jsonFile)
	if err != nil {
		c.log.Error("Unable to read chain info file: ",
			logging.String("file", c.jsonFile),
			logging.Error(err))
		c.onCriticalError()
		return "", err
	}

	var ci storedInfo
	err = json.Unmarshal(jsonData, &ci)
	if err != nil {
		c.log.Error("Unable to deserialize chain info", logging.Error(err))
		c.onCriticalError()
	}
	c.storedInfo = &ci
	return ci.ChainID, nil
}
