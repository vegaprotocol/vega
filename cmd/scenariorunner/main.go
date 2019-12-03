package main

import (
	"fmt"
	//"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/storage"

	"github.com/urfave/cli"
)

var (
	// VersionHash specifies the git commit used to build the application. Passed in via ldflags. See VERSION_HASH in Makefile for details.
	VersionHash = "unknown"
	// Version specifies the version used to build the application. Passed in via ldflags. See VERSION in Makefile for details.
	Version = "unknown"
	// Revision specifies app variation that was built to work with the VEGA version above.
	Revision = "0.0.1"
	// Logger
	log = logging.NewProdLogger()
)

func main() {
	app := cli.NewApp()
	runner := scenarioRunner{}
	info(app)
	commands(app, &runner)
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err.Error())
	}
}

func info(app *cli.App) {
	app.Name = "scenario-runner-cli"
	app.Usage = "Interact with a Vega node running without the consensus layer via command line."
	app.Description = "Command line tool interacting with a Vega node running without the consensus layer. It allows submission of instructions in bulk and persistence of respones along with the accompanying metadata."
	app.Version = fmt.Sprintf("%v for VEGA v.%v (%v)", Revision, Version, VersionHash)
}

func commands(app *cli.App, runner *scenarioRunner) {
	var optionalResultSetFile string
	var optionalProtocolSummaryFile string
	var configFile string

	var submit = "submit"
	app.Commands = []cli.Command{
		{
			Name:    submit,
			Aliases: []string{submit[:1]},
			Usage:   "Submits a batch of node instructions read from a JSON file - subcommands available, see 'help'.",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "result, r",
					Usage:       "Save instruction results set to a `FILE`. Files will be suffixed with a number when multiple instruction sets get submitted.",
					Destination: &optionalResultSetFile,
				},
				cli.StringFlag{
					Name:        "extract, e",
					Usage:       "Save protocol summary after successful execution of all instruction sets.",
					Destination: &optionalProtocolSummaryFile,
				},
				cli.StringFlag{
					Name:        "config, c",
					Usage:       "Specify config file. Default config used if omitted.",
					Destination: &configFile,
				},
			},
			Action: func(c *cli.Context) {
				dir, err := os.Getwd()
				if err != nil {
					log.Fatal(err.Error())
				}
				log.Info(dir)
				if c.NArg() > 0 {
					instrSet, err := ProcessFiles(c.Args())
					if err != nil {
						log.Fatal(err.Error())
					}
					n := len(instrSet)
					runner.lazyInit(configFile)
					defer runner.cleanUp()
					for i, instr := range instrSet {
						res, err := runner.engine.ProcessInstructions(*instr)
						if err != nil {
							log.Fatal(err.Error())
						}
						if optionalResultSetFile != "" {
							fileName := optionalResultSetFile
							if n > 1 {
								dir, file := filepath.Split(optionalResultSetFile)
								ext := filepath.Ext(file)
								fileName = fmt.Sprintf("%s%s_%vof%v%s", dir, strings.TrimSuffix(file, ext), i+1, n, ext)
							}
							Output(res, fileName)
						}
					}
					if optionalProtocolSummaryFile != "" {
						summary, err := runner.engine.ExtractData()
						if err != nil {
							log.Fatal(err.Error())
						}
						Output(summary, optionalProtocolSummaryFile)
					}

				} else {
					cli.ShowCommandHelp(c, submit)
				}
			},
		},
	}
}

type scenarioRunner struct {
	engineOnce    sync.Once
	engine        *Engine
	storageConfig storage.Config
}

func (s *scenarioRunner) lazyInit(configFileWithPath string) {
	s.engineOnce.Do(func() {
		config := NewDefaultConfig()

		storageConfig, storeErr := storage.NewTestConfig()
		if storeErr != nil {
			log.Fatal(storeErr.Error())
		}
		s.storageConfig = storageConfig
		if configFileWithPath != "" {
			f, err := os.Open(configFileWithPath)
			if err != nil {
				log.Fatal(err.Error())
			}
			err = unmarshal(f, &config)
			if err != nil {
				log.Fatal(err.Error())
			}
		}
		engine, engErr := NewEngine(log, config, s.storageConfig, Version)
		if engErr != nil {
			log.Fatal(engErr.Error())
		}
		s.engine = engine
	})
}

func (s *scenarioRunner) cleanUp() {
	storage.FlushStores(log, s.storageConfig)
}
