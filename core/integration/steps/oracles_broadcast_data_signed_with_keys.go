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

package steps

import (
	"context"
	"fmt"
	"time"

	"github.com/cucumber/godog"

	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
)

func OraclesBroadcastDataSignedWithKeys(
	oracleEngine *spec.Engine,
	timesvc *stubs.TimeStub,
	rawPubKeys string,
	rawProperties *godog.Table,
) error {
	pubKeys := parseOracleDataSigners(rawPubKeys)
	pubKeysSigners := make([]*dstypes.Signer, len(pubKeys))
	for i, s := range pubKeys {
		pubKeysSigners[i] = dstypes.CreateSignerFromString(s, dstypes.SignerTypePubKey)
	}

	properties, row, err := parseOracleDataProperties(rawProperties)
	if err != nil {
		return err
	}
	meta := map[string]string{
		"eth-block-time": row.metaTimeSeconds(timesvc),
	}

	// we need a traceID here in case of final MTM settlement -> an idgen is required
	ctx := vgcontext.WithTraceID(context.Background(), "deadbeef")
	return oracleEngine.BroadcastData(ctx, dstypes.Data{
		Signers:  pubKeysSigners,
		Data:     properties,
		MetaData: meta,
	})
}

func OraclesBroadcastDataWithBlockTimeSignedWithKeys(
	oracleEngine *spec.Engine,
	timesvc *stubs.TimeStub,
	rawPubKeys string,
	rawProperties *godog.Table,
) error {
	pubKeys := parseOracleDataSigners(rawPubKeys)
	pubKeysSigners := make([]*dstypes.Signer, len(pubKeys))
	for i, s := range pubKeys {
		pubKeysSigners[i] = dstypes.CreateSignerFromString(s, dstypes.SignerTypePubKey)
	}

	rows := parseOracleTimedBroadcastTable(rawProperties)
	// we need a traceID here in case of final MTM settlement -> an idgen is required
	ctx := vgcontext.WithTraceID(context.Background(), "deadbeef")
	ordered := []string{}
	data := map[string]dstypes.Data{}
	for _, r := range rows {
		row := oracleDataPropertyRow{row: r}
		time := row.metaTime(timesvc)
		props, ok := data[time]
		if !ok {
			ordered = append(ordered, time)
			props = dstypes.Data{
				Signers: pubKeysSigners,
				Data:    map[string]string{},
				MetaData: map[string]string{
					"eth-block-time": row.metaTimeSeconds(timesvc),
				},
			}
		}
		if _, ok := props.Data[row.name()]; ok {
			return errPropertyRedeclared(row.name())
		}
		props.Data[row.name()] = row.value()
		data[time] = props
	}
	for _, k := range ordered {
		if err := oracleEngine.BroadcastData(ctx, data[k]); err != nil {
			return err
		}
	}
	return nil
}

func parseOracleDataSigners(rawPubKeys string) []string {
	return StrSlice(rawPubKeys, ",")
}

func parseOracleDataProperties(table *godog.Table) (map[string]string, oracleDataPropertyRow, error) {
	properties := map[string]string{}

	row := oracleDataPropertyRow{}
	for _, r := range parseOracleBroadcastTable(table) {
		row.row = r
		_, ok := properties[row.name()]
		if ok {
			return nil, row, errPropertyRedeclared(row.name())
		}
		properties[row.name()] = row.value()
	}

	return properties, row, nil
}

func errPropertyRedeclared(name string) error {
	return fmt.Errorf("property %s has been declared multiple times", name)
}

func parseOracleTimedBroadcastTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"name",
		"value",
		"time offset",
	}, []string{
		"eth-block-time",
	})
}

func parseOracleBroadcastTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"name",
		"value",
	}, []string{
		"eth-block-time",
	})
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

func (r oracleDataPropertyRow) timeOffset() time.Duration {
	if r.row.HasColumn("time offset") {
		return r.row.MustDurationStr("time offset")
	}
	return 0
}

func (r oracleDataPropertyRow) metaTime(timeSvc *stubs.TimeStub) string {
	if r.row.HasColumn("eth-block-time") {
		return r.row.MustStr("eth-block-time")
	}
	tm := timeSvc.GetTimeNow().Add(r.timeOffset())
	return fmt.Sprintf("%d", tm.UnixNano())
}

func (r oracleDataPropertyRow) metaTimeSeconds(timeSvc *stubs.TimeStub) string {
	if r.row.HasColumn("eth-block-time") {
		return r.row.MustStr("eth-block-time")
	}
	tm := timeSvc.GetTimeNow().Add(r.timeOffset())
	return fmt.Sprintf("%d", tm.Unix())
}
