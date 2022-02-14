package vegatime

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/broker/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestTimeUpdateEventIsSentBeforeCallbacksAreInvoked(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mocks.NewMockBrokerI(ctrl)

	s := New(Config{}, m)

	callOrder := make([]int, 0)
	m.EXPECT().Send(gomock.Any()).DoAndReturn(func(any interface{}) { callOrder = append(callOrder, 1) })
	s.NotifyOnTick(func(ctx context.Context, t time.Time) { callOrder = append(callOrder, 2) })
	s.SetTimeNow(context.Background(), time.Now())

	assert.Equal(t, 1, callOrder[0])
	assert.Equal(t, 2, callOrder[1])
}
