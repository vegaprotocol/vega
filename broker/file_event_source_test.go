package broker_test

import (
	"context"
	"encoding/binary"
	"os"
	"testing"

	"code.vegaprotocol.io/data-node/broker"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestReceiveEvents(t *testing.T) {
	path := t.TempDir() + "/test.evt"

	a1 := events.NewAssetEvent(context.Background(), types.Asset{ID: "1"})
	a2 := events.NewAssetEvent(context.Background(), types.Asset{ID: "2"})
	a3 := events.NewAssetEvent(context.Background(), types.Asset{ID: "3"})

	evts := []*eventspb.BusEvent{
		a1.StreamMessage(), a2.StreamMessage(),
		a3.StreamMessage(),
	}

	err := writeEventsToFile(evts, path)
	if err != nil {
		t.Fatalf("failed to write events to %s: %s", path, err)
	}

	source, err := broker.NewFileEventSource(path, 0, 0)
	if err != nil {
		t.Errorf("failed to create file event source:%s", err)
	}

	evtCh, _ := source.Receive(context.Background())

	e1 := <-evtCh
	r1 := e1.(*events.Asset)
	e2 := <-evtCh
	r2 := e2.(*events.Asset)
	e3 := <-evtCh
	r3 := e3.(*events.Asset)

	assert.Equal(t, "1", r1.Asset().Id)
	assert.Equal(t, "2", r2.Asset().Id)
	assert.Equal(t, "3", r3.Asset().Id)
}

func writeEventsToFile(events []*eventspb.BusEvent, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	sizeBytes := make([]byte, 4)

	for _, e := range events {
		size := uint32(proto.Size(e))
		protoBytes, err := proto.Marshal(e)
		if err != nil {
			panic("failed to marshal bus event:" + e.String())
		}

		binary.BigEndian.PutUint32(sizeBytes, size)
		allBytes := append(sizeBytes, protoBytes...)
		file.Write(allBytes)
	}

	return nil
}
