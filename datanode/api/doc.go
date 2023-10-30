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
