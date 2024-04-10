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

package netparams

var stagnet1 = `{
	"network_id": "421614",
	"chain_id": "421614",
	"collateral_bridge_contract": {
	  "address": "0x52d95d30fc8e4d8fe9cc7ce285d0c07c8e629719"
	},
	"confirmations": 3,
	"multisig_control_contract": {
	  "address": "0x764c51de728f09407f7f073f63fc0a8a6adf110e",
	  "deployment_block_height": 27160717
	}
  }`

var testnet = `{
	"network_id": "421614",
	"chain_id": "421614",
	"collateral_bridge_contract": {
	  "address": "0x204F34b7D14b7eca9f95D9D6322bbdc2e51eCAa7"
	},
	"confirmations": 3,
	"multisig_control_contract": {
	  "address": "0x0A3f3E72FCe9862c750B0682aA75bb7261b3eb15",
	  "deployment_block_height": 31628794
	}
  }`

var bridgeMapping = map[string]string{
	"vega-stagnet1-202307191148":       stagnet1,
	"vega-fairground-202305051805":     testnet,
	"vega-mainnet-mirror-202306231148": "{}",
	"vega-testnet-0002-v4":             "{}",
	"vega-mainnet-0011":                "{}",
}
