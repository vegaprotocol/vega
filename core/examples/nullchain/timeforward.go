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
	"time"

	config "code.vegaprotocol.io/vega/core/examples/nullchain/config"
)

var ErrTimeForward = errors.New("time forward failed")

func move(raw string) error {
	values := map[string]string{"forward": raw}

	jsonValue, _ := json.Marshal(values)

	r, err := http.Post(config.TimeforwardAddress, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return fmt.Errorf("time forward failed: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode == http.StatusOK {
		return nil
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("time forward failed: %w", err)
	}
	return fmt.Errorf("%w: %s", ErrTimeForward, string(data))
}

func MoveByDuration(d time.Duration) error {
	return move(d.String())
}

func MoveToDate(t time.Time) error {
	return move(t.Format(time.RFC3339))
}
