package main

import (
	"context"
	"net/http"
	"os"

	"code.vegaprotocol.io/vega/internal/auth/handler"
	"code.vegaprotocol.io/vega/internal/logging"

	"github.com/gorilla/handlers"
	"github.com/spf13/cobra"
)

type authCommand struct {
	command

	dbfile  string
	address string
	Log     *logging.Logger
}

func (a *authCommand) Init(c *Cli) {
	a.cli = c
	a.cmd = &cobra.Command{
		Use:   "auth",
		Short: "Start the auth server",
		Long:  "Start up the vega auth server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runAuth(args)
		},
	}

	fs := a.cmd.Flags()
	fs.StringVarP(&a.dbfile, "dbfile", "d", "db.json", "path to the json db file")
	fs.StringVarP(&a.address, "address", "a", "0.0.0.0:80", "address of the http server")
}

func (a *authCommand) runAuth(args []string) error {
	handler.InitJWT()

	svc := &handler.PartyService{File: a.dbfile}
	if err := svc.Load(); err != nil {
		a.Log.Warn("unable to load dbfile, a new one will be created at destinations",
			logging.Error(err),
		)
	}

	handler := handlers.CombinedLoggingHandler(os.Stdout, svc)

	s := &http.Server{
		Addr:    a.address,
		Handler: handler,
	}
	go func() {
		a.Log.Info("starting server",
			logging.String("http-address", a.address))
		a.Log.Error("http server exited",
			logging.Error(s.ListenAndServe()))
	}()

	ctx := context.Background()
	waitSig(ctx, a.Log)
	s.Shutdown(ctx)

	return nil
}
