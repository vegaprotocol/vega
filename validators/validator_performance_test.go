package validators_test

import (
	"context"
	"encoding/hex"
	"strings"
	"testing"

	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/require"
	abcitypes "github.com/tendermint/tendermint/abci/types"
	types1 "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"

	"code.vegaprotocol.io/vega/validators"
)

func TestValidatorPerformanceNoPerformance(t *testing.T) {
	vp := validators.NewValidatorPerformance()
	require.Equal(t, num.DecimalFromFloat(1), vp.ValidatorPerformanceScore("some name"))
}

func TestProposedButNotElected(t *testing.T) {
	// this could happen on the first block

	ID := "1C9B6E2708F8217F8D5BFC8D8734ED9A5BC19B21"
	validator, _ := hex.DecodeString(ID)
	vp := validators.NewValidatorPerformance()
	req := abcitypes.RequestBeginBlock{Header: types1.Header{ProposerAddress: validator, Height: int64(1)}}
	vp.BeginBlock(context.Background(), req)

	// the validator has a proposal but we didn't see them being elected
	require.Equal(t, num.DecimalFromFloat(1), vp.ValidatorPerformanceScore(ID))
}

func TestElectedExpectation(t *testing.T) {
	vp := validators.NewValidatorPerformance()
	address1, _ := hex.DecodeString("1C9B6E2708F8217F8D5BFC8D8734ED9A5BC19B21")
	address2, _ := hex.DecodeString("31D6EBD2A8E40524142613A241CA1D2056159EF4")
	address3, _ := hex.DecodeString("6DB7E2A705ABF86C6B4A4817E778669D45421166")
	address4, _ := hex.DecodeString("A5429AF24A820AFD9C3D21507C8642F27F5DD308")
	address5, _ := hex.DecodeString("AE5B9A8193AEFC405C159C930ED2BBF40A806785")

	// the numbers are based on observing the consensus state on block x and the state of the validators on block x-1 on tendermint
	vd1 := []*tmtypes.Validator{
		{Address: address1, VotingPower: 10, ProposerPriority: 17},
		{Address: address2, VotingPower: 10, ProposerPriority: 17},
		{Address: address3, VotingPower: 10, ProposerPriority: -28},
		{Address: address4, VotingPower: 10, ProposerPriority: -28},
		{Address: address5, VotingPower: 10, ProposerPriority: 22},
	}
	vp.EndOfBlock(1, []abcitypes.ValidatorUpdate{}, vd1)
	// address5 elected once but never proposed
	println(vp.ValidatorPerformanceScore(hex.EncodeToString(address5)).String())
	require.Equal(t, "0", vp.ValidatorPerformanceScore(hex.EncodeToString(address5)).String())

	vd2 := []*tmtypes.Validator{
		{Address: address1, VotingPower: 10, ProposerPriority: 17},
		{Address: address2, VotingPower: 10, ProposerPriority: 17},
		{Address: address3, VotingPower: 10, ProposerPriority: -28},
		{Address: address4, VotingPower: 10, ProposerPriority: -28},
		{Address: address5, VotingPower: 10, ProposerPriority: 22},
	}

	// add another time for election of address 5
	vp.EndOfBlock(2, []abcitypes.ValidatorUpdate{}, vd2)

	// so now address5 has been elected twice and proposed never so still perf score expected to be zero
	require.Equal(t, "0", vp.ValidatorPerformanceScore(hex.EncodeToString(address5)).String())

	// now let address 5 propose so that the performance score would become 0.5
	req := abcitypes.RequestBeginBlock{Header: types1.Header{ProposerAddress: address5, Height: int64(3)}}
	vp.BeginBlock(context.Background(), req)
	require.Equal(t, "0.5", vp.ValidatorPerformanceScore(hex.EncodeToString(address5)).String())
	// verify that the lookup is case insensitive
	require.Equal(t, "0.5", vp.ValidatorPerformanceScore(strings.ToUpper(hex.EncodeToString(address5))).String())
}
