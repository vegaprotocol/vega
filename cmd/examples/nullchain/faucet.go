package nullchain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	config "code.vegaprotocol.io/vega/cmd/examples/nullchain/config"
)

var ErrFaucet = errors.New("faucet failed")

func mint(asset, amount, party string) error {
	values := map[string]string{
		"party":  party,
		"amount": amount,
		"asset":  asset,
	}

	jsonValue, _ := json.Marshal(values)

	r, err := http.Post(config.FaucetAddress, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return fmt.Errorf("faucet failed: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode == http.StatusOK {
		return nil
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("time forward failed: %w", err)
	}
	return fmt.Errorf("%w: %s", ErrFaucet, string(data))
}

func FillAccounts(asset, amount string, parties []*Party) {
	for _, party := range parties {
		mint(asset, amount, party.pubkey)
		MoveByDuration(config.BlockDuration)
	}
	MoveByDuration(config.BlockDuration)
}
