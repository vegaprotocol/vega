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

package vegatime

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/broker/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestTimeUpdateEventIsSentBeforeCallbacksAreInvoked(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := mocks.NewMockBroker(ctrl)

	s := New(Config{}, m)

	callOrder := make([]int, 0)
	m.EXPECT().Send(gomock.Any()).DoAndReturn(func(any interface{}) { callOrder = append(callOrder, 1) })
	s.NotifyOnTick(func(ctx context.Context, t time.Time) { callOrder = append(callOrder, 2) })
	s.SetTimeNow(context.Background(), time.Now())

	assert.Equal(t, 1, callOrder[0])
	assert.Equal(t, 2, callOrder[1])
}
