package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	"code.vegaprotocol.io/vega/internal/logging"
)

const (
	pprofDir       = "pprof"
	memprofileFile = "mem.pprof"
	cpuprofileFile = "cpu.pprof"
)

type pprofhandler struct {
	memprofilePath string
	cpuprofilePath string
	log            *logging.Logger
}

func newpprof(log *logging.Logger, profileRootPath string) (*pprofhandler, error) {
	p := &pprofhandler{
		memprofilePath: filepath.Join(profileRootPath, pprofDir, memprofileFile),
		cpuprofilePath: filepath.Join(profileRootPath, pprofDir, cpuprofileFile),
		log:            log,
	}

	// start the pprof http server
	go func() {
		p.log.Error("pprof web server closed", logging.Error(http.ListenAndServe("localhost:6060", nil)))
	}()

	// start cpu and mem profilers
	if err := ensureDir(filepath.Join(profileRootPath, pprofDir)); err != nil {
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
func (p *pprofhandler) Stop() error {
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
