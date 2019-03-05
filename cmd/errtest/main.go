package main

import (
	"fmt"
	"vega/internal/logging"
	"vega/proto"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func getSomeError() error {
	return proto.ErrInvalidMarketID
}

func main() {
	log := logging.NewLoggerFromEnv("dev")
	defer log.AtExit()

	fmt.Printf("LOG\n")
	err := getSomeError()
	log.Info("some order error", zap.Error(err))

	err2 := errors.Wrap(err, proto.ErrInvalidExpirationDatetime.Error())
	log.Error("wrapped errors", zap.Error(err2))

	fmt.Printf("\nPRINTF\n")
	fmt.Printf("%v\n", err)
	fmt.Printf("%v\n", err2)

}
