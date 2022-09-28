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

package pipe

import (
	"fmt"

	"go.uber.org/atomic"
)

var pipeCounter = atomic.Uint64{}

type Addr struct {
	name         string
	serialNumber uint64
}

func NewAddr(name string) *Addr {
	return &Addr{
		name:         name,
		serialNumber: pipeCounter.Inc(),
	}
}

func (p *Addr) String() string {
	return fmt.Sprintf("%s:%v", p.name, p.serialNumber)
}

func (p *Addr) Network() string {
	return "pipe"
}
