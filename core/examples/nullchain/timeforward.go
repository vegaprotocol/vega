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
