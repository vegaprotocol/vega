package orders

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/internal/vegatime"
	"code.vegaprotocol.io/vega/internal/vegatime/mocks"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestOrderService_validateOrderExpirationTS(t *testing.T) {
	orderService := orderService{
		timeService: vegatime.NewTimeService(nil),
	}
	validdt := time.Date(2019, time.June, 1, 0, 0, 0, 0, time.UTC)

	t.Run("datetime is not RFC3339", func(t *testing.T) {
		invaliddt := "not a valid time at all"
		_, err := orderService.validateOrderExpirationTS(invaliddt)
		assert.NotNil(t, err)
		assert.Equal(t, ErrInvalidExpirationDTFmt, err)
	})

	t.Run("unable to get vegatime now", func(t *testing.T) {
		expctErr := errors.New("time error")
		mock := &mocks.Service{}

		mock.On("GetTimeNow").
			Return(vegatime.Stamp(0), time.Time{}, expctErr)
		orderService.timeService = mock
		_, err := orderService.validateOrderExpirationTS(validdt.Format(time.RFC3339))
		assert.NotNil(t, err)
		assert.Equal(t, expctErr, err)
	})

	t.Run("datetime is invalid (in the past)", func(t *testing.T) {
		mock := &mocks.Service{}
		mock.On("GetTimeNow").
			Return(vegatime.Stamp(0), validdt.Add(24*time.Second), nil)
		orderService.timeService = mock
		_, err := orderService.validateOrderExpirationTS(validdt.Format(time.RFC3339))
		assert.NotNil(t, err)
		assert.Equal(t, ErrInvalidExpirationDT, err)
	})

	t.Run("datatime is valid (in the future)", func(t *testing.T) {
		mock := &mocks.Service{}
		mock.On("GetTimeNow").
			Return(vegatime.Stamp(0), validdt.Add(-24*time.Second), nil)
		orderService.timeService = mock
		ts, err := orderService.validateOrderExpirationTS(validdt.Format(time.RFC3339))
		assert.Nil(t, err)
		assert.Equal(t, validdt, ts)
	})
}
