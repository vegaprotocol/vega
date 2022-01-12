package validators

import (
	"context"
	"encoding/hex"

	"code.vegaprotocol.io/vega/types/num"

	abcitypes "github.com/tendermint/tendermint/abci/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

type validatorPerformance struct {
	proposed           uint64
	elected            uint64
	voted              uint64
	lastHeightVoted    int64
	lastHeightProposed int64
	lastHeightElected  int64
}

var decimalOne = num.DecimalFromFloat(1)

type ValidatorPerformance struct {
	performance map[string]*validatorPerformance
}

func NewValidatorPerformance() *ValidatorPerformance {
	return &ValidatorPerformance{
		performance: map[string]*validatorPerformance{},
	}
}

func newPerformance() *validatorPerformance {
	return &validatorPerformance{
		proposed:           0,
		elected:            0,
		voted:              0,
		lastHeightVoted:    -1,
		lastHeightProposed: -1,
		lastHeightElected:  -1,
	}
}

// ValidatorPerformanceScore returns the validator's performance score.
// in case the validator was never elected - they get a performance score of 1.
func (vp *ValidatorPerformance) ValidatorPerformanceScore(address string) num.Decimal {
	if _, ok := vp.performance[address]; !ok {
		return decimalOne
	}
	perf := vp.performance[address]
	if perf.elected == 0 {
		return decimalOne
	}
	return num.DecimalFromInt64(int64(perf.proposed)).Div(num.DecimalFromInt64(int64(perf.elected)))
}

// EndOfBlock is called at the end of a block to calculate the next block's expected proposer. This is done by
// applying the validator set changes on top of validator state from the ending block and getting the next proposer from the validator set.
func (vp *ValidatorPerformance) EndOfBlock(height int64, updates []abcitypes.ValidatorUpdate, vd []*tmtypes.Validator) {
	// given the state from the end of block we apply our changes
	vs := tmtypes.NewValidatorSet(vd)
	if len(updates) > 0 {
		changes, err := tmtypes.PB2TM.ValidatorUpdates(updates)
		if err != nil {
			return
		}
		vs.UpdateWithChangeSet(changes)
	}

	// get the proposer for the next round
	nextProposer := hex.EncodeToString(vs.Proposer.Address)

	if _, ok := vp.performance[nextProposer]; !ok {
		vp.performance[nextProposer] = newPerformance()
	}
	vp.performance[nextProposer].elected++
	vp.performance[nextProposer].lastHeightElected = height
}

// BeginBlock is called when a new block begins.
func (vp *ValidatorPerformance) BeginBlock(ctx context.Context, r abcitypes.RequestBeginBlock) {
	proposer := hex.EncodeToString(r.Header.ProposerAddress)

	if _, ok := vp.performance[proposer]; !ok {
		vp.performance[proposer] = newPerformance()
	}
	vp.performance[proposer].proposed++
	vp.performance[proposer].lastHeightProposed = r.Header.Height

	for _, vote := range r.LastCommitInfo.Votes {
		voter := hex.EncodeToString(vote.Validator.Address)
		if _, ok := vp.performance[voter]; !ok {
			vp.performance[proposer] = newPerformance()
		}
		vp.performance[voter].voted++
		vp.performance[proposer].lastHeightVoted = r.Header.Height
	}
}
