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

package common

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/libs/ptr"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

type InternalTimeTrigger struct {
	// This is optional to reflect the proto, but it will always be set by the governance
	Initial     *time.Time
	Every       int64
	nextTrigger *time.Time
}

func (i InternalTimeTrigger) String() string {
	return fmt.Sprintf(
		"initial(%v) every(%d) nextTrigger(%v)",
		i.Initial,
		i.Every,
		i.nextTrigger,
	)
}

func (i InternalTimeTrigger) IntoProto() *datapb.InternalTimeTrigger {
	var initial *int64
	if i.Initial != nil {
		initial = ptr.From(i.Initial.Unix())
	}

	return &datapb.InternalTimeTrigger{
		Initial: initial,
		Every:   i.Every,
	}
}

func (i InternalTimeTrigger) DeepClone() *InternalTimeTrigger {
	var initial *time.Time
	if i.Initial != nil {
		initial = *ptr.From(i.Initial)
	}

	var nextTrigger *time.Time
	if i.nextTrigger != nil {
		nextTrigger = *ptr.From(i.nextTrigger)
	}

	return &InternalTimeTrigger{
		Initial:     initial,
		Every:       i.Every,
		nextTrigger: nextTrigger,
	}
}

func (i InternalTimeTrigger) IsTriggered(timeNow time.Time) bool {
	if i.nextTrigger != nil {
		if i.nextTrigger.Before(timeNow) {
			*i.nextTrigger = i.nextTrigger.Add(time.Duration(i.Every) * time.Second)
			return true
		}
	}

	return false
}

func (i *InternalTimeTrigger) SetNextTrigger(timeNow time.Time) {
	if i.Initial == nil {
		// Set panic
		panic("initial time value is missing")
	}

	i.nextTrigger = ptr.From(*i.Initial)

	// If initial > timeNow, we never been triggered
	// so we set the next trigger to `initial`
	if i.Initial.After(timeNow) {
		return
	}

	// If `initial` is in the past, we have been triggered already
	// then -> find when the next trigger is
	for i.nextTrigger.Before(timeNow) {
		*i.nextTrigger = i.nextTrigger.Add(time.Duration(i.Every) * time.Second)
	}
}

func (i *InternalTimeTrigger) SetInitial(initial, timeNow time.Time) {
	if i.Initial != nil {
		// this is incorrect, we should only overwrite time
		// when not submitted by the user
		panic("invalid initial time override")
	}

	i.Initial = ptr.From(initial)
	i.SetNextTrigger(timeNow)
}

func InternalTimeTriggerFromProto(
	protoTrigger *datapb.InternalTimeTrigger,
	timeNow time.Time,
) *InternalTimeTrigger {
	var initial *time.Time
	if protoTrigger.Initial != nil {
		initial = ptr.From(time.Unix(*protoTrigger.Initial, 0))
	}

	tt := &InternalTimeTrigger{
		Initial: initial,
		Every:   protoTrigger.Every,
	}

	if initial != nil {
		tt.SetNextTrigger(timeNow)
	}

	return tt
}

type InternalTimeTriggers [1]*InternalTimeTrigger

func (i InternalTimeTriggers) Empty() error {
	if len(i) <= 0 || i[0] == nil {
		return errors.New("no time trigger is set")
	}

	return nil
}

func (i InternalTimeTriggers) String() string {
	if len(i) != 1 {
		return "[]"
	}

	strs := make([]string, 0, len(i))
	for _, f := range i {
		// We handle length of 1 for the moment, but later will be extended.
		if f == nil {
			strs = append(strs, "nil")
			continue
		}
		strs = append(strs, f.String())
	}
	return "[" + strings.Join(strs, ", ") + "]"
}

func (i InternalTimeTriggers) IntoProto() []*datapb.InternalTimeTrigger {
	protoTriggers := [1]*datapb.InternalTimeTrigger{}
	if len(i) == 1 {
		if len(i) == 1 && i[0] != nil {
			protoTriggers[0] = i[0].IntoProto()
		}
	}

	return protoTriggers[:]
}

func InternalTimeTriggersFromProto(protoTriggers []*datapb.InternalTimeTrigger, timeNow time.Time) InternalTimeTriggers {
	ts := InternalTimeTriggers{}
	for i, protoTrigger := range protoTriggers {
		// Handle length of 1 for now
		if i == 0 {
			ts[0] = InternalTimeTriggerFromProto(protoTrigger, timeNow)
		}
	}

	return ts
}

func (i InternalTimeTriggers) DeepClone() InternalTimeTriggers {
	clonedTriggers := InternalTimeTriggers{}
	if len(i) == 1 && i[0] != nil {
		clonedTriggers[0] = i[0].DeepClone()
	}

	return clonedTriggers
}

func (i InternalTimeTriggers) IsTriggered(now time.Time) bool {
	for _, v := range i {
		if v.IsTriggered(now) {
			return true
		}
	}

	return false
}
