// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package governance_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/data-node/governance"
	"code.vegaprotocol.io/data-node/governance/mocks"
	"code.vegaprotocol.io/data-node/logging"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testSvc struct {
	*governance.Svc
	ctrl  *gomock.Controller
	ctx   context.Context
	cfunc context.CancelFunc

	bus   *mocks.MockEventBus
	gov   *mocks.MockGovernanceDataSub
	votes *mocks.MockVoteSub
}

func newTestService(t *testing.T) *testSvc {
	ctrl := gomock.NewController(t)
	bus := mocks.NewMockEventBus(ctrl)
	gov := mocks.NewMockGovernanceDataSub(ctrl)
	votes := mocks.NewMockVoteSub(ctrl)

	ctx, cfunc := context.WithCancel(context.Background())

	result := &testSvc{
		ctrl:  ctrl,
		ctx:   ctx,
		cfunc: cfunc,
		bus:   bus,
		gov:   gov,
		votes: votes,
	}
	result.Svc = governance.NewService(logging.NewTestLogger(), governance.NewDefaultConfig(), bus, gov, votes)
	assert.NotNil(t, result.Svc)
	return result
}

func TestGovernanceService(t *testing.T) {
	svc := newTestService(t)
	defer svc.ctrl.Finish()

	cfg := svc.Config
	cfg.Level.Level = logging.DebugLevel
	svc.ReloadConf(cfg)
	assert.Equal(t, svc.Config.Level.Level, logging.DebugLevel)

	cfg.Level.Level = logging.InfoLevel
	svc.ReloadConf(cfg)
	assert.Equal(t, svc.Config.Level.Level, logging.InfoLevel)
}
