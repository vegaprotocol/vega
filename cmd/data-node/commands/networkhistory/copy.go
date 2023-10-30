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
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/logging"
)

type copyCmd struct {
	config.VegaHomeFlag
	config.Config
}

func (cmd *copyCmd) Execute(args []string) error {
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	cfg := logging.NewDefaultConfig()
	cfg.Custom.Zap.Level = logging.InfoLevel
	cfg.Environment = "custom"
	log := logging.NewLoggerFromConfig(
		cfg,
	)
	defer log.AtExit()

	if len(args) != 2 {
		return errors.New("expected <history-segment-id> <output-file>")
	}

	segmentID := args[0]
	outputPath := args[1]

	client := getDatanodeAdminClient(log, cmd.Config)
	reply, err := client.CopyHistorySegmentToFile(ctx, segmentID, outputPath)
	if err != nil {
		return fmt.Errorf("failed to copy history segment to file: %w", err)
	}

	if reply.Err != nil {
		return reply.Err
	}

	log.Info(reply.Reply)

	return nil
}
