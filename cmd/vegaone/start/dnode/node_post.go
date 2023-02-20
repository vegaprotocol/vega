// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package dnode

import (
	"strings"

	"code.vegaprotocol.io/vega/logging"

	"github.com/pkg/errors"
)

type errStack []error

func (l *DN) postRun(_ []string) error {
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

func (l *DN) persistentPost(_ []string) error {
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
