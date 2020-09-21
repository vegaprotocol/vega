package recorder

import "github.com/tendermint/tendermint/abci/types"

// Recorder records and replay ABCI events given a record file path.
type Recorder struct {
}

// New returns a new event recorder given a file path.
func New(path string) (*Recorder, error) {
	return nil, nil
}

// Record records events.
func (r *Recorder) Record(ev interface{}) error {
	return nil
}

// Replay reads events previously recorded into a record file and replay them in order.
func (r *Recorder) Replay(app types.Application) error {
	return nil
}
