package main

import (
	"code.vegaprotocol.io/vega/blockchain/recorder"

	"github.com/spf13/afero"
)

func replayAll(app recorder.ABCIApp, recordingPath string) error {
	rec, err := recorder.NewReplay(recordingPath, afero.NewOsFs())
	if err != nil {
		return err
	}
	return rec.Replay(app)
}
