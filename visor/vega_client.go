// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package visor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type UpgradeStatusResponse struct {
	Result struct {
		ReadyToUpgrade      bool
		AcceptedReleaseInfo struct {
			VegaReleaseTag     string
			DatanodeReleaseTag string
		}
	} `json:"result"`
}

// TODO - use actual API from Core
func UpgradeStatus(url string) (*UpgradeStatusResponse, error) {
	payload := `{"method": "protocolupgrade.UpgradeStatus", "params": [],"id": "1"}`

	res, err := http.Post(url, "application/json", strings.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to call api: %w", err)
	}

	b, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to parse response api: %w", err)
	}

	r := UpgradeStatusResponse{}
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &r, nil
}
