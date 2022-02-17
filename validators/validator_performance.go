package validators

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"strings"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types/num"

	abcitypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/sr25519"
	tmtypes "github.com/tendermint/tendermint/types"
)

type performanceStats struct {
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

type validatorPerformance struct {
	performance map[string]*performanceStats
	log         *logging.Logger
}

func NewValidatorPerformance(log *logging.Logger) *validatorPerformance {
	return &validatorPerformance{
		performance: map[string]*performanceStats{},
		log:         log,
	}
}

func newPerformance() *performanceStats {
	return &performanceStats{
		proposed:           0,
		elected:            0,
		voted:              0,
		lastHeightVoted:    -1,
		lastHeightProposed: -1,
		lastHeightElected:  -1,
	}
}

func tmPubKeyToAddress(tmPubKey string) string {
	if len(tmPubKey) == 0 {
		return ""
	}
	pubkey, err := base64.StdEncoding.DecodeString(tmPubKey)
	if err != nil {
		return ""
	}
	pke := sr25519.PubKey(pubkey)
	address := hex.EncodeToString(pke.Address().Bytes())
	return strings.ToLower(address)
}

// ValidatorPerformanceScore returns the validator's performance score.
// in case the validator was never elected - they get a performance score of 1.
func (vp *validatorPerformance) ValidatorPerformanceScore(tmPubKey string) num.Decimal {
	// convert from tendermint public key key to address
	address := tmPubKeyToAddress(tmPubKey)
	if _, ok := vp.performance[address]; !ok {
		return decimalOne
	}
	perf := vp.performance[address]
	if perf.elected == 0 {
		return decimalOne
	}
	score := num.MaxD(minPerfScore, num.DecimalFromInt64(int64(perf.proposed)).Div(num.DecimalFromInt64(int64(perf.elected))))
	vp.log.Info("loooking up performance for", logging.String("address", address), logging.String("perf-score", score.String()))
	return score
}

// BeginBlock is called when a new block begins. it calculates who should have been the proposer and updates the counters with the expected and actual proposers and voters.
func (vp *validatorPerformance) BeginBlock(ctx context.Context, r abcitypes.RequestBeginBlock, vd []*tmtypes.Validator) {
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

	vp.log.Info("validatorPerformance", logging.String("expected-proposer", expectedProposer), logging.String("actual-proposer", proposer))

	for _, vote := range r.LastCommitInfo.Votes {
		voter := hex.EncodeToString(vote.Validator.Address)
		if _, ok := vp.performance[voter]; !ok {
			vp.performance[voter] = newPerformance()
		}
		vp.performance[voter].voted++
		vp.performance[voter].lastHeightVoted = r.Header.Height
	}
}
