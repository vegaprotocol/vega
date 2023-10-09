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

package visor

import (
	"errors"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/visor/utils"
)

const (
	dataNodeArg              = "datanode"
	homeFlagName             = "--home"
	noHistorySegmentFoundMsg = "no history segments found"
)

var jsonOutputFlag = []string{"--output", "json"}

type latestSegmentCommanndOutput struct {
	LatestSegment struct {
		Height int64 `json:"to_height"`
	}
}

var ErrNoHistorySegmentFound = errors.New(noHistorySegmentFoundMsg)

func latestDataNodeHistorySegment(binary string, binaryArgs Args) (*latestSegmentCommanndOutput, error) {
	args := []string{}

	if binaryArgs.Exists(dataNodeArg) {
		args = append(args, dataNodeArg)
	}

	args = append(args, []string{"network-history", "latest-history-segment"}...)
	args = append(args, binaryArgs.GetFlagWithArg(homeFlagName)...)

	var output latestSegmentCommanndOutput
	_, err := utils.ExecuteBinary(binary, append(args, jsonOutputFlag...), &output)
	if err != nil {
		if strings.Contains(err.Error(), noHistorySegmentFoundMsg) {
			return nil, ErrNoHistorySegmentFound
		}

		return nil, err
	}

	return &output, nil
}

type versionCommandOutput struct {
	Version string `json:"version"`
	Hash    string `json:"hash"`
}

func ensureBinaryVersion(binary, version string) error {
	var output versionCommandOutput

	if _, err := utils.ExecuteBinary(binary, append([]string{"version"}, jsonOutputFlag...), &output); err != nil {
		return err
	}

	if output.Version != version {
		return fmt.Errorf("wrong binary version provided - provided: %s, want: %s", output.Version, version)
	}

	return nil
}
