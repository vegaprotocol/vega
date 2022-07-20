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

package events_test

import (
	"context"
	"testing"

	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/events"
	"github.com/stretchr/testify/assert"
)

func TestOracleSpecDeepClone(t *testing.T) {
	ctx := context.Background()

	os := &oraclespb.OracleSpec{
		Id:        "Id",
		CreatedAt: 10000,
		UpdatedAt: 20000,
		PubKeys: []string{
			"PubKey1", "PubKey2",
		},
		Filters: []*oraclespb.Filter{
			{
				Key: &oraclespb.PropertyKey{
					Name: "Name",
					Type: oraclespb.PropertyKey_TYPE_BOOLEAN,
				},
				Conditions: []*oraclespb.Condition{
					{
						Operator: oraclespb.Condition_OPERATOR_EQUALS,
						Value:    "Value",
					},
				},
			},
		},
		Status: oraclespb.OracleSpec_STATUS_ACTIVE,
	}

	osEvent := events.NewOracleSpecEvent(ctx, *os)
	os2 := osEvent.OracleSpec()

	// Change the original values
	os.Id = "Changed"
	os.CreatedAt = 999
	os.UpdatedAt = 999
	os.PubKeys[0] = "Changed1"
	os.PubKeys[1] = "Changed2"
	os.Filters[0].Key.Name = "Changed"
	os.Filters[0].Key.Type = oraclespb.PropertyKey_TYPE_EMPTY
	os.Filters[0].Conditions[0].Operator = oraclespb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL
	os.Filters[0].Conditions[0].Value = "Changed"
	os.Status = oraclespb.OracleSpec_STATUS_UNSPECIFIED

	// Check things have changed
	assert.NotEqual(t, os.Id, os2.Id)
	assert.NotEqual(t, os.CreatedAt, os2.CreatedAt)
	assert.NotEqual(t, os.UpdatedAt, os2.UpdatedAt)
	assert.NotEqual(t, os.PubKeys[0], os2.PubKeys[0])
	assert.NotEqual(t, os.PubKeys[1], os2.PubKeys[1])
	assert.NotEqual(t, os.Filters[0].Key.Name, os2.Filters[0].Key.Name)
	assert.NotEqual(t, os.Filters[0].Key.Type, os2.Filters[0].Key.Type)
	assert.NotEqual(t, os.Filters[0].Conditions[0].Operator, os2.Filters[0].Conditions[0].Operator)
	assert.NotEqual(t, os.Filters[0].Conditions[0].Value, os2.Filters[0].Conditions[0].Value)
	assert.NotEqual(t, os.Status, os2.Status)
}
