package api

import "code.vegaprotocol.io/vega/wallet/network"

type AdminNetwork struct {
	Name     string             `json:"name"`
	Metadata []network.Metadata `json:"metadata"`
	API      AdminAPIConfig     `json:"api"`
	Apps     AdminAppConfig     `json:"apps"`
}

type AdminAPIConfig struct {
	GRPC    AdminGRPCConfig    `json:"grpc"`
	REST    AdminRESTConfig    `json:"rest"`
	GraphQL AdminGraphQLConfig `json:"graphQL"`
}

type AdminGRPCConfig struct {
	Hosts []string `json:"hosts"`
}

type AdminRESTConfig struct {
	Hosts []string `json:"hosts"`
}

type AdminGraphQLConfig struct {
	Hosts []string `json:"hosts"`
}

type AdminAppConfig struct {
	Explorer   string `json:"explorer"`
	Console    string `json:"console"`
	Governance string `json:"governance"`
}
