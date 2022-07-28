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

package events_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	vgcontext "code.vegaprotocol.io/vega/core/libs/context"

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
