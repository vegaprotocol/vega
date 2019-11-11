package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	sr "code.vegaprotocol.io/vega/scenariorunner"

	"github.com/urfave/cli"
)

var (
	engine *sr.Engine

	// VersionHash specifies the git commit used to build the application. Passed in via ldflags
	VersionHash = "unknown"
	// Version specifies the version used to build the application. Passed in via ldflags
	Version = "unknown"
	// Revision specifies app variation that was built to work with the VEGA version above
	Revision = 0
)

func main() {
	app := cli.NewApp()
	info(app)
	commands(app)
	initializeEngine()
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func info(app *cli.App) {
	app.Name = "scenario-runner-cli"
	app.Usage = "Interact with a Vega node running without the consensus layer via command line."
	app.Description = "Command line tool interacting with a Vega node running without the consensus layer. It allows submission of instructions in bulk and persistence of respones along with the accompanying metadata."
	app.Version = fmt.Sprintf("%v (%v) / %d", Version, VersionHash, Revision)
}

func commands(app *cli.App) {
	var optionalResultSetFile string
	var optionalProtocolSummaryFile string

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
					for i, instr := range instrSet {
						res, err := engine.ProcessInstructions(*instr)
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
						summary, err := engine.ExtractData()
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

func initializeEngine() {
	var err error
	// TODO (WG 08/11/2019): Read from file
	config := sr.NewDefaultConfig()
	engine, err = sr.NewEngine(config)
	if err != nil {
		log.Fatal(err)
	}
}
