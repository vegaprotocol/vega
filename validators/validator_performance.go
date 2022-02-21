package validators

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"strings"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/tendermint/tendermint/crypto/sr25519"
)

var (
	decimalOne   = num.DecimalFromFloat(1)
	minPerfScore = num.DecimalFromFloat(0.05)
)

type validatorPerformance struct {
	proposals map[string]int64
	total     int64
	log       *logging.Logger
}

func NewValidatorPerformance(log *logging.Logger) *validatorPerformance {
	return &validatorPerformance{
		proposals: map[string]int64{},
		total:     0,
		log:       log,
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

// ValidatorPerformanceScore returns the validator's performance score calculated as the numer of proposals out of the total number of proposals
// normalised by their power out of the total power.
func (vp *validatorPerformance) ValidatorPerformanceScore(tmPubKey string, power, totalPower int64) num.Decimal {
	if vp.total == 0 || totalPower == 0 {
		return minPerfScore
	}

	// convert from tendermint public key key to address
	address := tmPubKeyToAddress(tmPubKey)
	noProposals := int64(0)
	if _, ok := vp.proposals[address]; ok {
		noProposals = vp.proposals[address]
	}

	actual := num.DecimalFromInt64(noProposals).Div(num.DecimalFromInt64(vp.total))
	expected := num.DecimalFromInt64(power).Div(num.DecimalFromInt64(totalPower))

	score := num.MaxD(minPerfScore, num.MinD(decimalOne, actual.Div(expected)))
	vp.log.Info("loooking up performance for", logging.String("address", address), logging.String("perf-score", score.String()))
	return score
}

// BeginBlock is called when a new block begins. it calculates who should have been the proposer and updates the counters with the expected and actual proposers and voters.
func (vp *validatorPerformance) BeginBlock(ctx context.Context, proposer string) {
	if _, ok := vp.proposals[proposer]; !ok {
		vp.proposals[proposer] = 0
	}
	vp.proposals[proposer]++
	vp.total++
}

func (vp *validatorPerformance) Reset() {
	vp.total = 0
	vp.proposals = map[string]int64{}
}
