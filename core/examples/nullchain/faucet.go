// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
