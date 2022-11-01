package api

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	protoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

func Test_recvEventRequest(t *testing.T) {
	type args struct {
		ctx     context.Context
		timeout time.Duration
		stream  eventBusServer
	}
	tests := []struct {
		name    string
		args    args
		want    *protoapi.ObserveEventBusRequest
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy path",
			args: args{
				ctx: context.Background(),
				stream: &mockEventBusServer{
					msg:   exampleObserveEventBusRequest(),
					sleep: 10 * time.Millisecond,
				},
				timeout: 100 * time.Millisecond,
			},
			want:    exampleObserveEventBusRequest(),
			wantErr: assert.NoError,
		}, {
			name: "error on stream",
			args: args{
				ctx: context.Background(),
				stream: &mockEventBusServer{
					err:   status.Error(codes.Internal, "error"),
					sleep: 10 * time.Millisecond,
				},
				timeout: 100 * time.Millisecond,
			},
			wantErr: assert.Error,
		}, {
			name: "timeout",
			args: args{
				ctx: context.Background(),
				stream: &mockEventBusServer{
					sleep: 100 * time.Millisecond,
				},
				timeout: 10 * time.Millisecond,
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := recvEventRequest(tt.args.ctx, tt.args.timeout, tt.args.stream)
			if !tt.wantErr(t, err, fmt.Sprintf("recvEventRequest(%v, %v, %v)", tt.args.ctx, tt.args.timeout, tt.args.stream)) {
				return
			}
			if err != nil {
				return
			}
			assert.Equalf(t, tt.want, got, "recvEventRequest(%v, %v, %v)", tt.args.ctx, tt.args.timeout, tt.args.stream)
		})
	}
}

func exampleObserveEventBusRequest() *protoapi.ObserveEventBusRequest {
	return &protoapi.ObserveEventBusRequest{
		Type:      []eventspb.BusEventType{eventspb.BusEventType_BUS_EVENT_TYPE_ALL},
		MarketId:  "123",
		PartyId:   "asdf",
		BatchSize: 100,
	}
}

type mockEventBusServer struct {
	msg   *protoapi.ObserveEventBusRequest
	err   error
	sleep time.Duration
}

func (m *mockEventBusServer) RecvMsg(i interface{}) error {
	time.Sleep(m.sleep)
	if m.err == nil && m.msg != nil {
		*i.(*protoapi.ObserveEventBusRequest) = *m.msg
	}
	return m.err
}

func (m *mockEventBusServer) Context() context.Context {
	return context.Background()
}

func (m *mockEventBusServer) Send([]*eventspb.BusEvent) error {
	return nil
}
