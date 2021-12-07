package faucet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	config "code.vegaprotocol.io/vega/cmd/examples/nullchain/config"
)

func Mint(party, amount, asset string) error {
	values := map[string]string{
		"party":  party,
		"amount": amount,
		"asset":  asset,
	}

	jsonValue, _ := json.Marshal(values)

	r, err := http.Post(config.FaucetAddress, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}

	if r.StatusCode == http.StatusOK {
		return nil
	}

	data, _ := ioutil.ReadAll(r.Body)
	fmt.Println(string(data))
	return nil
}
