package validators_test

import (
	"context"
	"encoding/hex"
	"testing"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/validators"
	"github.com/stretchr/testify/require"
	abcitypes "github.com/tendermint/tendermint/abci/types"
	types1 "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

var (
	address1, _ = hex.DecodeString("31D6EBD2A8E40524142613A241CA1D2056159EF4")
	address2, _ = hex.DecodeString("6DB7E2A705ABF86C6B4A4817E778669D45421166")
	address3, _ = hex.DecodeString("A5429AF24A820AFD9C3D21507C8642F27F5DD308")
	address4, _ = hex.DecodeString("AE5B9A8193AEFC405C159C930ED2BBF40A806785")
	address5, _ = hex.DecodeString("1C9B6E2708F8217F8D5BFC8D8734ED9A5BC19B21")
)

func TestValidatorPerformanceNoPerformance(t *testing.T) {
	vp := validators.NewValidatorPerformance(logging.NewTestLogger())
	require.Equal(t, num.DecimalFromFloat(1), vp.ValidatorPerformanceScore("some name"))
}

func TestElectedExpectationWithVotingPower(t *testing.T) {
	vp := validators.NewValidatorPerformance(logging.NewTestLogger())

	vd1 := []*tmtypes.Validator{
		{Address: address1, VotingPower: 3715, ProposerPriority: 5249},
		{Address: address2, VotingPower: 3351, ProposerPriority: 796},
		{Address: address3, VotingPower: 2793, ProposerPriority: -797},
		{Address: address4, VotingPower: 139, ProposerPriority: 1016},
		{Address: address5, VotingPower: 1, ProposerPriority: -6264},
	}
	req1 := abcitypes.RequestBeginBlock{Header: types1.Header{ProposerAddress: address1, Height: int64(1)}}
	vp.BeginBlock(context.Background(), req1, vd1)

	// expect all validators to have the same performance score, all by address1 for not being selected and address1 for being selected and proposing
	require.Equal(t, "1", vp.ValidatorPerformanceScore(hex.EncodeToString(address1)).String())
	require.Equal(t, "1", vp.ValidatorPerformanceScore(hex.EncodeToString(address2)).String())
	require.Equal(t, "1", vp.ValidatorPerformanceScore(hex.EncodeToString(address3)).String())
	require.Equal(t, "1", vp.ValidatorPerformanceScore(hex.EncodeToString(address4)).String())
	require.Equal(t, "1", vp.ValidatorPerformanceScore(hex.EncodeToString(address5)).String())

	vd2 := []*tmtypes.Validator{
		{Address: address1, VotingPower: 3715, ProposerPriority: 6433},
		{Address: address2, VotingPower: 3351, ProposerPriority: -1853},
		{Address: address3, VotingPower: 2793, ProposerPriority: 5347},
		{Address: address4, VotingPower: 139, ProposerPriority: -3701},
		{Address: address5, VotingPower: 1, ProposerPriority: -6226},
	}

	// expecting address1 to propose but got address3
	req2 := abcitypes.RequestBeginBlock{Header: types1.Header{ProposerAddress: address3, Height: int64(1)}}
	vp.BeginBlock(context.Background(), req2, vd2)

	vd3 := []*tmtypes.Validator{
		{Address: address1, VotingPower: 3715, ProposerPriority: -6433},
		{Address: address2, VotingPower: 3351, ProposerPriority: -1853},
		{Address: address3, VotingPower: 2793, ProposerPriority: -5347},
		{Address: address4, VotingPower: 139, ProposerPriority: 3701},
		{Address: address5, VotingPower: 1, ProposerPriority: -6226},
	}

	// expecting address4 to propose but got address5
	req3 := abcitypes.RequestBeginBlock{Header: types1.Header{ProposerAddress: address5, Height: int64(1)}}
	vp.BeginBlock(context.Background(), req3, vd3)

	require.Equal(t, "0.5", vp.ValidatorPerformanceScore(hex.EncodeToString(address1)).String())
	require.Equal(t, "1", vp.ValidatorPerformanceScore(hex.EncodeToString(address2)).String())
	require.Equal(t, "1", vp.ValidatorPerformanceScore(hex.EncodeToString(address3)).String())
	require.Equal(t, "0.05", vp.ValidatorPerformanceScore(hex.EncodeToString(address4)).String())
	require.Equal(t, "1", vp.ValidatorPerformanceScore(hex.EncodeToString(address5)).String())
}
