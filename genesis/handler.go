package genesis

import (
	"time"

	"code.vegaprotocol.io/vega/logging"

	"github.com/tendermint/tendermint/abci/types"
)

type Handler struct {
	log *logging.Logger

	onGenesisTimeLoadedCB     []func(time.Time)
	onGenesisAppStateLoadedCB []func([]byte)
}

func New(log *logging.Logger) *Handler {
	return &Handler{
		log:                       log,
		onGenesisTimeLoadedCB:     []func(time.Time){},
		onGenesisAppStateLoadedCB: []func([]byte){},
	}
}

func (h *Handler) HandleGenesis(req types.RequestInitChain) {
	for _, f := range h.onGenesisTimeLoadedCB {
		f(req.Time)
	}
	for _, f := range h.onGenesisAppStateLoadedCB {
		f(req.AppStateBytes)
	}
}

func (h *Handler) OnGenesisTimeLoaded(f func(time.Time)) {
	h.onGenesisTimeLoadedCB = append(h.onGenesisTimeLoadedCB, f)
}

func (h *Handler) OnGenesisAppStateLoaded(f func([]byte)) {
	h.onGenesisAppStateLoadedCB = append(h.onGenesisAppStateLoadedCB, f)
}
