package recorder_test

import (
	"testing"

	"code.vegaprotocol.io/vega/blockchain/recorder"
	"code.vegaprotocol.io/vega/blockchain/recorder/mocks"

	"github.com/golang/mock/gomock"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/abci/types"
)

const (
	filePath = "recording.vega"
)

func TestRecorder(t *testing.T) {
	t.Run("record and replay - success", testRecordAndReplay)
	t.Run("fail replay - no file", testFailReplaying)
	t.Run("fail record - no file", testFailRecording)
}

func testRecordAndReplay(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctrl := gomock.NewController(t)
	abciapp := mocks.NewMockABCIApp(ctrl)
	rec, err := recorder.NewRecord(filePath, fs)
	assert.NoError(t, err)
	assert.NotNil(t, rec)

	initChainReq := types.RequestInitChain{
		ChainId: "mychain",
	}
	beginBlockReq := types.RequestBeginBlock{
		Hash: []byte("stronghashing"),
	}
	deliverTxReq := types.RequestDeliverTx{
		Tx: []byte("goodTX"),
	}
	// now we going to produce a bunch of request
	err = rec.Record(&initChainReq)
	assert.NoError(t, err)
	err = rec.Record(&beginBlockReq)
	assert.NoError(t, err)
	err = rec.Record(&deliverTxReq)
	assert.NoError(t, err)

	// ensure the file is not empty
	fi, err := fs.Stat(filePath)
	assert.NoError(t, err)
	assert.NotEqual(t, fi.Size(), 0)

	assert.NoError(t, rec.Stop())

	// now we create a Replay and we'll expect to replay all these events
	rep, err := recorder.NewReplay(filePath, fs)
	assert.NoError(t, err)
	assert.NotNil(t, rep)

	abciapp.EXPECT().InitChain(gomock.Any()).Times(1).Do(
		func(req types.RequestInitChain) (res types.ResponseInitChain) {
			assert.Equal(t, req.ChainId, initChainReq.ChainId)
			return
		},
	)
	abciapp.EXPECT().BeginBlock(gomock.Any()).Times(1).Do(
		func(req types.RequestBeginBlock) (res types.ResponseBeginBlock) {
			assert.Equal(t, string(req.Hash), string(beginBlockReq.Hash))
			return
		},
	)
	abciapp.EXPECT().DeliverTx(gomock.Any()).Times(1).Do(
		func(req types.RequestDeliverTx) (res types.ResponseDeliverTx) {
			assert.Equal(t, string(req.Tx), string(deliverTxReq.Tx))
			return
		},
	)

	err = rep.Replay(abciapp)
	assert.NoError(t, err)

	assert.NoError(t, rec.Stop())
}

func testFailReplaying(t *testing.T) {
	fs := afero.NewMemMapFs()
	rec, err := recorder.NewReplay(filePath, fs)
	assert.EqualError(t, err, "open recording.vega: file does not exist")
	assert.Nil(t, rec)
}

func testFailRecording(t *testing.T) {
	// start with a read only fs so we cannot open a file.
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs())
	rec, err := recorder.NewRecord(filePath, fs)
	assert.EqualError(t, err, "operation not permitted")
	assert.Nil(t, rec)
}
