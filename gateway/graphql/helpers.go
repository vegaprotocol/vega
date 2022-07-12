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

package gql

import (
	"fmt"
	"strconv"

	"github.com/vektah/gqlparser/v2/gqlerror"
	"google.golang.org/grpc/status"

	"code.vegaprotocol.io/data-node/vegatime"
	types "code.vegaprotocol.io/protos/vega"
)

func safeStringUint64(input string) (uint64, error) {
	i, err := strconv.ParseUint(input, 10, 64)
	if err != nil {
		// A conversion error occurred, return the error
		return 0, fmt.Errorf("invalid input string for uint64 conversion %s", input)
	}
	return i, nil
}

// customErrorFromStatus provides a richer error experience from grpc ErrorDetails
// which is provided by the Vega grpc API. This helper takes in the error provided
// by a grpc client and either returns a custom graphql error or the raw error string.
func customErrorFromStatus(err error) error {
	st, ok := status.FromError(err)
	if ok {
		customCode := ""
		customDetail := ""
		customInner := ""
		customMessage := st.Message()
		errorDetails := st.Details()
		for _, s := range errorDetails {
			det := s.(*types.ErrorDetail)
			customDetail = det.Message
			customCode = fmt.Sprintf("%d", det.Code)
			customInner = det.Inner
			break
		}
		return &gqlerror.Error{
			Message: customMessage,
			Extensions: map[string]interface{}{
				"detail": customDetail,
				"code":   customCode,
				"inner":  customInner,
			},
		}
	}
	return err
}

func secondsTSToDatetime(timestampInSeconds int64) string {
	return vegatime.Format(vegatime.Unix(timestampInSeconds, 0))
}

func nanoTSToDatetime(timestampInNanoSeconds int64) string {
	return vegatime.Format(vegatime.UnixNano(timestampInNanoSeconds))
}

func convertVersion(version *int) (uint64, error) {
	const defaultValue = 0

	if version != nil {
		if *version >= 0 {
			return uint64(*version), nil
		}
		return defaultValue, fmt.Errorf("invalid version value %d", *version)
	}
	return defaultValue, nil
}
