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

package events_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	vgcontext "code.vegaprotocol.io/vega/libs/context"

	"github.com/stretchr/testify/assert"
)

func TestTimeEvent(t *testing.T) {
	now := time.Now()
	ctx := context.Background()
	e := events.NewTime(ctx, now)
	assert.Equal(t, e.Time(), now)
	assert.Equal(t, events.TimeUpdate, e.Type())
	assert.NotEmpty(t, e.TraceID())
	_, trace := vgcontext.TraceIDFromContext(e.Context())
	assert.NotNil(t, trace)
	assert.Equal(t, trace, e.TraceID())
	assert.Zero(t, e.Sequence())
}

func TestContextReplace(t *testing.T) {
	now := time.Now()
	ctx := vgcontext.WithBlockHeight(context.Background(), 100)
	e := events.NewTime(ctx, now)
	assert.Equal(t, int64(100), e.BlockNr())

	ctx = vgcontext.WithBlockHeight(context.Background(), 200)
	e.Replace(ctx)
	assert.Equal(t, int64(200), e.BlockNr())
}
