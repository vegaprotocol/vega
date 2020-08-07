package main

import (
	"fmt"
	"os"

	"code.vegaprotocol.io/vega/logging"
)

func main() {
	// Set up the root logger
	logDefaultConfig := logging.NewDefaultConfig()
	log := logging.NewLoggerFromConfig(logDefaultConfig)
	defer log.AtExit()

	cli := NewCli()

	base := &command{cmd: cli.rootCmd, cli: cli}
	base.Cmd().SilenceErrors = true

	cli.AddCommand(base, &NodeCommand{
		Log: log,
	})
	cli.AddCommand(base, &initCommand{
		Log: log,
	})

	cli.AddCommand(base, &gatewayCommand{
		Log: log,
	})

	cli.AddCommand(base, &walletCommand{
		Log: log,
	})

	cli.AddCommand(base, &faucetCommand{
		log: log,
	})

	cli.AddCommand(base, &nodeWalletCommand{
		Log: log,
	})

	if err := cli.Run(); err != nil {
		// deal with ExitError, which should be recognized as error, and should
		// not exit with status 0.
		if exitErr, ok := err.(ExitError); ok {
			log.Error("Command returned an error",
				logging.Int("code", exitErr.Code),
				logging.String("status", exitErr.Status))
			if exitErr.Code == 0 {
				// when get error with ExitError, code should not be 0.
				exitErr.Code = 1
			}
			os.Exit(exitErr.Code)
		}

		// not ExitError, print error to os.Stderr, exit code 1.
		log.Error("Command returned an unexpected error",
			logging.Error(err))
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
