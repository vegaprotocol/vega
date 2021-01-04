package netparams_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/netparams"
	types "code.vegaprotocol.io/vega/proto/gen/golang"
	"github.com/stretchr/testify/assert"
)

type serviceTest struct {
	*netparams.Service
	ctx   context.Context
	cfunc context.CancelFunc
}

func getServiceTest(t *testing.T) *serviceTest {
	ctx, cfunc := context.WithCancel(context.Background())
	s := netparams.NewService(ctx)
	return &serviceTest{
		Service: s,
		ctx:     ctx,
		cfunc:   cfunc,
	}
}

func TestGetAllNetParams(t *testing.T) {
	svc := getServiceTest(t)
	evts := []*events.NetworkParameter{
		events.NewNetworkParameterEvent(svc.ctx, "key1", "value1"),
		events.NewNetworkParameterEvent(svc.ctx, "key2", "value2"),
		events.NewNetworkParameterEvent(svc.ctx, "key3", "value3"),
	}

	svc.Push(evts[0], evts[1], evts[2])

	var (
		hasError bool = true
		retries       = 50
	)

	for hasError && retries > 0 {
		retries -= 1
		all := svc.GetAll()
		// we expect 3 elements to be returned
		hasError = len(all) != 3
		time.Sleep(50 * time.Millisecond)
	}

	hasNP := func(nps []types.NetworkParameter, k, v string) bool {
		for _, np := range nps {
			if np.Key == k && np.Value == v {
				return true
			}
		}
		return false
	}

	all := svc.GetAll()
	assert.Len(t, all, 3)
	assert.True(t, hasNP(all, "key1", "value1"))
	assert.True(t, hasNP(all, "key2", "value2"))
	assert.True(t, hasNP(all, "key3", "value3"))
}
