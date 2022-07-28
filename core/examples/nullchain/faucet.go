// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package nullchain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	config "code.vegaprotocol.io/vega/core/examples/nullchain/config"
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

func FillAccounts(asset, amount string, parties []*Party) error {
	var err error
	for _, party := range parties {

		err = mint(asset, amount, party.pubkey)
		if err != nil {
			return err
		}
		err = MoveByDuration(config.BlockDuration)
		if err != nil {
			return err
		}
	}
	err = MoveByDuration(config.BlockDuration)
	if err != nil {
		return err
	}

	return nil
}
