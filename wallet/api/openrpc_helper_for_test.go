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

package api_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

type nodeType uint16

const (
	nodeTypeUnknown nodeType = 0
	nodeTypeString  nodeType = 1
	nodeTypeNumber  nodeType = 2
	nodeTypeBoolean nodeType = 3
	nodeTypeObject  nodeType = 4
	nodeTypeArray   nodeType = 5
)

// astNode represents the object that is going to be walked over.
type astNode struct {
	name             string
	nodeType         nodeType
	nestedProperties []astNode
}

type methodIODefinition struct {
	Params astNode
	Result astNode
}

func assertEqualSchema(t *testing.T, method string, params interface{}, result interface{}) {
	t.Helper()

	paramsAST, err := parseASTFromGo(t, params)
	require.NoError(t, err)

	resultAST, err := parseASTFromGo(t, result)
	require.NoError(t, err)

	mioDefinitionFromGo := methodIODefinition{
		Params: paramsAST,
		Result: resultAST,
	}

	mioDefinitionFromDoc, err := parseASTFromDoc(t, method)
	require.NoError(t, err)

	require.Equal(t, mioDefinitionFromGo, mioDefinitionFromDoc, "The openRPC and the go code are not in sync!")
}

func deterministNestedProperties(nodes []astNode) []astNode {
	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].name < nodes[j].name
	})

	return nodes
}
