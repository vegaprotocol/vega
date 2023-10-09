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

package entities

import (
	"encoding/hex"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	"github.com/pkg/errors"
)

type BeginBlockEvent interface {
	events.Event
	BeginBlock() eventspb.BeginBlock
}

func BlockFromBeginBlock(b BeginBlockEvent) (*Block, error) {
	hash, err := hex.DecodeString(b.TraceID())
	if err != nil {
		return nil, errors.Wrapf(err, "Trace ID is not valid hex string, trace ID:%s", b.TraceID())
	}

	vegaTime := time.Unix(0, b.BeginBlock().Timestamp)

	// Postgres only stores timestamps in microsecond resolution
	block := Block{
		VegaTime: vegaTime.Truncate(time.Microsecond),
		Hash:     hash,
		Height:   b.BlockNr(),
	}
	return &block, err
}
