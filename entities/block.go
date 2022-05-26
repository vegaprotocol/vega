package entities

import (
	"encoding/hex"
	"time"

	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

type TimeUpdateEvent interface {
	events.Event
	Time() time.Time
}

type Block struct {
	VegaTime time.Time
	Height   int64
	Hash     []byte
}

func BlockFromTimeUpdate(te TimeUpdateEvent) (*Block, error) {
	hash, err := hex.DecodeString(te.TraceID())
	if err != nil {
		errors.Wrapf(err, "Trace ID is not valid hex string, trace ID:%s", te.TraceID())
	}

	// Postgres only stores timestamps in microsecond resolution
	block := Block{
		VegaTime: te.Time().Truncate(time.Microsecond),
		Hash:     hash,
		Height:   te.BlockNr(),
	}
	return &block, err
}
