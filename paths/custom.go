package paths

import (
	"fmt"
	"path/filepath"

	vgfs "code.vegaprotocol.io/shared/libs/fs"
)

// When opting for a custom Vega home, the all files are located under the
// specified folder. They are sorted, by purpose, in sub-folders. The structure
// of these sub-folder is described in paths.go.
//
// File structure for custom home:
//
// VEGA_HOME
// 	├── cache/
// 	├── config/
// 	├── data/
// 	└── state/

type CustomPaths struct {
	CustomHome string
}

// CreateCacheDirFor builds the path for cache files at the configured home and
// creates intermediate directories.
func (p *CustomPaths) CreateCacheDirFor(relDirPath CachePath) (string, error) {
	return CreateCustomCacheDirFor(p.CustomHome, relDirPath)
}

// CreateCachePathFor builds the path for cache directories at the configured home
// and creates intermediate directories.
func (p *CustomPaths) CreateCachePathFor(relFilePath CachePath) (string, error) {
	return CreateCustomCachePathFor(p.CustomHome, relFilePath)
}

// CreateConfigDirFor builds the path for configuration files at a given configured
// home and creates intermediate directories.
func (p *CustomPaths) CreateConfigDirFor(relDirPath ConfigPath) (string, error) {
	return CreateCustomConfigDirFor(p.CustomHome, relDirPath)
}

// CreateConfigPathFor builds the path for config directories at the configured
// home and creates intermediate directories.
func (p *CustomPaths) CreateConfigPathFor(relFilePath ConfigPath) (string, error) {
	return CreateCustomConfigPathFor(p.CustomHome, relFilePath)
}

// CreateDataDirFor builds the path for data files at the configured home and
// creates intermediate directories.
func (p *CustomPaths) CreateDataDirFor(relDirPath DataPath) (string, error) {
	return CreateCustomDataDirFor(p.CustomHome, relDirPath)
}

// CreateDataPathFor builds the path for data directories at the configured home
// and creates intermediate directories.
func (p *CustomPaths) CreateDataPathFor(relFilePath DataPath) (string, error) {
	return CreateCustomDataPathFor(p.CustomHome, relFilePath)
}

// CreateStateDirFor builds the path for cache files at the configured home and
// creates intermediate directories.
func (p *CustomPaths) CreateStateDirFor(relDirPath StatePath) (string, error) {
	return CreateCustomStateDirFor(p.CustomHome, relDirPath)
}

// CreateStatePathFor builds the path for data directories at the configured home
// and creates intermediate directories.
func (p *CustomPaths) CreateStatePathFor(relFilePath StatePath) (string, error) {
	return CreateCustomStatePathFor(p.CustomHome, relFilePath)
}

// CachePathFor builds the path for a cache file or directories at the
// configured home. It doesn't create any resources.
func (p *CustomPaths) CachePathFor(relPath CachePath) string {
	return CustomCachePathFor(p.CustomHome, relPath)
}

// ConfigPathFor builds the path for a config file or directories at the
// configured home. It doesn't create any resources.
func (p *CustomPaths) ConfigPathFor(relPath ConfigPath) string {
	return CustomConfigPathFor(p.CustomHome, relPath)
}

// DataPathFor builds the path for a data file or directories at the configured
// home. It doesn't create any resources.
func (p *CustomPaths) DataPathFor(relPath DataPath) string {
	return CustomDataPathFor(p.CustomHome, relPath)
}

// StatePathFor builds the path for a state file or directories at the
// configured home. It doesn't create any resources.
func (p *CustomPaths) StatePathFor(relPath StatePath) string {
	return CustomStatePathFor(p.CustomHome, relPath)
}

// CreateCustomCachePathFor builds the path for cache files at a given root path and
// creates intermediate directories. It scoped the files under a "cache" folder,
// and follow the default structure.
func CreateCustomCachePathFor(customHome string, relFilePath CachePath) (string, error) {
	fullPath := CustomCachePathFor(customHome, relFilePath)
	dir := filepath.Dir(fullPath)
	if err := vgfs.EnsureDir(dir); err != nil {
		return "", fmt.Errorf("couldn't create directories for file: %w", err)
	}
	return fullPath, nil
}

