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

// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package metrics

import (
	"time"
)

// TimeCounter holds a time.Time and a list of label values, hiding the start time from being accidentally
// overwritten, and removing the need to duplicate the label values.
type TimeCounter struct {
	labelValues []string
	start       time.Time
}

// NewTimeCounter returns a new TimeCounter, with the start time already recorded.
func NewTimeCounter(labelValues ...string) *TimeCounter {
	return &TimeCounter{
		labelValues: labelValues,
		start:       time.Now(),
	}
}

/*
EngineTimeCounterAdd is used to time a function.
e.g.

	func DoSomething() {
		timer := metrics.NewTimeCounter("x", "y", "z")
		// do something
		timer.EngineTimeCounterAdd()
	}
*/
func (tc *TimeCounter) EngineTimeCounterAdd() {
	// Check that the metric has been set up. (Testing does not use metrics.)
	if engineTime == nil {
		return
	}
	engineTime.WithLabelValues(tc.labelValues...).Add(time.Since(tc.start).Seconds())
}

func (tc *TimeCounter) EventTimeCounterAdd() {
	// Check that the metric has been set up. (Testing does not use metrics.)
	if eventHandlingTime == nil {
		return
	}
	eventHandlingTime.WithLabelValues(tc.labelValues...).Add(time.Since(tc.start).Seconds())
}

func (tc *TimeCounter) FlushTimeCounterAdd() {
	// Check that the metric has been set up. (Testing does not use metrics.)
	if flushHandlingTime == nil {
		return
	}
	flushHandlingTime.WithLabelValues(tc.labelValues...).Add(time.Since(tc.start).Seconds())
}

func (tc *TimeCounter) SQLQueryTimeCounterAdd() {
	// Check that the metric has been set up. (Testing does not use metrics.)
	if sqlQueryTime == nil {
		return
	}
	sqlQueryTime.WithLabelValues(tc.labelValues...).Add(time.Since(tc.start).Seconds())
}
