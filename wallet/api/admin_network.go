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