// CreateCustomCacheDirFor builds the path for cache directories at a given root path
// and creates intermediate directories. It scoped the files under a "data"
// folder, and follow the default structure.
func CreateCustomCacheDirFor(customHome string, relDirPath CachePath) (string, error) {
	path := CustomCachePathFor(customHome, relDirPath)
	if err := vgfs.EnsureDir(path); err != nil {
		return "", fmt.Errorf("couldn't create directories: %w", err)
	}
	return path, nil
}

// CreateCustomConfigPathFor builds the path for configuration files at a given root
// path and creates intermediate directories. It scoped the files under a
// "config" folder, and follow the default structure.
func CreateCustomConfigPathFor(customHome string, relFilePath ConfigPath) (string, error) {
	fullPath := CustomConfigPathFor(customHome, relFilePath)
	dir := filepath.Dir(fullPath)
	if err := vgfs.EnsureDir(dir); err != nil {
		return "", fmt.Errorf("couldn't create directories for file: %w", err)
	}
	return fullPath, nil
}

// CreateCustomConfigDirFor builds the path for config directories at a given root path
// and creates intermediate directories. It scoped the files under a "data"
// folder, and follow the default structure.
func CreateCustomConfigDirFor(customHome string, relDirPath ConfigPath) (string, error) {
	path := CustomConfigPathFor(customHome, relDirPath)
	if err := vgfs.EnsureDir(path); err != nil {
		return "", fmt.Errorf("couldn't create directories: %w", err)
	}
	return path, nil
}

// CreateCustomDataPathFor builds the path for data files at a given root path and
// creates intermediate directories. It scoped the files under a "data" folder,
// and follow the default structure.
func CreateCustomDataPathFor(customHome string, relFilePath DataPath) (string, error) {
	fullPath := CustomDataPathFor(customHome, relFilePath)
	dir := filepath.Dir(fullPath)
	if err := vgfs.EnsureDir(dir); err != nil {
		return "", fmt.Errorf("couldn't create directories for file: %w", err)
	}
	return fullPath, nil
}

// CreateCustomDataDirFor builds the path for data directories at a given root path
// and creates intermediate directories. It scoped the files under a "data"
// folder, and follow the default structure.
func CreateCustomDataDirFor(customHome string, relDirPath DataPath) (string, error) {
	path := CustomDataPathFor(customHome, relDirPath)
	if err := vgfs.EnsureDir(path); err != nil {
		return "", fmt.Errorf("couldn't create directories: %w", err)
	}
	return path, nil
}

// CreateCustomStatePathFor builds the path for cache files at a given root path and
// creates intermediate directories. It scoped the files under a "cache" folder,
// and follow the default structure.
func CreateCustomStatePathFor(customHome string, relFilePath StatePath) (string, error) {
	fullPath := CustomStatePathFor(customHome, relFilePath)
	dir := filepath.Dir(fullPath)
	if err := vgfs.EnsureDir(dir); err != nil {
		return "", fmt.Errorf("couldn't create directories for file: %w", err)
	}
	return fullPath, nil
}

// CreateCustomStateDirFor builds the path for data directories at a given root path
// and creates intermediate directories. It scoped the files under a "data"
// folder, and follow the default structure.
func CreateCustomStateDirFor(customHome string, relDirPath StatePath) (string, error) {
	path := CustomStatePathFor(customHome, relDirPath)
	if err := vgfs.EnsureDir(path); err != nil {
		return "", fmt.Errorf("couldn't create directories: %w", err)
	}
	return path, nil
}

// CustomCachePathFor builds the path for a cache file or directories at a given
// root path. It doesn't create any resources.
func CustomCachePathFor(customHome string, relPath CachePath) string {
	return filepath.Join(customHome, "cache", relPath.String())
}

// CustomConfigPathFor builds the path for a config file or directories at a given
// root path. It doesn't create any resources.
func CustomConfigPathFor(customHome string, relPath ConfigPath) string {
	return filepath.Join(customHome, "config", relPath.String())
}

// CustomDataPathFor builds the path for a data file or directories at a given
// root path. It doesn't create any resources.
func CustomDataPathFor(customHome string, relPath DataPath) string {
	return filepath.Join(customHome, "data", relPath.String())
}

// CustomStatePathFor builds the path for a state file or directories at a given
// root path. It doesn't create any resources.
func CustomStatePathFor(customHome string, relPath StatePath) string {
	return filepath.Join(customHome, "state", relPath.String())
}
