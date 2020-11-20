package gql

import (
	"fmt"
	"strconv"

	"github.com/vektah/gqlparser/v2/gqlerror"
	"google.golang.org/grpc/status"

	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"
)

func safeStringUint64(input string) (uint64, error) {
	i, err := strconv.ParseUint(input, 10, 64)
	if err != nil {
		// A conversion error occurred, return the error
		return 0, fmt.Errorf("invalid input string for uint64 conversion %s", input)
	}
	return i, nil
}

func safeStringInt64(input string) (int64, error) {
	i, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		// A conversion error occurred, return the error
		return 0, fmt.Errorf("invalid input string for int64 conversion %s", input)
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

func datetimeToSecondsTS(timestamp string) (int64, error) {
	converted, err := vegatime.Parse(timestamp)
	if err != nil {
		return 0, err
	}
	return converted.UTC().Unix(), nil
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
