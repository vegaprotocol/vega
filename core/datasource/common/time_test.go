// Copyright (c) 2023 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package common_test

import (
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/libs/ptr"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/stretchr/testify/assert"
)

func TestInternalTimeTriggerString(t *testing.T) {
	timeNow := time.Now()
	nt := timeNow.Add(time.Minute)

	tt := common.InternalTimeTrigger{
		Initial: &timeNow,
		Every:   int64(15),
	}

	tt.SetNextTrigger(nt)
	assert.Equal(
		t,
		fmt.Sprintf("initial(%s) every(15) nextTrigger(%s)", timeNow, nt),
		tt.String(),
	)
}

func TestInternalTimeTriggerIntoProto(t *testing.T) {
	timeNow := time.Now()
	tt := common.InternalTimeTrigger{
		Initial: &timeNow,
		Every:   int64(15),
	}

	pt := tt.IntoProto()
	assert.NotNil(t, pt)
	assert.IsType(t, &datapb.InternalTimeTrigger{}, pt)
	assert.Equal(t, timeNow.Unix(), *pt.Initial)
	assert.Equal(t, int64(15), pt.Every)
}

func TestInternalTimeTriggerDeepClone(t *testing.T) {
	timeNow := time.Now()
	tt := common.InternalTimeTrigger{
		Initial: &timeNow,
		Every:   int64(15),
	}

	pt := tt.DeepClone()
	assert.NotNil(t, pt)
	assert.IsType(t, &common.InternalTimeTrigger{}, pt)
	assert.Equal(t, timeNow, *pt.Initial)
	assert.Equal(t, int64(15), pt.Every)
}

func TestInternalTimeTriggerIsTriggered(t *testing.T) {
	timeNow := time.Now()
	nt := timeNow.Add(time.Minute)
	tt := common.InternalTimeTrigger{
		Initial: &timeNow,
		Every:   int64(15),
	}

	tt.SetNextTrigger(nt)
	// Given time is before the next trigger
	triggered := tt.IsTriggered(timeNow)
	assert.Equal(t, false, triggered)

	// Given time is the same as the next trigger
	triggered = tt.IsTriggered(nt)
	assert.Equal(t, false, triggered)

	// Given time is after the next trigger
	triggered = tt.IsTriggered(nt.Add(time.Second * 2))
	assert.Equal(t, true, triggered)
}

func TestInternalTimeTriggerFromProto(t *testing.T) {
	timeNow := time.Now()
	tt := common.InternalTimeTrigger{
		Initial: &timeNow,
		Every:   int64(15),
	}

	pt := tt.IntoProto()
	ntt := common.InternalTimeTriggerFromProto(pt)
	assert.NotNil(t, pt)
	assert.IsType(t, &common.InternalTimeTrigger{}, ntt)
	assert.Equal(t, ptr.From(time.Unix(timeNow.Unix(), 0)), ntt.Initial)
	assert.Equal(t, int64(15), ntt.Every)
}

func TestInternalTimeTriggersString(t *testing.T) {
	timeNow := time.Now()

	tt := &common.InternalTimeTrigger{
		Initial: &timeNow,
		Every:   int64(15),
	}

	var ttl common.InternalTimeTriggers
	assert.Equal(t, "[nil]", ttl.String())

	ttl = common.InternalTimeTriggers{}
	assert.Equal(t, "[nil]", ttl.String())

	ttl = common.InternalTimeTriggers{tt}
	assert.Equal(
		t,
		fmt.Sprintf("[initial(%s) every(15) nextTrigger(<nil>)]", timeNow),
		ttl.String(),
	)
}

func TestInternalTimeTriggersIntoProto(t *testing.T) {
	timeNow := time.Now()
	tt := &common.InternalTimeTrigger{
		Initial: &timeNow,
		Every:   int64(15),
	}

	ttl := common.InternalTimeTriggers{tt}

	pt := ttl.IntoProto()
	assert.NotNil(t, pt)
	assert.IsType(t, []*datapb.InternalTimeTrigger{}, pt)
	assert.Equal(t, 1, len(pt))
	assert.Equal(t, timeNow.Unix(), *pt[0].Initial)
	assert.Equal(t, int64(15), pt[0].Every)
}

func TestInternalTimeTriggersIsTriggered(t *testing.T) {
	timeNow := time.Now()
	nt := timeNow.Add(time.Minute)
	tt := &common.InternalTimeTrigger{
		Initial: &timeNow,
		Every:   int64(15),
	}

	tt.SetNextTrigger(nt)
	ttl := common.InternalTimeTriggers{tt}

	// Given time is before the next trigger
	triggered := ttl.IsTriggered(timeNow)
	assert.Equal(t, false, triggered)

	// Given time is the same as the next trigger
	triggered = ttl.IsTriggered(nt)
	assert.Equal(t, false, triggered)

	// Given time is after the next trigger
	triggered = ttl.IsTriggered(nt.Add(time.Second * 15))
	assert.Equal(t, true, triggered)

	// check trigger time is progressed
	triggered = ttl.IsTriggered(nt.Add(time.Second * 15))
	assert.Equal(t, false, triggered)
}
