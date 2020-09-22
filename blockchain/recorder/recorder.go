package recorder

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync/atomic"

	tmreplay "code.vegaprotocol.io/vega/proto/tm"

	"github.com/golang/protobuf/proto"
	"github.com/spf13/afero"
	"github.com/tendermint/tendermint/abci/types"
)

var (
	ErrRecorderStopped       = errors.New("recorder stopped")
	ErrTMMessageNotSupported = errors.New("tm message not supported")
	ErrUnsupportedAction     = errors.New("unsupported action")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/abci_app_mock.go -package mocks code.vegaprotocol.io/vega/blockchain/recorder ABCIApp
type ABCIApp interface {
	InitChain(types.RequestInitChain) types.ResponseInitChain
	DeliverTx(types.RequestDeliverTx) types.ResponseDeliverTx
	BeginBlock(types.RequestBeginBlock) types.ResponseBeginBlock
}

// Recorder records and replay ABCI events given a record file path.
type Recorder struct {
	size    [4]byte
	f       afero.File
	running int32 // any value different to 0 means not running
}

func NewRecord(path string, fs afero.Fs) (*Recorder, error) {
	f, err := fs.Create(path)
	if err != nil {
		return nil, err
	}
	return &Recorder{
		f: f,
	}, nil
}

func NewReplay(path string, fs afero.Fs) (*Recorder, error) {
	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	return &Recorder{
		f: f,
	}, nil
}

func (r *Recorder) isRunning() bool {
	return atomic.LoadInt32(&r.running) == 0
}

func (r *Recorder) Stop() error {
	atomic.StoreInt32(&r.running, 1)
	return r.f.Close()
}

// Record records events.
func (r *Recorder) Record(ev interface{}) error {
	if !r.isRunning() {
		return ErrRecorderStopped
	}

	tmEvent := tmreplay.TmEvent{}
	switch ev := ev.(type) {
	case *types.RequestInitChain:
		tmEvent.Action = &tmreplay.TmEvent_ReqInitChain{
			ReqInitChain: tmreplay.RequestInitChain{}.FromTM(ev),
		}
	case *types.ResponseInitChain:
		// later
	case *types.RequestBeginBlock:
		tmEvent.Action = &tmreplay.TmEvent_ReqBeginBlock{
			ReqBeginBlock: tmreplay.RequestBeginBlock{}.FromTM(ev),
		}
	case *types.ResponseBeginBlock:
		// later
	case *types.RequestDeliverTx:
		tmEvent.Action = &tmreplay.TmEvent_ReqDeliverTx{
			ReqDeliverTx: tmreplay.RequestDeliverTx{}.FromTM(ev),
		}
	case *types.ResponseDeliverTx:
		// later
	default:
		return ErrTMMessageNotSupported
	}

	buf, err := proto.Marshal(&tmEvent)
	if err != nil {
		return err
	}

	binary.BigEndian.PutUint32(r.size[0:], uint32(len(buf)))

	_, err = r.f.Write(append(r.size[0:], buf...))
	return err
}

func (r *Recorder) read() ([]byte, error) {
	if _, err := r.f.Read(r.size[0:]); err != nil {
		return nil, fmt.Errorf("unable to read msg size: %w", err)
	}

	bufsize := binary.BigEndian.Uint32(r.size[0:])
	buf := make([]byte, bufsize)
	if _, err := r.f.Read(buf); err != nil {
		// in this case as we reading from a file
		// if we cannot get all the size we asked for, an error happend
		return nil, fmt.Errorf("unable to read msg: %w", err)
	}

	return buf, nil
}

// Replay reads events previously recorded into a record file and replay them in order.
func (r *Recorder) Replay(app ABCIApp) error {
	for {
		if !r.isRunning() {
			return ErrRecorderStopped
		}

		buf, err := r.read()
		if err != nil {
			// mask the error if we reached the end of file
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		// unmarshal the buffer
		tmEvent := tmreplay.TmEvent{}
		err = proto.Unmarshal(buf, &tmEvent)
		if err != nil {
			return fmt.Errorf("unable to unmarshal message: %w", err)
		}

		switch ev := tmEvent.Action.(type) {
		case *tmreplay.TmEvent_ReqInitChain:
			app.InitChain(ev.ReqInitChain.IntoTM())
		case *tmreplay.TmEvent_ResInitChain:
			// nothing to do for now
		case *tmreplay.TmEvent_ReqDeliverTx:
			app.DeliverTx(ev.ReqDeliverTx.IntoTM())
		case *tmreplay.TmEvent_ResDeliverTx:
			// nothing to do for now
		case *tmreplay.TmEvent_ReqBeginBlock:
			app.BeginBlock(ev.ReqBeginBlock.IntoTM())
		case *tmreplay.TmEvent_ResBeginBlock:
			// nopthing to do for now
		default:
			return ErrUnsupportedAction
		}
	}
}
