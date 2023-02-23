package broker

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"code.vegaprotocol.io/vega/core/broker"
	"code.vegaprotocol.io/vega/core/events"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	eventsv1 "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/stretchr/testify/assert"

	"code.vegaprotocol.io/vega/core/types"
)

func TestReceiveEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bufferFilesDir := t.TempDir()
	bufferFile, err := os.Create(filepath.Join(bufferFilesDir, "file1"))
	assert.NoError(t, err)
	defer bufferFile.Close()

	ctxBlock1 := vgcontext.WithTraceID(ctx, "1")
	a1 := events.NewAssetEvent(ctxBlock1, types.Asset{ID: "1"})

	ctxBlock2 := vgcontext.WithTraceID(ctx, "2")
	beginBlockEvent := events.NewBeginBlock(ctxBlock2, eventsv1.BeginBlock{
		Height:    10,
		Timestamp: 0,
	})

	a2 := events.NewAssetEvent(ctxBlock2, types.Asset{ID: "2"})
	a3 := events.NewAssetEvent(ctxBlock2, types.Asset{ID: "3"})

	broker.WriteToBufferFile(bufferFile, 6, a1)
	broker.WriteToBufferFile(bufferFile, 7, beginBlockEvent)
	broker.WriteToBufferFile(bufferFile, 8, a2)
	broker.WriteToBufferFile(bufferFile, 9, a3)
	bufferFile.Close()

	bufferFile2, err := os.Create(filepath.Join(bufferFilesDir, "file2"))
	assert.NoError(t, err)
	defer bufferFile2.Close()

	a4 := events.NewAssetEvent(ctxBlock2, types.Asset{ID: "4"})
	a5 := events.NewAssetEvent(ctxBlock2, types.Asset{ID: "5"})

	broker.WriteToBufferFile(bufferFile2, 10, a4)
	broker.WriteToBufferFile(bufferFile2, 11, a5)
	bufferFile2.Close()

	eventSource, err := NewBufferFilesEventSource(bufferFilesDir, 0, 1000, "")
	assert.NoError(t, err)

	err = eventSource.Listen()
	assert.NoError(t, err)

	evtCh, _ := eventSource.Receive(ctx)

	e1 := <-evtCh
	r1 := e1.(*events.BeginBlock)
	e2 := <-evtCh
	r2 := e2.(*events.Asset)
	e3 := <-evtCh
	r3 := e3.(*events.Asset)
	e4 := <-evtCh
	r4 := e4.(*events.Asset)
	e5 := <-evtCh
	r5 := e5.(*events.Asset)

	assert.Equal(t, uint64(10), r1.BeginBlock().Height)
	assert.Equal(t, "2", r2.Asset().Id)
	assert.Equal(t, "3", r3.Asset().Id)
	assert.Equal(t, "4", r4.Asset().Id)
	assert.Equal(t, "5", r5.Asset().Id)
}
