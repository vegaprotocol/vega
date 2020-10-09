package node

import (
	"strings"

	"code.vegaprotocol.io/vega/logging"

	"github.com/pkg/errors"
)

type errStack []error

func (l *NodeCommand) postRun(_ []string) error {
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
	if l.accounts != nil {
		if err := l.accounts.Close(); err != nil {
			werr = append(werr, errors.Wrap(err, "error closing account store in command."))
		}
	}
	if l.transferResponseStore != nil {
		if err := l.transferResponseStore.Close(); err != nil {
			werr = append(werr, errors.Wrap(err, "error closing transfer response store in command."))
		}
	}
	if l.pproffhandlr != nil {
		if err := l.pproffhandlr.Stop(); err != nil {
			werr = append(werr, errors.Wrap(err, "error stopping pprof"))
		}
	}

	l.Log.Info("Vega shutdown complete",
		logging.String("version", l.Version),
		logging.String("version-hash", l.VersionHash))

	l.Log.Sync()

	if len(werr) == 0 {
		// Prevent printing of empty error and exiting with non-zero code.
		return nil
	}
	return werr
}

func (l *NodeCommand) persistentPost(_ []string) error {
	l.cancel()
	return nil
}

// Error - implement the error interface on the errStack type
func (e errStack) Error() string {
	s := make([]string, 0, len(e))
	for _, err := range e {
		s = append(s, err.Error())
	}
	return strings.Join(s, "\n")
}
