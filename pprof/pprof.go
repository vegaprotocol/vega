package pprof

import (
	"fmt"
	"net/http"
	"time"

	// import pprof globally because it's used to init the package
	// and this comment is mostly here as well in order to make
	// golint very many much happy
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
)

const (
	pprofDir       = "pprof"
	memprofileName = "mem"
	cpuprofileName = "cpu"
	profileExt     = ".pprof"

	namedLogger = "pprof"
)

// Config represent the configuration of the pprof package
type Config struct {
	Level       encoding.LogLevel `long:"level"`
	Enabled     bool              `long:"enabled"`
	Port        uint16            `long:"port"`
	ProfilesDir string            `long:"profiles-dir"`
	// To include every blocking event in the profile, pass rate = 1.
	// To turn off profiling entirely, pass rate <= 0.
	BlockProfileRate int `long:"block-profile-rate"`
	// To turn off profiling entirely, pass rate 0.
	// To just read the current rate, pass rate < 0.
	// (For n>1 the details of sampling may change.)
	MutexProfileFraction int `long:"mutex-profile-fraction"`
}

// Pprofhandler is handling pprof profile management
type Pprofhandler struct {
	Config

	log            *logging.Logger
	memprofilePath string
	cpuprofilePath string
}

// NewDefaultConfig create a new default configuration for the pprof handler
func NewDefaultConfig() Config {
	return Config{
		Level:                encoding.LogLevel{Level: logging.InfoLevel},
		Enabled:              false,
		Port:                 6060,
		ProfilesDir:          "/tmp",
		BlockProfileRate:     0,
		MutexProfileFraction: 0,
	}
}

// New creates a new pprof handler
func New(log *logging.Logger, config Config) (*Pprofhandler, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	runtime.SetBlockProfileRate(config.BlockProfileRate)
	runtime.SetMutexProfileFraction(config.MutexProfileFraction)

	t := time.Now()
	memprofileFile := fmt.Sprintf("%s-%s%s", memprofileName, t.Format("2006-01-02-15-04-05"), profileExt)
	cpuprofileFile := fmt.Sprintf("%s-%s%s", cpuprofileName, t.Format("2006-01-02-15-04-05"), profileExt)

	p := &Pprofhandler{
		log:            log,
		Config:         config,
		memprofilePath: filepath.Join(config.ProfilesDir, pprofDir, memprofileFile),
		cpuprofilePath: filepath.Join(config.ProfilesDir, pprofDir, cpuprofileFile),
	}

	// start the pprof http server
	go func() {
		p.log.Error("pprof web server closed", logging.Error(http.ListenAndServe("localhost:6060", nil)))
	}()

	// start cpu and mem profilers
	if err := fsutil.EnsureDir(filepath.Join(config.ProfilesDir, pprofDir)); err != nil {
		p.log.Error("Could not create CPU profile file",
			logging.String("path", p.cpuprofilePath),
			logging.Error(err),
		)
		return nil, err
	}

	profileFile, err := os.Create(p.cpuprofilePath)
	if err != nil {
		p.log.Error("Could not create CPU profile file",
			logging.String("path", p.cpuprofilePath),
			logging.Error(err),
		)
		return nil, err
	}
	pprof.StartCPUProfile(profileFile)

	return p, nil
}

// ReloadConf update the configuration of the pprof package
func (p *Pprofhandler) ReloadConf(cfg Config) {
	p.log.Info("reloading configuration")
	if p.log.GetLevel() != cfg.Level.Get() {
		p.log.Info("updating log level",
			logging.String("old", p.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		p.log.SetLevel(cfg.Level.Get())
	}

	// the config will not be used anyway, just use the log level in here
	p.Config = cfg
	runtime.SetBlockProfileRate(cfg.BlockProfileRate)
	runtime.SetMutexProfileFraction(cfg.MutexProfileFraction)
}

// Stop is meant to be use to stop the pprof profile, should be use with defer probably
func (p *Pprofhandler) Stop() error {
	// stop cpu profile once the memory profile is written
	defer pprof.StopCPUProfile()

	p.log.Info("saving pprof memory profile", logging.String("path", p.memprofilePath))
	p.log.Info("saving pprof cpu profile", logging.String("path", p.cpuprofilePath))

	// save memory profile
	f, err := os.Create(p.memprofilePath)
	if err != nil {
		p.log.Error("Could not create memory profile file",
			logging.String("path", p.memprofilePath),
			logging.Error(err),
		)
		return err
	}

	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		p.log.Error("Could not write memory profile",
			logging.Error(err),
		)
		return err
	}
	f.Close()

	return nil
}
