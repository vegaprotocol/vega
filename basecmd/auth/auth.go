package auth

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"code.vegaprotocol.io/vega/auth/handler"
	"code.vegaprotocol.io/vega/basecmd"
	"code.vegaprotocol.io/vega/logging"
	"github.com/gorilla/handlers"
	"github.com/rs/cors"
)

var (
	Command basecmd.Command

	address string
	dbfile  string
)

func init() {
	Command.Name = "auth"
	Command.Short = "Start a new vega auth server"

	cmd := flag.NewFlagSet("auth", flag.ContinueOnError)
	cmd.StringVar(&address, "address", "0.0.0.0:80", "address of the http server")
	cmd.StringVar(&dbfile, "dbfile", "db.json", "path of the json db file")

	cmd.Usage = func() {
		fmt.Fprintf(cmd.Output(), "%v\n\n", helpAuth())
		cmd.PrintDefaults()
	}

	Command.FlagSet = cmd
	Command.Usage = Command.FlagSet.Usage
	Command.Run = runCommand
}

func helpAuth() string {
	helpStr := `
Usage: vega auth [options]
`
	return strings.TrimSpace(helpStr)
}

func runCommand(log *logging.Logger, args []string) int {
	if err := Command.FlagSet.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		fmt.Fprintf(Command.FlagSet.Output(), "%v\n", err)
		return 1
	}

	if len(address) <= 0 {
		fmt.Fprintln(os.Stderr, "address parameter cannot be empty")
		return 1
	}
	if len(address) <= 0 {
		fmt.Fprintln(os.Stderr, "dbfile parameter cannot be empty")
		return 1
	}

	handler.InitJWT()

	corz := cors.AllowAll()
	svc := &handler.PartyService{File: dbfile}
	if err := svc.Load(); err != nil {
		log.Warn("unable to load dbfile, a new one will be created at destinations",
			logging.Error(err),
		)
	}

	handler := handlers.CombinedLoggingHandler(os.Stdout, svc)
	handler = corz.Handler(handler)

	s := &http.Server{
		Addr:    address,
		Handler: handler,
	}
	go func() {
		log.Info("starting server",
			logging.String("http-address", address))
		log.Error("http server exited",
			logging.Error(s.ListenAndServe()))
	}()

	ctx := context.Background()
	basecmd.WaitSig(ctx, log)
	s.Shutdown(ctx)

	return 0
}
