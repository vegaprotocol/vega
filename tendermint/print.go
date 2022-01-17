package tendermint

import (
	"fmt"

	tmjson "github.com/tendermint/tendermint/libs/json"
)

func Prettify(v interface{}) (string, error) {
	marshalledGenesisDoc, err := tmjson.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("couldn't marshal tendermint data as JSON: %w", err)
	}
	return string(marshalledGenesisDoc), nil
}
