package stream

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/grpc"

	api "code.vegaprotocol.io/vega/protos/vega/api/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

func connect(ctx context.Context,
	batchSize uint,
	party, market, serverAddr string, types []string,
) (conn *grpc.ClientConn, stream api.CoreService_ObserveEventBusClient, err error) {
	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()

	conn, err = grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return
	}

	stream, err = api.NewCoreServiceClient(conn).ObserveEventBus(ctx)
	if err != nil {
		return
	}

	busEventTypes, err := typesToBETypes(types)
	if err != nil {
		return
	}

	req := &api.ObserveEventBusRequest{
		MarketId:  market,
		PartyId:   party,
		BatchSize: int64(batchSize),
		Type:      busEventTypes,
	}

	if err = stream.Send(req); err != nil {
		err = fmt.Errorf("error when sending initial message in stream: %w", err)
		return
	}

	return conn, stream, nil
}

func typesToBETypes(types []string) ([]eventspb.BusEventType, error) {
	if len(types) == 0 {
		return []eventspb.BusEventType{
			eventspb.BusEventType_BUS_EVENT_TYPE_ALL,
		}, nil
	}

	dedup := map[string]struct{}{}
	beTypes := make([]eventspb.BusEventType, 0, len(types))

	for _, t := range types {
		// check if t is numeric:
		if n, err := strconv.ParseInt(t, 10, 32); err != nil && n > 0 {
			// it was numeric, and we found the name to match
			if ts, ok := eventspb.BusEventType_name[int32(n)]; ok {
				t = ts
			}
		}
		// deduplicate
		if _, ok := dedup[t]; ok {
			continue
		}

		dedup[t] = struct{}{}
		// now get the constant value and add it to the slice if possible
		if i, ok := eventspb.BusEventType_value[t]; ok {
			bet := eventspb.BusEventType(i)
			if bet == eventspb.BusEventType_BUS_EVENT_TYPE_ALL {
				return typesToBETypes(nil)
			}

			beTypes = append(beTypes, bet)
		} else {
			// We could not match the event string to the list defined in the proto file so stop now
			// so the user does not think everything is fine
			return nil, fmt.Errorf("no such event %s", t)
		}
	}

	if len(beTypes) == 0 {
		// default to ALL
		return typesToBETypes(nil)
	}

	return beTypes, nil
}

// ReadEvents reads all the events from the server.
func ReadEvents(
	ctx context.Context,
	cancel context.CancelFunc,
	wg *sync.WaitGroup,
	batchSize uint,
	party, market, serverAddr string,
	handleEvent func(event *eventspb.BusEvent),
	reconnect bool,
	types []string,
) error {
	if len(types) == 0 || (len(types) == 1 && len(types[0]) == 0) {
		types = nil
	}

	conn, stream, err := connect(ctx, batchSize, party, market, serverAddr, types)
	if err != nil {
		return fmt.Errorf("failed to connect to event stream: %w", err)
	}

	poll := &api.ObserveEventBusRequest{
		BatchSize: int64(batchSize),
	}

	wg.Add(1)

	go func() {
		defer func() {
			wg.Done()
			cancel()
			_ = conn.Close()
			_ = stream.CloseSend()
		}()

		for {
			for {
				o, err := stream.Recv()
				if err == io.EOF {
					log.Printf("stream closed by server err=%v", err)
					break
				}

				if err != nil {
					log.Printf("stream closed err=%v", err)
					break
				}

				for _, e := range o.Events {
					handleEvent(e)
				}

				if batchSize > 0 {
					if err := stream.SendMsg(poll); err != nil {
						log.Printf("failed to poll next event batch err=%v", err)
						return
					}
				}
			}

			if reconnect {
				// Keep waiting and retrying until we reconnect
				for {
					select {
					case <-ctx.Done():
						return
					default:
						time.Sleep(time.Second * 5)
						log.Printf("Attempting to reconnect to the node")
						conn, stream, err = connect(ctx, batchSize, party, market, serverAddr, types)
						if err == nil {
							break
						}
					}
					if err == nil {
						break
					}
				}
			} else {
				break
			}
		}
	}()

	return nil
}

// Run is the main function of `stream` package.
func Run(
	batchSize uint,
	party, market, serverAddr, logFormat string,
	reconnect bool,
	types []string,
) error {
	flag.Parse()

	if len(serverAddr) <= 0 {
		return fmt.Errorf("error: missing grpc server address")
	}

	handleEvent, err := NewLogEventToConsoleFn(logFormat)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := sync.WaitGroup{}

	if err := ReadEvents(ctx, cancel, &wg, batchSize, party, market, serverAddr, handleEvent, reconnect, types); err != nil {
		return fmt.Errorf("error when starting the stream: %v", err)
	}

	WaitSig(ctx, cancel)
	wg.Wait()

	return nil
}

// NewLogEventToConsoleFn returns a common logging function for use across tools that deal with events.
func NewLogEventToConsoleFn(logFormat string) (func(e *eventspb.BusEvent), error) {
	var printEvent func(string)
	switch logFormat {
	case "raw":
		printEvent = func(event string) { log.Printf("%v\n", event) }
	case "text":
		printEvent = func(event string) {
			log.Printf("%v;%v", time.Now().UTC().Format(time.RFC3339Nano), event)
		}
	case "json":
		printEvent = func(event string) {
			log.Printf("{\"time\":\"%v\",%v\n", time.Now().UTC().Format(time.RFC3339Nano), event[1:])
		}
	default:
		return nil, fmt.Errorf("error: unknown log-format: \"%v\". Allowed values: raw, text, json", logFormat)
	}

	m := jsonpb.Marshaler{}
	handleEvent := func(e *eventspb.BusEvent) {
		estr, err := m.MarshalToString(e)
		if err != nil {
			log.Printf("unable to marshal event err=%v", err)
		}
		printEvent(estr)
	}

	return handleEvent, nil
}

// WaitSig waits until Terminate or interrupt event is received.
func WaitSig(ctx context.Context, cancel func()) {
	gracefulStop := make(chan os.Signal, 1)

	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	select {
	case sig := <-gracefulStop:
		log.Printf("Caught signal name=%v", sig)
		log.Printf("closing client connections")
		cancel()
	case <-ctx.Done():
		return
	}
}
