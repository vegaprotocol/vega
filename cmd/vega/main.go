package main

import (
	"fmt"
	"os"

	"vega/internal/logging"
)

func main() {

	// Set up the root logger
	log := logging.NewLoggerFromEnv("dev")
	defer log.AtExit()

	cli := NewCli()
	cli.SetFlags()

	base := &command{cmd: cli.rootCmd, cli: cli}
	base.Cmd().SilenceErrors = true

	cli.AddCommand(base, &NodeCommand{
		Log: log,
	})
	cli.AddCommand(base, &initCommand{
		Log: log,
	})

	if err := cli.Run(); err != nil {
		// deal with ExitError, which should be recognize as error, and should
		// not be exit with status 0.
		if exitErr, ok := err.(ExitError); ok {
			if exitErr.Status != "" {
				fmt.Fprintln(os.Stderr, exitErr.Status)
			}
			if exitErr.Code == 0 {
				// when get error with ExitError, code should not be 0.
				exitErr.Code = 1
			}
			os.Exit(exitErr.Code)
		}

		// not ExitError, print error to os.Stderr, exit code 1.
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// ExitError defines exit error produce by cli commands.
type ExitError struct {
	Code   int
	Status string
}

// Error implements the error interface.
func (e ExitError) Error() string {
	return fmt.Sprintf("Exit Code: %d, Status: %s", e.Code, e.Status)
}
