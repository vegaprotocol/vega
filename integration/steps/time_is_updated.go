package steps

import (
	"code.vegaprotocol.io/vega/integration/stubs"
)

func TimeIsUpdatedTo(timeService *stubs.TimeStub, newTime string) {
	t, err := Time(newTime)
	panicW("date", err)
	timeService.SetTime(t)
}
