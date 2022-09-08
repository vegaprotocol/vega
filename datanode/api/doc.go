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

// Package api contains code for running the gRPC server.
//
// In order to add a new gRPC endpoint, add proto content (rpc call, request
// and response messages), then add the endpoint function implementation in
// `api/somefile.go`. Example:
//
//	func (s *tradingService) SomeNewEndpoint(
//	    ctx context.Context, req *protoapi.SomeNewEndpointRequest,
//	) (*protoapi.SomeNewEndpointResponse, error) {
//	    /* Implementation goes here */
//	    return &protoapi.SomeNewEndpointResponse{/* ... */}, nil
//	}
//
// Add a test for the newly created endpoint in `api/trading_test.go`.
package api
