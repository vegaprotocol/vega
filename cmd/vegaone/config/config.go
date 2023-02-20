package config

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
)

const configFileName = "config.json"

type Config struct {
	WithDatanode bool
}

func Load(home string) (*Config, error) {
	buf, err := ioutil.ReadFile(filepath.Join(home, configFileName))
	if err != nil {
		return nil, err
	}

	c := &Config{}
	return c, json.Unmarshal(buf, c)
}

func Save(home string, c *Config) (string, error) {
	buf, err := json.Marshal(c)
	if err != nil {
		return "", err
	}

	path := filepath.Join(home, configFileName)
	return path, ioutil.WriteFile(path, buf, 0644)
}
