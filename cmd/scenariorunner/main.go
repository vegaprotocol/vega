package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	sr "code.vegaprotocol.io/vega/scenariorunner"

	"github.com/urfave/cli"
)

var (
	app    = cli.NewApp()
	runner = scenariorunner{}
)

var (
	// VersionHash specifies the git commit used to build the application. See VERSION_HASH in Makefile for details.
	VersionHash = ""

	// Version specifies the version used to build the application. See VERSION in Makefile for details.
	Version = ""
)

func main() {
	info()
	commands()
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func info() {
	app.Name = "scenario-runner-cli"
	app.Usage = "Interact with a Vega node running without the consensus layer via command line."
	app.Description = "Command line tool interacting with a Vega node running without the consensus layer. It allows submission of instructions in bulk and persistence of respones along with the accompanying metadata."
	app.Version = Version
}

func commands() {
	var optionalResultSetFile string
	var optionalProtocolSummaryFile string
	var configFile string

	var submit = "submit"
	app.Commands = []cli.Command{
		{
			Name:    submit,
			Aliases: []string{submit[:1]},
			Usage:   "Submits a batch of node instructions read from a JSON file - subcommands available, see 'help'",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "result, r",
					Usage:       "Save instrution results set to a `FILE`. Files will be suffixed with a number when multiple instruction sets get submitted",
					Destination: &optionalResultSetFile,
				},
				cli.StringFlag{
					Name:        "extract, e",
					Usage:       "Save protocol summary after successful execution of all instruction sets",
					Destination: &optionalProtocolSummaryFile,
				},
				cli.StringFlag{
					Name:        "config, c",
					Usage:       "Use config file",
					Destination: &configFile,
				},
			},
			Action: func(c *cli.Context) {
				dir, err := os.Getwd()
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(dir)
				if c.NArg() > 0 {
					instrSet, err := ProcessFiles(c.Args())
					if err != nil {
						log.Fatal(err)
					}
					n := len(instrSet)
					runner.lazyInit(configFile)
					for i, instr := range instrSet {
						res, err := runner.engine.ProcessInstructions(*instr)
						if err != nil {
							log.Fatal(err)
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
							log.Fatal(err)
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

type scenariorunner struct {
	engineOnce sync.Once
	engine     *sr.Engine
}

func (s *scenariorunner) lazyInit(configFileWithPath string) {
	s.engineOnce.Do(func() {
		config := sr.NewDefaultConfig()

		if configFileWithPath != "" {
			f, err := os.Open(configFileWithPath)
			if err != nil {
				log.Fatal(err)
			}
			err = unmarshall(f, &config)
			if err != nil {
				log.Fatal(err)
			}
		}
		engine, err := sr.NewEngine(config)
		if err != nil {
			log.Fatal(err)
		}
		s.engine = engine
	})
}
