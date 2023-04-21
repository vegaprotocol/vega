package v1

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	vgfs "code.vegaprotocol.io/vega/libs/fs"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/api"

	"code.vegaprotocol.io/vega/wallet/network"
)

const fileExt = ".toml"

var (
	ErrNetworkNameCannotBeEmpty      = errors.New("the network name cannot be empty")
	ErrNetworkNameCannotStartWithDot = errors.New("the network name cannot start with a `.`")
	ErrNetworkNameCannotContainSlash = errors.New("the network name cannot contain `\\` nor `/`")
)

type FileStore struct {
	networksHome string
}

type networkFileContentV1 struct {
	FileVersion uint32            `json:"fileVersion"`
	API         network.APIConfig `json:"api"`
	TokenDApp   struct {
		URL string `json:"url"`
	} `json:"tokenDApp"`
	Console struct {
		URL string `json:"url"`
	} `json:"console"`
}

type networkFileContentV2 struct {
	FileVersion uint32             `json:"fileVersion"`
	API         network.APIConfig  `json:"api"`
	Apps        network.AppsConfig `json:"apps"`
	Metadata    []network.Metadata `json:"metadata"`
}

func InitialiseStore(vegaPaths paths.Paths) (*FileStore, error) {
	networksHome, err := vegaPaths.CreateConfigDirFor(paths.WalletServiceNetworksConfigHome)
	if err != nil {
		return nil, fmt.Errorf("couldn't get config path for %s: %w", paths.WalletServiceNetworksConfigHome, err)
	}

	return &FileStore{
		networksHome: networksHome,
	}, nil
}

func (s *FileStore) ListNetworks() ([]string, error) {
	networksParentDir, networksDir := filepath.Split(s.networksHome)
	entries, err := fs.ReadDir(os.DirFS(networksParentDir), networksDir)
	if err != nil {
		return nil, fmt.Errorf("couldn't read directory at %s: %w", s.networksHome, err)
	}
	networks := []string{}
	for _, entry := range entries {
		if err := ensureValidNetworkName(entry.Name()); err != nil {
			continue
		}

		if strings.HasSuffix(entry.Name(), ".toml") {
			networks = append(networks, s.fileNameToName(entry.Name()))
		}
	}
	sort.Strings(networks)
	return networks, nil
}

func (s *FileStore) GetNetworksPath() string {
	return s.networksHome
}

func (s *FileStore) GetNetworkPath(name string) string {
	return s.nameToFilePath(name)
}

func (s *FileStore) NetworkExists(name string) (bool, error) {
	if err := ensureValidNetworkName(name); err != nil {
		return false, err
	}

	return vgfs.FileExists(s.GetNetworkPath(name))
}

func (s *FileStore) GetNetwork(name string) (*network.Network, error) {
	if err := ensureValidNetworkName(name); err != nil {
		return nil, err
	}

	versionedFile := struct {
		FileVersion int32 `json:"fileVersion"`
	}{}

	if err := paths.ReadStructuredFile(s.nameToFilePath(name), &versionedFile); err != nil {
		return nil, fmt.Errorf("couldn't read network configuration file: %w", err)
	}

	if versionedFile.FileVersion == 2 {
		nfc := &networkFileContentV2{}
		if err := paths.ReadStructuredFile(s.nameToFilePath(name), &nfc); err != nil {
			return nil, fmt.Errorf("couldn't read network configuration file (v2): %w", err)
		}

		return &network.Network{
			Name:     name,
			Metadata: nfc.Metadata,
			API:      nfc.API,
			Apps: network.AppsConfig{
				Console:    nfc.Apps.Console,
				Governance: nfc.Apps.Governance,
				Explorer:   nfc.Apps.Explorer,
			},
		}, nil
	}

	nfc := &networkFileContentV1{}
	if err := paths.ReadStructuredFile(s.nameToFilePath(name), &nfc); err != nil {
		return nil, fmt.Errorf("couldn't read network configuration file (v1): %w", err)
	}

	return &network.Network{
		Name: name,
		API:  nfc.API,
		Apps: network.AppsConfig{
			Console:    nfc.Console.URL,
			Governance: nfc.TokenDApp.URL,
		},
	}, nil
}

func (s *FileStore) SaveNetwork(net *network.Network) error {
	if err := ensureValidNetworkName(net.Name); err != nil {
		return err
	}

	nfc := &networkFileContentV2{
		FileVersion: 2,
		API:         net.API,
		Apps:        net.Apps,
		Metadata:    net.Metadata,
	}
	if err := paths.WriteStructuredFile(s.nameToFilePath(net.Name), nfc); err != nil {
		return fmt.Errorf("couldn't write network configuration file: %w", err)
	}
	return nil
}

func (s *FileStore) DeleteNetwork(name string) error {
	if err := ensureValidNetworkName(name); err != nil {
		return err
	}
	path := s.GetNetworkPath(name)
	return os.Remove(path)
}

func (s *FileStore) RenameNetwork(currentName, newName string) error {
	if err := ensureValidNetworkName(currentName); err != nil {
		return err
	}

	if err := ensureValidNetworkName(newName); err != nil {
		return err
	}

	currentNetworkPath := s.nameToFilePath(currentName)

	if exists, err := vgfs.PathExists(currentNetworkPath); err != nil {
		return fmt.Errorf("could not verify the path at %s: %w", currentNetworkPath, err)
	} else if !exists {
		return api.ErrNetworkDoesNotExist
	}

	newNetworkPath := s.nameToFilePath(newName)

	if err := os.Rename(currentNetworkPath, newNetworkPath); err != nil {
		return fmt.Errorf("could not rename the network %q to %q at %s: %w", currentName, newName, s.networksHome, err)
	}

	return nil
}

func (s *FileStore) nameToFilePath(network string) string {
	return filepath.Join(s.networksHome, network+fileExt)
}

func (s *FileStore) fileNameToName(fileName string) string {
	return fileName[:len(fileName)-len(fileExt)]
}

func ensureValidNetworkName(name string) error {
	if name == "" {
		return ErrNetworkNameCannotBeEmpty
	}
	if name[0] == '.' {
		return ErrNetworkNameCannotStartWithDot
	}

	if strings.ContainsAny(name, "\\/") {
		return ErrNetworkNameCannotContainSlash
	}

	return nil
}
