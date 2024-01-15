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
	if i.nextTrigger == nil {
		return false
	}

	triggered := false
	for i.nextTrigger.Before(timeNow) {
		triggered = true
		*i.nextTrigger = i.nextTrigger.Add(time.Duration(i.Every) * time.Second)
	}

	return triggered
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
}

func InternalTimeTriggerFromProto(
	protoTrigger *datapb.InternalTimeTrigger,
) *InternalTimeTrigger {
	var initial *time.Time
	if protoTrigger.Initial != nil {
		initial = ptr.From(time.Unix(*protoTrigger.Initial, 0))
	}

	tt := &InternalTimeTrigger{
		Initial: initial,
		Every:   protoTrigger.Every,
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

func InternalTimeTriggersFromProto(protoTriggers []*datapb.InternalTimeTrigger) InternalTimeTriggers {
	ts := InternalTimeTriggers{}
	for i, protoTrigger := range protoTriggers {
		// Handle length of 1 for now
		if i == 0 {
			ts[0] = InternalTimeTriggerFromProto(protoTrigger)
		}
	}

	return ts
}

func (i InternalTimeTriggers) DeepClone() *InternalTimeTriggers {
	clonedTriggers := &InternalTimeTriggers{}
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
