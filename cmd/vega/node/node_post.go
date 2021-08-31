package node

import (
	"code.vegaprotocol.io/vega/logging"
)

func (l *NodeCommand) postRun(_ []string) error {
	var err error

	if l.pproffhandlr != nil {
		err = l.pproffhandlr.Stop()
	}

	l.Log.Info("Vega shutdown complete",
		logging.String("version", l.Version),
		logging.String("version-hash", l.VersionHash))

	l.Log.Sync()

	return err
}

func (l *NodeCommand) persistentPost(_ []string) error {
	l.cancel()
	return nil
}
