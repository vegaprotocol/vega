package paths

import (
	"fmt"
	"path/filepath"

	"github.com/adrg/xdg"
)

// The default Vega file structure is mapped on the XDG standard. This standard
// defines where the files should be looked for, depending on their purpose,
// through environment variables, prefixed by `$XDG_`. The value of these
// variables matches the standards of the platform the program runs on.
//
// More on XDG at:
// https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
//
// At each location, Vega files are grouped under a `vega` folder, as follows
// `$XDG_*/vega`, before being sorted in sub-folders. The file structure of
// these sub-folder is described in paths.go.
//
// Default file structure:
//
// $XDG_CACHE_HOME
// └── vega
//
// $XDG_CONFIG_HOME
// └── vega
//
// $XDG_DATA_HOME
// └── vega
//
// $XDG_STATE_HOME
// └── vega

type DefaultPaths struct{}

// CreateCachePathFor builds the default path for a cache file and creates
// intermediate directories, if needed.
func (p *DefaultPaths) CreateCachePathFor(relFilePath CachePath) (string, error) {
	return CreateDefaultCachePathFor(relFilePath)
}

// CreateCacheDirFor builds the default path for a cache directory and creates
// it, along with intermediate directories, if needed.
func (p *DefaultPaths) CreateCacheDirFor(relDirPath CachePath) (string, error) {
	return CreateDefaultCacheDirFor(relDirPath)
}

// CreateConfigPathFor builds the default path for a configuration file and
// creates intermediate directories, if needed.
func (p *DefaultPaths) CreateConfigPathFor(relFilePath ConfigPath) (string, error) {
	return CreateDefaultConfigPathFor(relFilePath)
}

// CreateConfigDirFor builds the default path for a config directory and creates
// it, along with intermediate directories, if needed.
func (p *DefaultPaths) CreateConfigDirFor(relDirPath ConfigPath) (string, error) {
	return CreateDefaultConfigDirFor(relDirPath)
}

// CreateDataPathFor builds the default path for a data file and creates
// intermediate directories, if needed.
func (p *DefaultPaths) CreateDataPathFor(relFilePath DataPath) (string, error) {
	return CreateDefaultDataPathFor(relFilePath)
}

// CreateDataDirFor builds the default path for a data directory and creates
// it, along with intermediate directories, if needed.
func (p *DefaultPaths) CreateDataDirFor(relDirPath DataPath) (string, error) {
	return CreateDefaultDataDirFor(relDirPath)
}

// CreateStatePathFor builds the default path for a state file and creates
// intermediate directories, if needed.
func (p *DefaultPaths) CreateStatePathFor(relFilePath StatePath) (string, error) {
	return CreateDefaultStatePathFor(relFilePath)
}

// CreateStateDirFor builds the default path for a state directory and creates
// it, along with intermediate directories, if needed.
func (p *DefaultPaths) CreateStateDirFor(relDirPath StatePath) (string, error) {
	return CreateDefaultStateDirFor(relDirPath)
}

// CachePathFor build the default path for a cache file or directory. It
// doesn't create any resources.
func (p *DefaultPaths) CachePathFor(relPath CachePath) string {
	return DefaultCachePathFor(relPath)
}

// ConfigPathFor build the default path for a config file or directory. It
// doesn't create any resources.
func (p *DefaultPaths) ConfigPathFor(relPath ConfigPath) string {
	return DefaultConfigPathFor(relPath)
}

// DataPathFor build the default path for a data file or directory. It
// doesn't create any resources.
func (p *DefaultPaths) DataPathFor(relPath DataPath) string {
	return DefaultDataPathFor(relPath)
}

// StatePathFor build the default path for a state file or directory. It
// doesn't create any resources.
func (p *DefaultPaths) StatePathFor(relPath StatePath) string {
	return DefaultStatePathFor(relPath)
}

// CreateDefaultCachePathFor builds the default path for a cache file and creates
// intermediate directories, if needed.
func CreateDefaultCachePathFor(relFilePath CachePath) (string, error) {
	path, err := xdg.CacheFile(filepath.Join(VegaHome, relFilePath.String()))
	if err != nil {
		return "", fmt.Errorf("couldn't create the default directory for file: %w", err)
	}
	return path, nil
}

