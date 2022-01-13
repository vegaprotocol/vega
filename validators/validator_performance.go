package validators

import (
	"context"
	"encoding/hex"
	"strings"

	"code.vegaprotocol.io/vega/logging"
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

var (
	decimalOne   = num.DecimalFromFloat(1)
	minPerfScore = num.DecimalFromFloat(0.05)
)

type ValidatorPerformance struct {
	performance map[string]*validatorPerformance
	log         *logging.Logger
}

func NewValidatorPerformance(log *logging.Logger) *ValidatorPerformance {
	return &ValidatorPerformance{
		performance: map[string]*validatorPerformance{},
		log:         log,
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
	// the addresses of validators in the map are the result of encoding hex so are lower case apparently
	// so to make sure we find them given the key may be upper case first convert to lower case
	if _, ok := vp.performance[strings.ToLower(address)]; !ok {
		return decimalOne
	}
	perf := vp.performance[strings.ToLower(address)]
	if perf.elected == 0 {
		return decimalOne
	}
	return num.MaxD(minPerfScore, num.DecimalFromInt64(int64(perf.proposed)).Div(num.DecimalFromInt64(int64(perf.elected))))
}

// BeginBlock is called when a new block begins. it calculates who should have been the proposer and updates the counters with the expected and actual proposers and voters.
func (vp *ValidatorPerformance) BeginBlock(ctx context.Context, r abcitypes.RequestBeginBlock, vd []*tmtypes.Validator) {
	if len(vd) == 0 {
		return
	}
	vs := &tmtypes.ValidatorSet{Validators: vd}
	expectedProposer := hex.EncodeToString(vs.GetProposer().Address)
	if _, ok := vp.performance[expectedProposer]; !ok {
		vp.performance[expectedProposer] = newPerformance()
	}
	vp.performance[expectedProposer].elected++
	vp.performance[expectedProposer].lastHeightElected = r.Header.Height

	proposer := hex.EncodeToString(r.Header.ProposerAddress)
	if _, ok := vp.performance[proposer]; !ok {
		vp.performance[proposer] = newPerformance()
	}
	vp.performance[proposer].proposed++
	vp.performance[proposer].lastHeightProposed = r.Header.Height

	if vp.log.GetLevel() <= logging.DebugLevel {
		vp.log.Debug("ValidatorPerformance", logging.String("expected-proposer", expectedProposer), logging.String("actual-proposer", proposer))
	}

	for _, vote := range r.LastCommitInfo.Votes {
		voter := hex.EncodeToString(vote.Validator.Address)
		if _, ok := vp.performance[voter]; !ok {
			vp.performance[voter] = newPerformance()
		}
		vp.performance[voter].voted++
		vp.performance[voter].lastHeightVoted = r.Header.Height
	}
}
