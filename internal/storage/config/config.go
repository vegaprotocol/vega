package config

import (
	"math/rand"
	"os"
	"path/filepath"
	"time"

	cfgencoding "code.vegaprotocol.io/vega/internal/config/encoding"

	"github.com/dgraph-io/badger/options"
)

const (
	defaultStorageAccessTimeout = 5 * time.Second
)

// BadgerOptions are params for creating a DB object.
type BadgerOptions struct {
	// Dir                  string // not customisable by end user
	// ValueDir             string // not customisable by end user
	SyncWrites              bool
	TableLoadingMode        cfgencoding.FileLoadingMode
	ValueLogLoadingMode     cfgencoding.FileLoadingMode
	NumVersionsToKeep       int
	MaxTableSize            int64
	LevelSizeMultiplier     int
	MaxLevels               int
	ValueThreshold          int
	NumMemtables            int
	NumLevelZeroTables      int
	NumLevelZeroTablesStall int
	LevelOneSize            int64
	ValueLogFileSize        int64
	ValueLogMaxEntries      uint32
	NumCompactors           int
	CompactL0OnClose        bool
	ReadOnly                bool
	Truncate                bool
	// Logger               logging.Logger // not customisable by end user
}

// DefaultBadgerOptions supplies default badger options to be used for all stores.
func DefaultBadgerOptions() BadgerOptions {
	/*
		Notes:
		* MaxTableSize: set low to avoid badger grabbing-then-releasing gigs of memory (#147)
		* ValueThreshold: set low to move most data out of the LSM tree (#147)
	*/
	fileio := cfgencoding.FileLoadingMode{FileLoadingMode: options.FileIO}
	opts := BadgerOptions{
		// Dir:                  TBD,       // string
		// ValueDir:             TBD,       // string
		SyncWrites:              true,      // bool
		TableLoadingMode:        fileio,    // options.FileLoadingMode, default options.MemoryMap
		ValueLogLoadingMode:     fileio,    // options.FileLoadingMode, default options.MemoryMap
		NumVersionsToKeep:       1,         // int
		MaxTableSize:            16 << 20,  // int64, default 64<<20 (64MB)
		LevelSizeMultiplier:     10,        // int
		MaxLevels:               7,         // int
		ValueThreshold:          16,        // int, default 32
		NumMemtables:            1,         // int, default 5
		NumLevelZeroTables:      1,         // int, default 5
		NumLevelZeroTablesStall: 2,         // int, default 10
		LevelOneSize:            64 << 20,  // int64, default 256<<20
		ValueLogFileSize:        1<<30 - 1, // int64, default 1<<30-1 (almost 1GB)
		ValueLogMaxEntries:      1000000,   // uint32
		NumCompactors:           2,         // int, default 2
		CompactL0OnClose:        true,      // bool
		ReadOnly:                false,     // bool
		Truncate:                false,     // bool
		// Logger:               TBD,       // Logger, default defaultLogger
	}
	return opts
}

type StorageConfig struct {
	Badger  BadgerOptions
	Path    string
	Timeout cfgencoding.Duration
}

func ensureDir(path string) error {
	const (
		dirPerms = 0700
	)

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(path, dirPerms)
		}
		return err
	}
	return nil
}

var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func newDefaultStorageConfig(basePath, storePath string) StorageConfig {
	return StorageConfig{
		Badger:  DefaultBadgerOptions(),
		Path:    filepath.Join(basePath, storePath),
		Timeout: cfgencoding.Duration{Duration: defaultStorageAccessTimeout},
	}
}
