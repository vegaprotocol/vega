package wallet

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/fsutil"
	vhttp "code.vegaprotocol.io/vega/http"
	"code.vegaprotocol.io/vega/logging"

	"github.com/zannen/toml"
)

const (
	namedLogger     = "wallet"
	configFile      = "wallet-service-config.toml"
	rsaKeyPath      = "wallet_rsa"
	pubRsaKeyName   = "public.pem"
	privRsaKeyName  = "private.pem"
	defaultCoolDown = 1 * time.Minute

	//  7 days, needs to be in seconds for the token
	tokenExpiry = time.Hour * 24 * 7
)

type Config struct {
	Level       encoding.LogLevel     `long:"level"`
	TokenExpiry encoding.Duration     `long:"token-expiry"`
	Port        int                   `long:"port"`
	IP          string                `long:"ip"`
	Node        NodeConfig            `group:"Node" namespace:"node"`
	RsaKey      string                `long:"rsa-key"`
	RateLimit   vhttp.RateLimitConfig `long:"rateLimit"`
}

type NodeConfig struct {
	Port    int    `long:"port"`
	IP      string `long:"ip"`
	Retries uint64 `long:"retries"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:       encoding.LogLevel{Level: logging.InfoLevel},
		TokenExpiry: encoding.Duration{Duration: tokenExpiry},
		Node: NodeConfig{
			IP:      "127.0.0.1",
			Port:    3002,
			Retries: 5,
		},
		IP:     "0.0.0.0",
		Port:   1789,
		RsaKey: rsaKeyPath,
		RateLimit: vhttp.RateLimitConfig{
			CoolDown: encoding.Duration{Duration: defaultCoolDown},
		},
	}
}

func LoadConfig(path string) (*Config, error) {
	buf, err := ioutil.ReadFile(filepath.Join(path, configFile))
	if err != nil {
		return nil, err
	}
	cfg := Config{}
	if _, err := toml.Decode(string(buf), &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func GenConfig(log *logging.Logger, path string, rewrite, genRsaKey bool) error {
	confPath := filepath.Join(path, configFile)

	confPathExists, _ := fsutil.PathExists(confPath)

	if confPathExists {
		if rewrite {
			log.Info("removing existing configuration",
				logging.String("path", confPath))
			err := os.Remove(confPath)
			if err != nil {
				return fmt.Errorf("unable to remove configuration: %v", err)
			}
		} else {
			// file exist, but not allowed to rewrite, return an error
			return fmt.Errorf("configuration already exists at path: %v", confPath)
		}
	}

	cfg := NewDefaultConfig()

	// write configuration to toml
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(cfg); err != nil {
		return err
	}

	// create the configuration file
	f, err := os.Create(confPath)
	if err != nil {
		return err
	}

	if _, err = f.WriteString(buf.String()); err != nil {
		return err
	}

	log.Info("wallet service configuration generated successfully", logging.String("path", confPath))

	if genRsaKey {
		if err := GenRsaKeyFiles(log, path, rewrite); err != nil {
			return err
		}
	}

	return nil
}

func GenRsaKeyFiles(log *logging.Logger, path string, rewrite bool) error {
	keyFolderPath := filepath.Join(path, rsaKeyPath)
	confPathExists, _ := fsutil.PathExists(keyFolderPath)
	if confPathExists {
		if rewrite {
			log.Info("removing existing rsa keys",
				logging.String("path", keyFolderPath))
			err := os.RemoveAll(keyFolderPath)
			if err != nil {
				return fmt.Errorf("unable to remove rsa keys: %v", err)
			}
		} else {
			// file exist, but not allowed to rewrite, return an error
			return fmt.Errorf("rsa keys already exists at path: %v", rsaKeyPath)
		}
	}

	// create the folder then
	if err := fsutil.EnsureDir(keyFolderPath); err != nil {
		return fmt.Errorf("unable to create the rsa key folder: %v", err)
	}

	bitSize := 4096

	key, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return fmt.Errorf("unable to generate rsa keys: %v", err)
	}

	if err := savePEMKey(filepath.Join(keyFolderPath, privRsaKeyName), key); err != nil {
		return fmt.Errorf("unable to write private key: %v", err)
	}

	if err := savePublicPEMKey(filepath.Join(keyFolderPath, pubRsaKeyName), key.PublicKey); err != nil {
		return fmt.Errorf("unable to write private key: %v", err)
	}

	log.Info("wallet rsa key generated successfully", logging.String("path", keyFolderPath))

	return nil
}

func savePEMKey(fileName string, key *rsa.PrivateKey) error {
	outFile, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer outFile.Close()

	var privateKey = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	return pem.Encode(outFile, privateKey)
}

func savePublicPEMKey(fileName string, pubkey rsa.PublicKey) error {
	pubBytes, err := x509.MarshalPKIXPublicKey(&pubkey)
	if err != nil {
		return err
	}

	var pemkey = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}

	pemfile, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer pemfile.Close()

	return pem.Encode(pemfile, pemkey)
}

func readRsaKeys(rootPath string) (pub []byte, priv []byte, err error) {
	pub, err = ioutil.ReadFile(filepath.Join(rootPath, rsaKeyPath, pubRsaKeyName))
	if err != nil {
		return nil, nil, err
	}
	priv, err = ioutil.ReadFile(filepath.Join(rootPath, rsaKeyPath, privRsaKeyName))
	if err != nil {
		return nil, nil, err
	}
	return
}
