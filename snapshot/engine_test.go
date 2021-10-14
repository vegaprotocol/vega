package snapshot_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/snapshot"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type tstEngine struct {
	*snapshot.Engine
	ctrl *gomock.Controller
}

func getTestEngine(t *testing.T) *tstEngine {
	ctrl := gomock.NewController(t)
	eng, err := snapshot.New(context.Background(), nil, snapshot.NewTestConfig(), logging.NewTestLogger(), nil)
	require.NoError(t, err)
	return &tstEngine{
		Engine: eng,
		ctrl:   ctrl,
	}
}