// CreateDefaultCacheDirFor builds the default path for a cache directory and creates
// it, along with intermediate directories, if needed.
func CreateDefaultCacheDirFor(relDirPath CachePath) (string, error) {
	// We append fake-file to xdg library creates all directory up to fake-file.
	path, err := xdg.CacheFile(filepath.Join(VegaHome, relDirPath.String(), "fake-file"))
	if err != nil {
		return "", fmt.Errorf("couldn't create the default directory: %w", err)
	}
	return filepath.Dir(path), nil
}

// CreateDefaultConfigPathFor builds the default path for a configuration file and
// creates intermediate directories, if needed.
func CreateDefaultConfigPathFor(relFilePath ConfigPath) (string, error) {
	path, err := xdg.ConfigFile(filepath.Join(VegaHome, relFilePath.String()))
	if err != nil {
		return "", fmt.Errorf("couldn't create the default directory for file: %w", err)
	}
	return path, nil
}

// CreateDefaultConfigDirFor builds the default path for a config directory and creates
// it, along with intermediate directories, if needed.
func CreateDefaultConfigDirFor(relDirPath ConfigPath) (string, error) {
	// We append fake-file to xdg library creates all directory up to fake-file.
	path, err := xdg.ConfigFile(filepath.Join(VegaHome, relDirPath.String(), "fake-file"))
	if err != nil {
		return "", fmt.Errorf("couldn't create the default directory: %w", err)
	}
	return filepath.Dir(path), nil
}

// CreateDefaultDataPathFor builds the default path for a data file and creates
// intermediate directories, if needed.
func CreateDefaultDataPathFor(relFilePath DataPath) (string, error) {
	path, err := xdg.DataFile(filepath.Join(VegaHome, relFilePath.String()))
	if err != nil {
		return "", fmt.Errorf("couldn't create the default directory for file: %w", err)
	}
	return path, nil
}

// CreateDefaultDataDirFor builds the default path for a data directory and creates
// it, along with intermediate directories, if needed.
func CreateDefaultDataDirFor(relDirPath DataPath) (string, error) {
	// We append fake-file to xdg library creates all directory up to fake-file.
	path, err := xdg.DataFile(filepath.Join(VegaHome, relDirPath.String(), "fake-file"))
	if err != nil {
		return "", fmt.Errorf("couldn't create the default directory: %w", err)
	}
	return filepath.Dir(path), nil
}

// CreateDefaultStatePathFor builds the default path for a state file and creates
// intermediate directories, if needed.
func CreateDefaultStatePathFor(relFilePath StatePath) (string, error) {
	path, err := xdg.StateFile(filepath.Join(VegaHome, relFilePath.String()))
	if err != nil {
		return "", fmt.Errorf("couldn't create the default directory for file: %w", err)
	}
	return path, nil
}

// CreateDefaultStateDirFor builds the default path for a state directory and creates
// it, along with intermediate directories, if needed.
func CreateDefaultStateDirFor(relDirPath StatePath) (string, error) {
	// We append fake-file to xdg library creates all directory up to fake-file.
	path, err := xdg.StateFile(filepath.Join(VegaHome, relDirPath.String(), "fake-file"))
	if err != nil {
		return "", fmt.Errorf("couldn't create the default directory: %w", err)
	}
	return filepath.Dir(path), nil
}

// DefaultCachePathFor build the default path for a cache file or directory. It
// doesn't create any resources.
func DefaultCachePathFor(relPath CachePath) string {
	return filepath.Join(xdg.CacheHome, VegaHome, relPath.String())
}

// DefaultConfigPathFor build the default path for a config file or directory.
// It doesn't create any resources.
func DefaultConfigPathFor(relPath ConfigPath) string {
	return filepath.Join(xdg.ConfigHome, VegaHome, relPath.String())
}

// DefaultDataPathFor build the default path for a data file or directory. It
// doesn't create any resources.
func DefaultDataPathFor(relPath DataPath) string {
	return filepath.Join(xdg.DataHome, VegaHome, relPath.String())
}

// DefaultStatePathFor build the default path for a state file or directory. It
// doesn't create any resources.
func DefaultStatePathFor(relPath StatePath) string {
	return filepath.Join(xdg.StateHome, VegaHome, relPath.String())
}
