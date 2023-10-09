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

package networkhistory

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"github.com/cenkalti/backoff"
)

type BlockCommitHandler struct {
	log                       *logging.Logger
	cfg                       Config
	snapshotData              func(ctx context.Context, chainID string, toHeight int64) error
	usingEventFile            bool
	eventFileTimeBetweenBlock time.Duration

	timeBetweenRetries time.Duration
	maxRetries         uint64
}

func NewBlockCommitHandler(
	log *logging.Logger,
	cfg Config,
	snapshotData func(ctx context.Context, chainID string, toHeight int64) error,
	usingEventFile bool, eventFileTimeBetweenBlock time.Duration,
	timeBetweenRetries time.Duration,
	maxRetries uint64,
) *BlockCommitHandler {
	return &BlockCommitHandler{
		log:                       log.Named("block-commit-handler"),
		cfg:                       cfg,
		snapshotData:              snapshotData,
		usingEventFile:            usingEventFile,
		eventFileTimeBetweenBlock: eventFileTimeBetweenBlock,
		timeBetweenRetries:        timeBetweenRetries,
		maxRetries:                maxRetries,
	}
}

func (b *BlockCommitHandler) OnBlockCommitted(ctx context.Context, chainID string, blockHeight int64, snapshotTaken bool) {
	snapTaken := snapshotTaken
	if b.usingEventFile && b.eventFileTimeBetweenBlock < time.Second {
		snapTaken = blockHeight%1000 == 0
	}

	if blockHeight > 0 && bool(b.cfg.Publish) && snapTaken {
		snapshotData := func() (opErr error) {
			err := b.snapshotData(ctx, chainID, blockHeight)
			if err != nil {
				b.log.Errorf("failed to snapshot data, retrying in %v: %v", b.timeBetweenRetries, err)
			}

			return err
		}

		constantBackoff := backoff.NewConstantBackOff(b.timeBetweenRetries)
		backoff.WithMaxRetries(constantBackoff, 6)

		err := backoff.Retry(snapshotData, backoff.WithMaxRetries(constantBackoff, b.maxRetries))
		if err != nil {
			b.log.Panic(fmt.Sprintf("failed to snapshot data after %d retries", b.maxRetries), logging.Error(err))
		}
	}
}
