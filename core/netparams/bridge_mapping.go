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
        "address": "0xC873176F0fbEB036d156C9Bdb4F8288cA9D80C8b"
      },
      "confirmations": 3,
      "multisig_control_contract": {
        "address": "0xf3fb67707ee4c2c4afd156640268dc63852F9b85",
        "deployment_block_height": 37323267
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
        "address": "0xf7989D2902376cad63D0e5B7015efD0CFAd48eB5"
      },
      "confirmations": 3,
      "multisig_control_contract": {
        "address": "0x18E2298DC3B8F1BAa505ce27f07Dba743e205415",
        "deployment_block_height": 37322191
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
