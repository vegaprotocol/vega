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
