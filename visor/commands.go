package visor

import (
	"fmt"

	"code.vegaprotocol.io/vega/visor/utils"
)

const (
	dataNodeArg  = "datanode"
	homeFlagName = "--home"
)

var jsonOutputFlag = []string{"--output", "json"}

type latestSegmentCommanndOutput struct {
	LatestSegment struct {
		Height int64 `json:"to_height"`
	}
}

func latestDataNodeHistorySegment(binary string, binaryArgs Args) (*latestSegmentCommanndOutput, error) {
	args := []string{}

	if binaryArgs.Exists(dataNodeArg) {
		args = append(args, dataNodeArg)
	}

	args = append(args, []string{"network-history", "latest-history-segment"}...)
	args = append(args, binaryArgs.GetFlagWithArg(homeFlagName)...)

	var output latestSegmentCommanndOutput
	if _, err := utils.ExecuteBinary(binary, append(args, jsonOutputFlag...), &output); err != nil {
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
