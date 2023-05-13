// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package query

import (
	"context"
	"time"

	api "code.vegaprotocol.io/vega/protos/vega/api/v1"

	"google.golang.org/grpc"
)

func getClient(address string) (api.CoreStateServiceClient, error) {
	tdconn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return api.NewCoreStateServiceClient(tdconn), nil
}

func timeoutContext() (context.Context, func()) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
