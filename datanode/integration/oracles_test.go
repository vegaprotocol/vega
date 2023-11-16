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

package integration_test

import (
	"testing"
)

func TestOracles(t *testing.T) {
	queries := map[string]string{
		"OracleDataSourceExternal":     `{ oracleSpecsConnection { edges { node { dataSourceSpec { spec { id createdAt updatedAt status data { sourceType { ... on DataSourceDefinitionExternal { sourceType { ... on DataSourceSpecConfiguration { signers { signer { ... on ETHAddress { address } ... on PubKey { key } } } filters { key { name  type } conditions { operator value } } } } } } } } } } } } }`,
		"OracleDataConnectionExternal": `{ oracleSpecsConnection { edges { node { dataConnection { edges { node { externalData { data { matchedSpecIds broadcastAt } } } } } } } } }`,
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			assertGraphQLQueriesReturnSame(t, query)
		})
	}

	queries = map[string]string{
		"OracleDataSourceExternalEthereum":     `{ oracleSpecsConnection { edges { node { dataSourceSpec { spec { id createdAt updatedAt status data { sourceType { ... on DataSourceDefinitionExternal { sourceType { ... on EthCallSpec { abi args address method requiredConfirmations normalisers { name expression } trigger { trigger { ... on EthTimeTrigger { initial every until } } } filters { key { name  type } conditions { operator value } } } } } } } } } } } } }`,
		"OracleDataConnectionExternalEthereum": `{ oracleSpecsConnection { edges { node { dataConnection { edges { node { externalData { data { matchedSpecIds broadcastAt } } } } } } } } }`,
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			assertGraphQLQueriesReturnSame(t, query)
		})
	}

	queries = map[string]string{
		"OracleDataSourceInternal":     `{ oracleSpecsConnection { edges { node { dataSourceSpec { spec { id createdAt updatedAt status data { sourceType { ... on DataSourceDefinitionInternal { sourceType { ... on DataSourceSpecConfigurationTime { conditions { operator value }  } } } } } } } } } } }`,
		"OracleDataConnectionInternal": `{ oracleSpecsConnection { edges { node { dataConnection { edges { node { externalData { data { matchedSpecIds broadcastAt } } } } } } } } }`,
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			assertGraphQLQueriesReturnSame(t, query)
		})
	}
}
