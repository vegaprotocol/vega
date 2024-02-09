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

package abci

import (
	tmlog "github.com/cometbft/cometbft/libs/log"
	"go.uber.org/zap"
)

type TmLogger struct {
	log *zap.SugaredLogger
}

func (tl *TmLogger) Debug(msg string, keyVals ...interface{}) {
	tl.log.Debugw(msg, keyVals...)
}

func (tl *TmLogger) Info(msg string, keyVals ...interface{}) {
	tl.log.Infow(msg, keyVals...)
}

func (tl *TmLogger) Error(msg string, keyVals ...interface{}) {
	tl.log.Errorw(msg, keyVals...)
}

func (tl *TmLogger) With(keyVals ...interface{}) tmlog.Logger {
	tl.log.With(keyVals...)
	return tl
}
