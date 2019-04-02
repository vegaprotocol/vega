// This file will contain all hooks WRT "shutdown"
package main

import (
	"strings"

	"code.vegaprotocol.io/vega/internal/logging"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type errStack []error

func (l *NodeCommand) postRun(_ *cobra.Command, _ []string) error {
	var werr errStack
	if l.candleStore != nil {
		if err := l.candleStore.Close(); err != nil {
			werr = append(werr, errors.Wrap(err, "error closing candle store in command."))
		}
	}
	if l.riskStore != nil {
		if err := l.riskStore.Close(); err != nil {
			werr = append(werr, errors.Wrap(err, "error closing risk store in command."))
		}
	}
	if l.tradeStore != nil {
		if err := l.tradeStore.Close(); err != nil {
			werr = append(werr, errors.Wrap(err, "error closing trade store in command."))
		}
	}
	if l.orderStore != nil {
		if err := l.orderStore.Close(); err != nil {
			werr = append(werr, errors.Wrap(err, "error closing order store in command."))
		}
	}
	if l.marketStore != nil {
		if err := l.marketStore.Close(); err != nil {
			werr = append(werr, errors.Wrap(err, "error closing market store in command."))
		}
	}
	if l.partyStore != nil {
		if err := l.partyStore.Close(); err != nil {
			werr = append(werr, errors.Wrap(err, "error closing party store in command."))
		}
	}
	l.Log.Info("Vega shutdown complete",
		logging.String("version", Version),
		logging.String("version-hash", VersionHash))

	if len(werr) == 0 {
		// Prevent printing of empty error and exiting with non-zero code.
		return nil
	}
	return werr
}

func (l *NodeCommand) persistentPost(_ *cobra.Command, _ []string) {
	l.cfunc()
}

// Error - implement the error interface on the errStack type
func (e errStack) Error() string {
	s := make([]string, 0, len(e))
	for _, err := range e {
		s = append(s, err.Error())
	}
	return strings.Join(s, "\n")
}
