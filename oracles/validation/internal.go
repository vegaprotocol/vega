package validation

import (
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/oracles"
)

func CheckForInternalOracle(data map[string]string) error {
	for k := range data {
		if strings.HasPrefix(k, oracles.BuiltinOraclePrefix) {
			return fmt.Errorf("%s is not valid: %w", k, oracles.ErrInvalidPropertyKey)
		}
	}

	return nil
}
