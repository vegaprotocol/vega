package adaptors

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/oracles"
)

type ValidatorFunc func(map[string]string) error

func runValidation(data *oracles.OracleData, validators []ValidatorFunc) error {
	if data == nil {
		return errors.New("no data provided to validate")
	}

	for _, validate := range validators {
		if err := validate(data.Data); err != nil {
			return fmt.Errorf("could not validate data: %w", err)
		}
	}

	return nil
}
