package main

import (
	"errors"
	"fmt"

	"log"
	"os"

	"github.com/urfave/cli"
)

var app = cli.NewApp()
var ErrNotImplemented = errors.New("NotImplemented")

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
					Name:        "extract, e",
					Usage:       "Save instrution results to a `FILE` (if all get submitted without errors)",
					Destination: &optionalOutputFile,
				},
			},
			Action: func(c *cli.Context) error {
				dir, err := os.Getwd()
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(dir)
				if c.NArg() > 0 {
					contents, err := readFiles(c.Args())
					if err != nil {
						return err
					}
					fmt.Println(contents)
					return ErrNotImplemented
					if optionalOutputFile != "" {
						return ErrNotImplemented
					}
				} else {
					cli.ShowCommandHelp(c, submit)
				}
				return nil
			},
		},
		{
			Name:    extract,
			Aliases: []string{extract[:1]},
			Usage:   "Save instrution results to a JSON file",
			Action: func(c *cli.Context) error {
				if c.NArg() > 0 {
					fmt.Println("Extractdata", c.Args())
				} else {
					cli.ShowCommandHelp(c, extract)
				}
				return nil
			},
		},
	}
}
