package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	sr "code.vegaprotocol.io/vega/scenariorunner"

	"github.com/urfave/cli"
)

var app = cli.NewApp()
var ErrNotImplemented = errors.New("NotImplemented")
var engine *sr.Engine

func main() {
	info()
	commands()
	initializeEngine()
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func info() {
	app.Name = "scenario-runner-cli"
	app.Usage = "Interact with a Vega node running without the consensus layer via command line."
	app.Description = "Command line tool interacting with a Vega node running without the consensus layer. It allows submission of instructions in bulk and persistence of respones along with the accompanying metadata."
	app.Version = "0.0.0"
}

func commands() {
	var optionalOutputFile string

	var submit = "submit"
	var extract = "extract"
	app.Commands = []cli.Command{
		{
			Name:    submit,
			Aliases: []string{submit[:1]},
			Usage:   "Submits a batch of node instructions read from a JSON file - subcommands available, see 'help'",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "result, r",
					Usage:       "Save instrution results set to a `FILE`. Files will be suffixed with a number when multiple instruction sets get submitted",
					Destination: &optionalOutputFile,
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
						if optionalOutputFile != "" {
							fileName := optionalOutputFile
							if n > 1 {
								dir, file := filepath.Split(optionalOutputFile)
								ext := filepath.Ext(file)
								fileName = fmt.Sprintf("%s%s_%vof%v%s", dir, strings.TrimSuffix(file, ext), i+1, n, ext)
							}
							Output(res, fileName)
						}
					}

				} else {
					cli.ShowCommandHelp(c, submit)
				}
			},
		},
		{
			Name:    extract,
			Aliases: []string{extract[:1]},
			Usage:   "Save instrution results to a JSON file",
			Action: func(c *cli.Context) {
				if c.NArg() > 0 {
					fmt.Println("Extractdata", c.Args())
				} else {
					cli.ShowCommandHelp(c, extract)
				}
			},
		},
	}
}

func initializeEngine() {
	var err error
	engine, err = sr.NewEngine(sr.NewDefaultConfig())
	if err != nil {
		log.Fatal(err)
	}
}
