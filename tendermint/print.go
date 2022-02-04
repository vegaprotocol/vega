package tendermint

import (
	"encoding/json"
	"fmt"
)

func Prettify(v interface{}) (string, error) {
	marshalledGenesisDoc, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("couldn't marshal tendermint data as JSON: %w", err)
	}
	return string(marshalledGenesisDoc), nil
}
