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

var mainnetMirror = `{
  "configs": [
    {
      "network_id": "421614",
      "chain_id": "421614",
      "collateral_bridge_contract": {
        "address": "0x412eD3b1951C39c182ea6682D2a16c1Ca22A5874"
      },
      "confirmations": 3,
      "multisig_control_contract": {
        "address": "0x2F933bf63D4059D66F20F97f4a0B540Ea1d0dE69",
        "deployment_block_height": 46048993
      },
      "block_time": "250ms",
      "name": "Arbitrum (Sepolia)"
    }
  ]
}`

var validatorsTestnet = `{
  "configs": [
    {
      "network_id": "421614",
      "chain_id": "421614",
      "collateral_bridge_contract": {
        "address": "0x927067717B0A9bd553fC421Ae63b3377694b4166"
      },
      "confirmations": 3,
      "multisig_control_contract": {
        "address": "0x752faCb1e1EEf7A5a154db5Bf54988E80b0e96Da",
        "deployment_block_height": 43630575
      },
      "block_time": "250ms",
      "name": "Arbitrum (Sepolia)"
    }
  ]
}`

var mainnet = `{
  "configs": [
    {
      "network_id": "42161",
      "chain_id": "42161",
      "collateral_bridge_contract": {
        "address": "0x475B597652bCb2769949FD6787b1DC6916518407"
      },
      "confirmations": 3,
      "multisig_control_contract": {
        "address": "0x348372DE65Ca7F2567FE267ccc4D1bF6d4b71f6F",
        "deployment_block_height": 213213613
      },
      "block_time": "250ms",
      "name": "Arbitrum One"
    }
  ]
}`

var bridgeMapping = map[string]string{
	"vega-stagnet1-202307191148":       stagnet1,
	"vega-fairground-202305051805":     testnet,
	"vega-mainnet-mirror-202306231148": mainnetMirror,
	"vega-testnet-0002-v4":             validatorsTestnet,
	"vega-mainnet-0011":                mainnet,
}
