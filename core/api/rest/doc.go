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

// Package rest contains code for running the REST-to-gRPC gateway.
//
// In order to add a new REST endpoint, add an entry to
// `gateway/rest/grpc-rest-bindings.yml`.
//
// Run `make proto` to generate (among others) the swagger json file.
//
// Run `make rest_check` to make sure that all the endpoints in the bindings
// yaml file make it into the swagger json file.
package rest
