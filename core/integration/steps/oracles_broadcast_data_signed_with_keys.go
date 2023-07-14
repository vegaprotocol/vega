// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package steps

import (
	"context"
	"fmt"

	"github.com/cucumber/godog"

	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
)

func OraclesBroadcastDataSignedWithKeys(
	oracleEngine *spec.Engine,
	rawPubKeys string,
	rawProperties *godog.Table,
) error {
	pubKeys := parseOracleDataSigners(rawPubKeys)
	pubKeysSigners := make([]*dstypes.Signer, len(pubKeys))
	for i, s := range pubKeys {
		pubKeysSigners[i] = dstypes.CreateSignerFromString(s, dstypes.SignerTypePubKey)
	}

	properties, err := parseOracleDataProperties(rawProperties)
	if err != nil {
		return err
	}

	// we need a traceID here in case of final MTM settlement -> an idgen is required
	ctx := vgcontext.WithTraceID(context.Background(), "deadbeef")
	return oracleEngine.BroadcastData(ctx, dstypes.Data{
		Signers: pubKeysSigners,
		Data:    properties,
	})
}

func parseOracleDataSigners(rawPubKeys string) []string {
	return StrSlice(rawPubKeys, ",")
}

func parseOracleDataProperties(table *godog.Table) (map[string]string, error) {
	properties := map[string]string{}

	for _, r := range parseOracleBroadcastTable(table) {
		row := oracleDataPropertyRow{row: r}
		_, ok := properties[row.name()]
		if ok {
			return nil, errPropertyRedeclared(row.name())
		}
		properties[row.name()] = row.value()
	}

	return properties, nil
}

func errPropertyRedeclared(name string) error {
	return fmt.Errorf("property %s has been declared multiple times", name)
}

func parseOracleBroadcastTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"name",
		"value",
	}, []string{})
}

type oracleDataPropertyRow struct {
	row RowWrapper
}

func (r oracleDataPropertyRow) name() string {
	return r.row.MustStr("name")
}

func (r oracleDataPropertyRow) value() string {
	return r.row.MustStr("value")
}
