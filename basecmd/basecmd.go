package basecmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"code.vegaprotocol.io/vega/logging"
)

var (
	Version     string
	VersionHash string
)

type Command struct {
	Name    string
	Long    string
	Short   string
	Run     func(log *logging.Logger, args []string) int
	Usage   func()
	FlagSet *flag.FlagSet
}

func Main(cmds ...Command) {
	args := os.Args[1:]
	retval := 0

	log := buildLogger()

	if len(args) == 0 || args[0] == "help" {
		retval = printHelp(args, cmds)
	} else {
		var cmd Command
		var ok bool
		for _, v := range cmds {
			if v.Name == args[0] {
				cmd = v
				ok = true
				break
			}
		}
		if ok {
			retval = cmd.Run(log, args)
		} else {
			invalidCommand(args[0])
			retval = 1
		}

	}

	Exit(retval)
}

func buildLogger() *logging.Logger {
	logDefaultConfig := logging.NewDefaultConfig()
	return logging.NewLoggerFromConfig(logDefaultConfig)
}

func invalidCommand(cmd string) {
	str := `vega %s: unknown command
Run 'vega help for usage.'
`
	fmt.Fprintf(os.Stderr, str, cmd)
}

// waitSig will wait for a sigterm or sigint interrupt.
func WaitSig(ctx context.Context, log *logging.Logger) {
	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	select {
	case sig := <-gracefulStop:
		log.Info("Caught signal", logging.String("name", fmt.Sprintf("%+v", sig)))
	case <-ctx.Done():
		// nothing to do
	}
}

func EnvConfigPath() string {
	return os.Getenv("VEGA_CONFIG")
}
