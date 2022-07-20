package abci

import (
	tmlog "github.com/tendermint/tendermint/libs/log"
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
