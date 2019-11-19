package core

import (
	"time"

	"code.vegaprotocol.io/vega/vegatime"
)

type TimeControl struct {
	vegaTime *vegatime.Svc
}

func NewTimeControl(vegaTime *vegatime.Svc) *TimeControl {
	return &TimeControl{vegaTime}
}

// SetTime sets protocol time to the provided value
func (t *TimeControl) SetTime(time time.Time) {
	t.vegaTime.SetTimeNow(time)
}

// AdvanceTime advances protocol time by a specified duration
func (t *TimeControl) AdvanceTime(duration time.Duration) error {
	currentTime, err := t.vegaTime.GetTimeNow()
	if err != nil {
		return err
	}
	advancedTime := currentTime.Add(duration)
	t.SetTime(advancedTime)
	return nil
}
