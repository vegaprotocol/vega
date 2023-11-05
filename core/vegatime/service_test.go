// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
