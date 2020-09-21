package recorder

import "github.com/tendermint/tendermint/abci/types"

// App is an ABCI types that wraps a real implementation and records all its ABCI operation.
type App struct {
	types.Application
	rec *Recorder
}

// NewApp returns a new Recorder using path as the target for the record file.
func NewAppWithPath(app types.Application, path string) (*App, error) {
	rec, err := New(path)
	if err != nil {
		return nil, err
	}

	return NewApp(app, rec), nil

}

func NewApp(app types.Application, rec *Recorder) *App {
	return &App{
		Application: app,
		rec:         rec,
	}
}

func (r *App) InitChain(req types.RequestInitChain) types.ResponseInitChain {
	// record(req)
	resp := r.Application.InitChain(req)
	// record(resp)
	return resp
}

func (r *App) BeginBlock(req types.RequestBeginBlock) types.ResponseBeginBlock {
	// record(req)
	resp := r.Application.BeginBlock(req)
	// record(resp)
	return resp
}

func (r *App) DeliverTx(req types.RequestDeliverTx) types.ResponseDeliverTx {
	// record(req)
	resp := r.Application.DeliverTx(req)
	// record(resp)
	return resp
}

func (r *App) EndBlock(req types.RequestEndBlock) types.ResponseEndBlock {
	// record(req)
	resp := r.Application.EndBlock(req)
	// record(resp)
	return resp
}

func (r *App) Commit() types.ResponseCommit {
	// record(req)
	resp := r.Application.Commit()
	// record(resp)
	return resp
}
