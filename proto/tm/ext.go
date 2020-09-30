package tm

import (
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/tendermint/tendermint/abci/types"
)

func fromTMValidatorUpdates(ups []types.ValidatorUpdate) []*ValidatorUpdate {
	out := make([]*ValidatorUpdate, 0, len(ups))
	for _, up := range ups {
		out = append(out, &ValidatorUpdate{
			PubKey: &PublicKey{
				Sum: &PublicKey_Ed25519{
					Ed25519: up.PubKey.GetData(),
				},
			},
			Power: up.Power,
		})
	}
	return out
}

func intoTMValidatorUpdates(ups []*ValidatorUpdate) []types.ValidatorUpdate {
	out := make([]types.ValidatorUpdate, 0, len(ups))
	for _, up := range ups {
		out = append(out, types.ValidatorUpdate{
			PubKey: types.PubKey{
				Data: up.PubKey.GetEd25519(),
			},
			Power: up.Power,
		})
	}
	return out

}

func (RequestInitChain) FromTM(t *types.RequestInitChain) *RequestInitChain {
	return &RequestInitChain{
		Time:          t.Time.UnixNano(),
		ChainID:       t.ChainId,
		Validators:    fromTMValidatorUpdates(t.Validators),
		AppStateBytes: t.AppStateBytes,
	}
}

func (r *RequestInitChain) IntoTM() types.RequestInitChain {
	return types.RequestInitChain{
		Time:          vegatime.UnixNano(r.Time),
		ChainId:       r.ChainID,
		Validators:    intoTMValidatorUpdates(r.Validators),
		AppStateBytes: r.AppStateBytes,
	}
}

func fromTMHeader(t types.Header) *Header {
	return &Header{
		ChainId: t.ChainID,
		Height:  t.Height,
		Time:    t.Time.UnixNano(),
	}
}

func intoTMHeader(t *Header) types.Header {
	return types.Header{
		ChainID: t.ChainId,
		Height:  t.Height,
		Time:    vegatime.UnixNano(t.Time),
	}
}

func (RequestBeginBlock) FromTM(t *types.RequestBeginBlock) *RequestBeginBlock {
	return &RequestBeginBlock{
		Hash:   t.Hash,
		Header: fromTMHeader(t.Header),
	}
}

func (r *RequestBeginBlock) IntoTM() types.RequestBeginBlock {
	return types.RequestBeginBlock{
		Hash:   r.Hash,
		Header: intoTMHeader(r.Header),
	}
}

func (RequestDeliverTx) FromTM(t *types.RequestDeliverTx) *RequestDeliverTx {
	return &RequestDeliverTx{
		Tx: t.Tx,
	}
}

func (r *RequestDeliverTx) IntoTM() types.RequestDeliverTx {
	return types.RequestDeliverTx{
		Tx: r.Tx,
	}
}
