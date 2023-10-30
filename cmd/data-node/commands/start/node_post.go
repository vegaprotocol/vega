// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package start

import (
	"strings"

	"code.vegaprotocol.io/vega/logging"

	"github.com/pkg/errors"
)

type errStack []error

func (l *NodeCommand) postRun(_ []string) error {
	var werr errStack

	postLog := l.Log.Named("postRun")

	if l.embeddedPostgres != nil {
		if err := l.embeddedPostgres.Stop(); err != nil {
			werr = append(werr, errors.Wrap(err, "error closing embedded postgres in command"))
		}
	}
	if l.pproffhandlr != nil {
		if err := l.pproffhandlr.Stop(); err != nil {
			werr = append(werr, errors.Wrap(err, "error stopping pprof"))
		}
	}

	postLog.Info("Vega datanode shutdown complete",
		logging.String("version", l.Version),
		logging.String("version-hash", l.VersionHash))

	postLog.Sync()

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

// Error - implement the error interface on the errStack type.
func (e errStack) Error() string {
	s := make([]string, 0, len(e))
	for _, err := range e {
		s = append(s, err.Error())
	}
	return strings.Join(s, "\n")
}
