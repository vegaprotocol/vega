package api_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	vgfs "code.vegaprotocol.io/vega/libs/fs"
)

type openrpc struct {
	Methods    []method   `json:"methods"`
	Components components `json:"components"`
}

func (o openrpc) ASTForParams(methodName string) (astNode, error) {
	m := o.method(methodName)

	numParams := len(m.Params)

	// No params means nil.
	if numParams == 0 {
		return astNode{}, nil
	}

	rootNode := astNode{
		nodeType:         nodeTypeObject,
		nestedProperties: make([]astNode, 0, numParams),
	}

	for _, p := range m.Params {
		paramNode, err := o.parseParamSchema(p)
		if err != nil {
			return astNode{}, fmt.Errorf("could not parse params %q: %w", p.Name, err)
		}

		rootNode.nestedProperties = append(rootNode.nestedProperties, paramNode)
	}

	rootNode.nestedProperties = deterministNestedProperties(rootNode.nestedProperties)

	return rootNode, nil
}

func (o openrpc) ASTForResult(methodName string) (astNode, error) {
	m := o.method(methodName)

	return o.parseSchema(m.Result.Schema)
}

func (o openrpc) method(methodName string) method {
	for _, m := range o.Methods {
		if m.Name == methodName {
			return m
		}
	}
	panic(fmt.Sprintf("No method %q in the openrpc.json file", methodName))
}

func (o openrpc) component(ref string) schema {
	componentName := strings.Split(ref, "/")[3]
	componentSchema, ok := o.Components.Schemas[componentName]
	if !ok {
		panic(fmt.Sprintf("could not find the component %q", componentName))
	}
	return componentSchema
}

func (o openrpc) parseParamSchema(p paramsDescriptor) (astNode, error) {
	paramsNode, err := o.parseSchema(p.Schema)
	if err != nil {
		return astNode{}, fmt.Errorf("could not parse the params schema: %w", err)
	}
	paramsNode.name = p.Name

	return paramsNode, nil
}

func (o openrpc) parseSchema(s schema) (astNode, error) {
	if len(s.Ref) != 0 {
		componentSchema := o.component(s.Ref)
		return o.parseSchema(componentSchema)
	}

	if s.Type == "null" {
		return astNode{}, nil
	}

	jsonType, err := openrpcTypeToJSONType(s.Type)
	if err != nil {
		return astNode{}, fmt.Errorf("could not figure out the JSON type from OpenRPC type %q: %w", s.Type, err)
	}

	if jsonType == nodeTypeObject {
		return o.parseObjectSchema(s)
	} else if jsonType == nodeTypeArray {
		return o.parseArraySchema(s)
	}

	return astNode{
		nodeType: jsonType,
	}, nil
}

func (o openrpc) parseArraySchema(s schema) (astNode, error) {
	arrayNode := astNode{
		nodeType: nodeTypeArray,
	}

	if s.Items != nil {
		itemProperty, err := o.parseSchema(*s.Items)
		if err != nil {
			return astNode{}, fmt.Errorf("could not parse the item schema: %w", err)
		}
		arrayNode.nestedProperties = []astNode{itemProperty}
	}
	return arrayNode, nil
}

func (o openrpc) parseObjectSchema(s schema) (astNode, error) {
	if len(s.Properties) == 0 && len(s.PatternProperties) == 0 {
		return astNode{nodeType: nodeTypeObject}, nil
	}

	if len(s.PatternProperties) != 0 {
		// Very weird stuff to handle the weird definition of the map[string] in openRPC
		// format...
		for _, s := range s.PatternProperties {
			patternNode, err := o.parseSchema(s.Schema)
			if err != nil {
				return astNode{}, fmt.Errorf("could not parse the schema for pattern properties: %w", err)
			}

			// Yes we return after the first element because we don't support
			// multiple patterns.
			return astNode{
				nodeType:         nodeTypeObject,
				nestedProperties: []astNode{patternNode},
			}, nil
		}
	}

	nestedProperties := make([]astNode, 0, len(s.Properties))
	for propertyName, schema := range s.Properties {
		propertyNode, err := o.parseSchema(schema)
		if err != nil {
			return astNode{}, fmt.Errorf("could not parse the schema for property %q: %w", propertyName, err)
		}
		propertyNode.name = propertyName
		nestedProperties = append(nestedProperties, propertyNode)
	}

	return astNode{
		nodeType:         nodeTypeObject,
		nestedProperties: deterministNestedProperties(nestedProperties),
	}, nil
}

type components struct {
	Schemas map[string]schema `json:"schemas"`
}

type method struct {
	Name   string             `json:"name"`
	Params []paramsDescriptor `json:"params"`
	Result resultDescriptor   `json:"result"`
}

type paramsDescriptor struct {
	Name   string `json:"name"`
	Schema schema `json:"schema"`
}

type resultDescriptor struct {
	Schema schema `json:"schema"`
}

type schema struct {
	Type              string                     `json:"type,omitempty"`
	Properties        map[string]schema          `json:"properties,omitempty"`
	Ref               string                     `json:"$ref,omitempty"`
	Items             *schema                    `json:"items,omitempty"`
	PatternProperties map[string]patternProperty `json:"patternProperties"`
}

type patternProperty struct {
	Schema schema `json:"schema"`
}

func parseASTFromDoc(t *testing.T, methodName string) (methodIODefinition, error) {
	t.Helper()

	rawDoc, err := vgfs.ReadFile("./openrpc.json")
	if err != nil {
		return methodIODefinition{}, fmt.Errorf("could not read the OpenRPC documentation file: %w", err)
	}

	doc := &openrpc{}
	if err := json.Unmarshal(rawDoc, doc); err != nil {
		return methodIODefinition{}, fmt.Errorf("could not parse the OpenRPC documentation file: %w", err)
	}

	paramsAST, err := doc.ASTForParams(methodName)
	if err != nil {
		return methodIODefinition{}, fmt.Errorf("could not build the params AST for method %q: %w", methodName, err)
	}

	resultAST, err := doc.ASTForResult(methodName)
	if err != nil {
		return methodIODefinition{}, fmt.Errorf("could not build the result AST for method %q: %w", methodName, err)
	}

	return methodIODefinition{
		Params: paramsAST,
		Result: resultAST,
	}, nil
}

func openrpcTypeToJSONType(openrpcType string) (nodeType, error) {
	switch openrpcType {
	case "string":
		return nodeTypeString, nil
	case "number":
		return nodeTypeNumber, nil
	case "boolean":
		return nodeTypeBoolean, nil
	case "object":
		return nodeTypeObject, nil
	case "array":
		return nodeTypeArray, nil
	default:
		return nodeTypeUnknown, fmt.Errorf("openRPC type %q cannot be converted to node type", openrpcType)
	}
}
