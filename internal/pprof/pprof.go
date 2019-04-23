package pprof

import (
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	"code.vegaprotocol.io/vega/internal/config/encoding"
	"code.vegaprotocol.io/vega/internal/fsutil"
	"code.vegaprotocol.io/vega/internal/logging"
)

const (
	pprofDir       = "pprof"
	memprofileFile = "mem.pprof"
	cpuprofileFile = "cpu.pprof"

	namedLogger = "pprof"
)

type Config struct {
	Level       encoding.LogLevel
	Enabled     bool
	Port        uint16
	ProfilesDir string
}

type Pprofhandler struct {
	Config

	log            *logging.Logger
	memprofilePath string
	cpuprofilePath string
}

// NewDefaultConfig create a new default configuration for the pprof handler
func NewDefaultConfig() Config {
	return Config{
		Level:       encoding.LogLevel{Level: logging.InfoLevel},
		Enabled:     false,
		Port:        6060,
		ProfilesDir: "/tmp",
	}
}

// New creates a new pprof handler
func New(log *logging.Logger, config Config) (*Pprofhandler, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

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
