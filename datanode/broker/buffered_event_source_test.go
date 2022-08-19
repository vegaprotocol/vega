package broker

import (
	"context"
	"fmt"
	"io/ioutil"
	"sort"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	"github.com/stretchr/testify/assert"
)

func Test_FileBufferedEventSource_BufferingDisabledWhenEventsPerFileIsZero(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	path := t.TempDir()

	eventSource := &testEventSource{
		eventsCh: make(chan events.Event, 1000),
		errCh:    make(chan error),
	}

	fb, err := NewBufferedEventSource(logging.NewTestLogger(), BufferedEventSourceConfig{
		EventsPerFile:         0,
		SendChannelBufferSize: 1000,
		MaxBufferedEvents:     10000,
	}, eventSource, path)

	assert.NoError(t, err)

	evtCh, _ := fb.Receive(ctx)

	numberOfEventsToSend := 100
	for i := 0; i < numberOfEventsToSend; i++ {
		a := events.NewAssetEvent(context.Background(), types.Asset{ID: fmt.Sprintf("%03d", i)})
		eventSource.eventsCh <- a
	}

	// This check consumes all events, and after each event buffer file is read it checks that it is removed
	for i := 0; i < numberOfEventsToSend; i++ {
		files, _ := ioutil.ReadDir(path)
		assert.Equal(t, 0, len(files))
		e := <-evtCh
		r := e.(*events.Asset)
		assert.Equal(t, fmt.Sprintf("%03d", i), r.Asset().Id)
	}
}

func Test_FileBufferedEventSource_ErrorSentOnPathError(t *testing.T) {
	eventSource := &testEventSource{
		eventsCh: make(chan events.Event),
		errCh:    make(chan error),
	}

	fb, err := NewBufferedEventSource(logging.NewTestLogger(), BufferedEventSourceConfig{
		EventsPerFile:         10,
		SendChannelBufferSize: 0,
		MaxBufferedEvents:     10000,
	}, eventSource, "thepaththatdoesntexist")

	assert.NoError(t, err)

	_, errCh := fb.Receive(context.Background())

	eventSource.errCh <- fmt.Errorf("test error")

	assert.NotNil(t, <-errCh)
}

func Test_FileBufferedEventSource_ErrorsArePassedThrough(t *testing.T) {
	path := t.TempDir()

	eventSource := &testEventSource{
		eventsCh: make(chan events.Event),
		errCh:    make(chan error),
	}

	fb, err := NewBufferedEventSource(logging.NewTestLogger(), BufferedEventSourceConfig{
		EventsPerFile:         10,
		SendChannelBufferSize: 0,
		MaxBufferedEvents:     10000,
	}, eventSource, path)

	assert.NoError(t, err)

	_, errCh := fb.Receive(context.Background())

	eventSource.errCh <- fmt.Errorf("test error")

	assert.NotNil(t, <-errCh)
}

func Test_FileBufferedEventSource_EventsAreBufferedAndPassedThrough(t *testing.T) {
	path := t.TempDir()

	eventSource := &testEventSource{
		eventsCh: make(chan events.Event),
		errCh:    make(chan error),
	}

	fb, err := NewBufferedEventSource(logging.NewTestLogger(), BufferedEventSourceConfig{
		EventsPerFile:         10,
		SendChannelBufferSize: 0,
		MaxBufferedEvents:     10000,
	}, eventSource, path)

	assert.NoError(t, err)

	evtCh, _ := fb.Receive(context.Background())

	a1 := events.NewAssetEvent(context.Background(), types.Asset{ID: "1"})
	a2 := events.NewAssetEvent(context.Background(), types.Asset{ID: "2"})
	a3 := events.NewAssetEvent(context.Background(), types.Asset{ID: "3"})

	eventSource.eventsCh <- a1
	eventSource.eventsCh <- a2
	eventSource.eventsCh <- a3

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

func Test_FileBufferedEventSource_RollsBufferFiles(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	path := t.TempDir()

	eventSource := &testEventSource{
		eventsCh: make(chan events.Event),
		errCh:    make(chan error),
	}

	eventsPerFile := 10
	fb, err := NewBufferedEventSource(logging.NewTestLogger(), BufferedEventSourceConfig{
		EventsPerFile:         eventsPerFile,
		SendChannelBufferSize: 0,
		MaxBufferedEvents:     10000,
	}, eventSource, path)

	assert.NoError(t, err)

	evtCh, _ := fb.Receive(ctx)

	numberOfEventsToSend := 100
	for i := 0; i < numberOfEventsToSend; i++ {
		a := events.NewAssetEvent(context.Background(), types.Asset{ID: fmt.Sprintf("%03d", i)})
		eventSource.eventsCh <- a
	}

	// This check consumes all events, and after each event buffer file is read it checks that it is removed
	for i := 0; i < numberOfEventsToSend; i++ {
		if i%eventsPerFile == 0 {
			files, _ := ioutil.ReadDir(path)
			expectedNumFiles := (numberOfEventsToSend - i) / eventsPerFile

			// As it interacts with disk, there is a bit of asynchronicity, this loop is to ensure that the directory
			// has chance to update. It will timeout if this test fails
			for expectedNumFiles != len(files) {
				files, _ = ioutil.ReadDir(path)
				time.Sleep(5 * time.Millisecond)
			}

			sort.Slice(files, func(i int, j int) bool {
				return files[i].ModTime().Before(files[j].ModTime())
			})
			for j, f := range files {
				expectedFilename := fmt.Sprintf("datanode-buffer-%d-%d", (j+i/eventsPerFile)*eventsPerFile+1, (j+1+i/eventsPerFile)*eventsPerFile)
				assert.Equal(t, expectedFilename, f.Name())
			}
		}

		e := <-evtCh
		r := e.(*events.Asset)
		assert.Equal(t, fmt.Sprintf("%03d", i), r.Asset().Id)
	}
}

type testEventSource struct {
	eventsCh chan events.Event
	errCh    chan error
}

func (t *testEventSource) Listen() error {
	return nil
}

func (t *testEventSource) Receive(ctx context.Context) (<-chan events.Event, <-chan error) {
	return t.eventsCh, t.errCh
}
