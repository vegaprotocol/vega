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
  "configs": [
    {
      "network_id": "421614",
      "chain_id": "421614",
      "collateral_bridge_contract": {
        "address": "0x41C013Ed92337DBf44FC6d66a1f9d0cC5B46C389"
      },
      "confirmations": 3,
      "multisig_control_contract": {
        "address": "0x9138E4B468A4315FE05885eff7485c7244c65343",
        "deployment_block_height": 36996691
      },
      "block_time": "250ms",
      "name": "Arbitrum (Sepolia)"
    }
  ]
}`

var testnet = `{
  "configs": [
    {
      "network_id": "421614",
      "chain_id": "421614",
      "collateral_bridge_contract": {
        "address": "0x55c5b54930fB75e7e59f8bD953910B1bdff16340"
      },
      "confirmations": 3,
      "multisig_control_contract": {
        "address": "0x0476C5A7171aF83C14C6DfE0cF7FB6Ca507Ef0A1",
        "deployment_block_height": 36998306
      },
      "block_time": "250ms",
      "name": "Arbitrum (Sepolia)"
    }
  ]
}`

var bridgeMapping = map[string]string{
	"vega-stagnet1-202307191148":       stagnet1,
	"vega-fairground-202305051805":     testnet,
	"vega-mainnet-mirror-202306231148": "{}",
	"vega-testnet-0002-v4":             "{}",
	"vega-mainnet-0011":                "{}",
}
