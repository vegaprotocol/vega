package referral

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

type Engine struct {
	broker Broker

	// currentProgram is the program currently in used against which the reward
	// are computed.
	// It's `nil` is there is none.
	currentProgram *types.ReferralProgram

	// currentProgramHasEnded tells is the current program has reached its
	// end. It's flipped at the end of the epoch.
	currentProgramHasEnded bool

	// newProgram is the program born from the last enacted UpdateReferralProgram
	// proposal to apply at the start of the next epoch.
	// It's `nil` is there is none.
	newProgram *types.ReferralProgram
}

func (e *Engine) Update(newProgram *types.ReferralProgram) {
	e.newProgram = newProgram
}

func (e *Engine) HasProgramEnded() bool {
	return e.currentProgramHasEnded
}

func (e *Engine) OnEpoch(_ context.Context, ep types.Epoch) {
	switch ep.Action {
	case proto.EpochAction_EPOCH_ACTION_START:
		e.applyNewProgramIfAny()
	case proto.EpochAction_EPOCH_ACTION_END:
		e.endProgramIfReached(ep.EndTime)
	}
}

func (e *Engine) applyNewProgramIfAny() {
	if e.newProgram != nil {
		e.currentProgram = e.newProgram
		e.newProgram = nil
		e.currentProgramHasEnded = false
		// TODO: Send event to tell the new program started.
	}
}

func (e *Engine) endProgramIfReached(epochEnd time.Time) {
	// End the current program if it reached its end.
	if e.currentProgram != nil && e.currentProgram.EndOfProgramTimestamp.Before(epochEnd) {
		e.currentProgramHasEnded = true
		e.currentProgram = nil
		// TODO: Send event to tell the current program ended.
	}

	// Verifying if the new program is not already finished.
	if e.newProgram != nil && e.newProgram.EndOfProgramTimestamp.Before(epochEnd) {
		e.newProgram = nil
		// TODO: Send event to tell the new program didn't event start.
	}
}

func (e *Engine) OnEpochRestore(_ context.Context, _ types.Epoch) {}

func NewEngine(epochEngine EpochEngine, broker Broker) *Engine {
	engine := &Engine{
		broker: broker,

		// There is no program yet, so we mark it has ended so consumer of this
		// engine can know there is no reward computation to be done.
		currentProgramHasEnded: true,
	}

	epochEngine.NotifyOnEpoch(engine.OnEpoch, engine.OnEpochRestore)

	return engine
}
