package teams

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/teams EpochEngine,Broker,TimeService

// EpochEngine is used to know when to apply the team switches.
type EpochEngine interface {
	NotifyOnEpoch(func(context.Context, types.Epoch), func(context.Context, types.Epoch))
}

// Broker is used to notify administrative actions on teams and members.
type Broker interface {
	Send(event events.Event)
}

// TimeService is used to retrieve the current chain time to time stamp the
// team and there membership.
type TimeService interface {
	GetTimeNow() time.Time
}
