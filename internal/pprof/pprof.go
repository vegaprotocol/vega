package pprof

import (
	"errors"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	"code.vegaprotocol.io/vega/internal/fsutil"
	"code.vegaprotocol.io/vega/internal/logging"
)

const (
	pprofDir       = "pprof"
	memprofileFile = "mem.pprof"
	cpuprofileFile = "cpu.pprof"

	namedLogger = "pprof"
)

var (
	ErrNilPPROFConfiguration = errors.New("nil pprof configuration")
)

type Config struct {
	log   *logging.Logger
	Level logging.Level

	Enabled     bool
	Port        uint16
	ProfilesDir string
}

type Pprofhandler struct {
	*Config

	memprofilePath string
	cpuprofilePath string
}

// NewDefaultConfig create a new default configuration for the pprof handler
func NewDefaultConfig(logger *logging.Logger) *Config {
	logger = logger.Named(namedLogger)
	return &Config{
		log:         logger,
		Level:       logging.InfoLevel,
		Enabled:     false,
		Port:        6060,
		ProfilesDir: "/tmp",
	}
}

// SetLogger creates a new logger based on a given parent logger.
func (c *Config) SetLogger(parent *logging.Logger) {
	c.log = parent.Named(namedLogger)
}

// UpdateLogger will set any new values on the underlying logging core. Useful when configs are
// hot reloaded at run time. Currently we only check and refresh the logging level.
func (c *Config) UpdateLogger() {
	c.log.SetLevel(c.Level)
}

// New creates a new pprof handler
func New(config *Config) (*Pprofhandler, error) {
	if config == nil {
		config.log.Error("cannot start pprof", logging.Error(ErrNilPPROFConfiguration))
		return nil, ErrNilPPROFConfiguration
	}
	p := &Pprofhandler{
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
