package v1

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	vgfs "code.vegaprotocol.io/vega/libs/fs"
	"code.vegaprotocol.io/vega/paths"

	"code.vegaprotocol.io/vega/wallet/network"
)

const fileExt = ".toml"

type Store struct {
	networksHome string
}

type networkFileContent struct {
	API network.APIConfig `json:"api"`
}

func InitialiseStore(vegaPaths paths.Paths) (*Store, error) {
	networksHome, err := vegaPaths.CreateConfigDirFor(paths.WalletServiceNetworksConfigHome)
	if err != nil {
		return nil, fmt.Errorf("couldn't get config path for %s: %w", paths.WalletServiceNetworksConfigHome, err)
	}

	return &Store{
		networksHome: networksHome,
	}, nil
}

func (s *Store) ListNetworks() ([]string, error) {
	networksParentDir, networksDir := filepath.Split(s.networksHome)
	entries, err := fs.ReadDir(os.DirFS(networksParentDir), networksDir)
	if err != nil {
		return nil, fmt.Errorf("couldn't read directory at %s: %w", s.networksHome, err)
	}
	networks := []string{}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".toml") {
			networks = append(networks, s.fileNameToName(entry.Name()))
		}
	}
	sort.Strings(networks)
	return networks, nil
}

func (s *Store) GetNetworksPath() string {
	return s.networksHome
}

func (s *Store) GetNetworkPath(name string) string {
	return s.nameToFilePath(name)
}

func (s *Store) NetworkExists(name string) (bool, error) {
	return vgfs.FileExists(s.GetNetworkPath(name))
}

func (s *Store) GetNetwork(name string) (*network.Network, error) {
	nfc := &networkFileContent{}
	if err := paths.ReadStructuredFile(s.nameToFilePath(name), &nfc); err != nil {
		return nil, fmt.Errorf("couldn't read network configuration file: %w", err)
	}
	return &network.Network{
		Name: name,
		API:  nfc.API,
	}, nil
}

func (s *Store) SaveNetwork(net *network.Network) error {
	nfc := &networkFileContent{
		API: net.API,
	}
	if err := paths.WriteStructuredFile(s.nameToFilePath(net.Name), nfc); err != nil {
		return fmt.Errorf("couldn't write network configuration file: %w", err)
	}
	return nil
}

func (s *Store) DeleteNetwork(name string) error {
	path := s.GetNetworkPath(name)
	return os.Remove(path)
}

func (s *Store) nameToFilePath(network string) string {
	return filepath.Join(s.networksHome, network+fileExt)
}

func (s *Store) fileNameToName(fileName string) string {
	return fileName[:len(fileName)-len(fileExt)]
}
