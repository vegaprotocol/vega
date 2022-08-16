// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
