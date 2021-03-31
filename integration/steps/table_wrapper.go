package steps

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	types "code.vegaprotocol.io/vega/proto"
	oraclesv1 "code.vegaprotocol.io/vega/proto/oracles/v1"

	"github.com/cucumber/godog/gherkin"
)

type TableWrapper gherkin.DataTable

func (t TableWrapper) Parse() []RowWrapper {
	dt := gherkin.DataTable(t)
	out := make([]RowWrapper, 0, len(dt.Rows)-1)

	for _, row := range dt.Rows[1:] {
		wrapper := RowWrapper{values: map[string]string{}}
		for i := range row.Cells {
			wrapper.values[dt.Rows[0].Cells[i].Value] = row.Cells[i].Value
		}
		out = append(out, wrapper)
	}

	return out
}

type RowWrapper struct {
	values map[string]string
}

func (r RowWrapper) Str(name string) string {
	return r.values[name]
}

func (r RowWrapper) StrSlice(name, sep string) []string {
	return StrSlice(r.values[name], sep)
}

func StrSlice(value string, sep string) []string {
	if len(value) == 0 {
		return nil
	}
	return strings.Split(value, sep)
}

func (r RowWrapper) U64(name string) uint64 {
	value, err := U64(r.values[name])
	panicW(name, err)
	return value
}

func U64(value string) (uint64, error) {
	return strconv.ParseUint(value, 10, 0)
}

func (r RowWrapper) U32(name string) uint32 {
	value, err := U64(r.values[name])
	panicW(name, err)
	return uint32(value)
}

func (r RowWrapper) U64Slice(name, sep string) []uint64 {
	value, err := U64Slice(r.values[name], sep)
	panicW(name, err)
	return value
}

func U64Slice(rawValue, sep string) ([]uint64, error) {
	if len(rawValue) == 0 {
		return []uint64{}, nil
	}
	rawValues := strings.Split(rawValue, sep)
	valuesCount := len(rawValues)
	array := make([]uint64, 0, valuesCount)
	for i := 0; i < valuesCount; i++ {
		item, err := strconv.ParseUint(rawValues[i], 10, 0)
		if err != nil {
			return nil, err
		}
		array = append(array, item)
	}
	return array, nil
}

func (r RowWrapper) I64(name string) int64 {
	value, err := I64(r.values[name])
	panicW(name, err)
	return value
}

func I64(rawValue string) (int64, error) {
	return strconv.ParseInt(rawValue, 10, 0)
}

func (r RowWrapper) I64Slice(name, sep string) []int64 {
	value, err := I64Slice(r.values[name], sep)
	panicW(name, err)
	return value
}

func I64Slice(rawValue string, sep string) ([]int64, error) {
	if len(rawValue) == 0 {
		return []int64{}, nil
	}
	rawValues := strings.Split(rawValue, sep)
	valuesCount := len(rawValues)
	array := make([]int64, 0, valuesCount)
	for i := 0; i < valuesCount; i++ {
		item, err := strconv.ParseInt(rawValues[i], 10, 0)
		if err != nil {
			return nil, err
		}
		array = append(array, item)
	}
	return array, nil
}

func (r RowWrapper) F64(name string) float64 {
	value, err := F64(r.values[name])
	panicW(name, err)
	return value
}

func F64(rawValue string) (float64, error) {
	return strconv.ParseFloat(rawValue, 10)
}

func (r RowWrapper) F64Slice(name, sep string) []float64 {
	value, err := F64Slice(r.values[name], sep)
	panicW(name, err)
	return value
}

func F64Slice(rawValue string, sep string) ([]float64, error) {
	if len(rawValue) == 0 {
		return nil, nil
	}
	rawValues := strings.Split(rawValue, sep)
	valuesCount := len(rawValues)
	array := make([]float64, 0, valuesCount)
	for i := 0; i < valuesCount; i++ {
		item, err := strconv.ParseFloat(rawValues[i], 10)
		if err != nil {
			return nil, err
		}
		array = append(array, item)
	}
	return array, nil
}

func (r RowWrapper) Bool(name string) bool {
	b, err := Bool(r.values[name])
	panicW(name, err)
	return b
}

func Bool(rawValue string) (bool, error) {
	if rawValue == "true" {
		return true, nil
	} else if rawValue == "false" {
		return false, nil
	}
	return false, fmt.Errorf("invalid bool value: %v", rawValue)
}

func (r RowWrapper) Time(name string) time.Time {
	t, err := Time(r.values[name])
	panicW(name, err)
	return t
}

func Time(rawTime string) (time.Time, error) {
	parsedTime, err := time.Parse("2006-01-02T15:04:05Z", rawTime)
	if err != nil {
		return parsedTime, fmt.Errorf("invalid date value: %v", err)
	}
	return parsedTime, nil
}

func (r RowWrapper) OrderType(name string) types.Order_Type {
	orderType, err := OrderType(r.values[name])
	panicW(name, err)
	return orderType
}

func OrderType(rawValue string) (types.Order_Type, error) {
	ty, ok := types.Order_Type_value[rawValue]
	if !ok {
		return types.Order_Type(ty), fmt.Errorf("invalid order type: %v", rawValue)
	}
	return types.Order_Type(ty), nil
}

func (r RowWrapper) OrderStatus(name string) types.Order_Status {
	s, err := OrderStatus(r.values[name])
	panicW(name, err)
	return s
}

func OrderStatus(rawValue string) (types.Order_Status, error) {
	ty, ok := types.Order_Status_value[rawValue]
	if !ok {
		return types.Order_Status(ty), fmt.Errorf("invalid order status: %v", rawValue)
	}
	return types.Order_Status(ty), nil
}

func (r RowWrapper) LiquidityStatus(name string) types.LiquidityProvision_Status {
	s, err := LiquidityStatus(r.values[name])
	panicW(name, err)
	return s
}

func LiquidityStatus(rawValue string) (types.LiquidityProvision_Status, error) {
	ty, ok := types.LiquidityProvision_Status_value[rawValue]
	if !ok {
		return types.LiquidityProvision_Status(ty), fmt.Errorf("invalid liquidity provision status: %v", rawValue)
	}
	return types.LiquidityProvision_Status(ty), nil
}

func (r RowWrapper) TIF(name string) types.Order_TimeInForce {
	tif, err := TIF(r.values[name])
	panicW(name, err)
	return tif
}

func TIF(rawValue string) (types.Order_TimeInForce, error) {
	tif, ok := types.Order_TimeInForce_value[strings.ReplaceAll(rawValue, "TIF_", "TIME_IN_FORCE_")]
	if !ok {
		return types.Order_TimeInForce(tif), fmt.Errorf("invalid time in force: %v", rawValue)
	}
	return types.Order_TimeInForce(tif), nil
}

func (r RowWrapper) Side(name string) types.Side {
	side, err := Side(r.values[name])
	panicW(name, err)
	return side
}

func (r RowWrapper) PeggedReference(name string) types.PeggedReference {
	return peggedReference(r.values[name])
}

func peggedReference(rawValue string) types.PeggedReference {
	switch rawValue {
	case "MID":
		return types.PeggedReference_PEGGED_REFERENCE_MID
	case "ASK":
		return types.PeggedReference_PEGGED_REFERENCE_BEST_ASK
	case "BID":
		return types.PeggedReference_PEGGED_REFERENCE_BEST_BID
	}
	return types.PeggedReference_PEGGED_REFERENCE_UNSPECIFIED
}

func Side(rawValue string) (types.Side, error) {
	switch rawValue {
	case "sell":
		return types.Side_SIDE_SELL, nil
	case "buy":
		return types.Side_SIDE_BUY, nil
	default:
		return types.Side_SIDE_UNSPECIFIED, errors.New("invalid side")
	}
}

func (r RowWrapper) OracleSpecPropertyType(name string) oraclesv1.PropertyKey_Type {
	ty, err := OracleSpecPropertyType(r.Str(name))
	panicW(name, err)
	return ty
}

func OracleSpecPropertyType(name string) (oraclesv1.PropertyKey_Type, error) {
	ty, ok := oraclesv1.PropertyKey_Type_value[name]

	if !ok {
		return oraclesv1.PropertyKey_TYPE_UNSPECIFIED, fmt.Errorf("couldn't find %s as property type", name)
	}
	return oraclesv1.PropertyKey_Type(ty), nil
}

func (r RowWrapper) TradingMode(name string) types.Market_TradingMode {
	ty, err := TradingMode(r.Str(name))
	panicW(name, err)
	return ty
}

func TradingMode(name string) (types.Market_TradingMode, error) {
	ty, ok := types.Market_TradingMode_value[name]

	if !ok {
		return types.Market_TRADING_MODE_UNSPECIFIED, fmt.Errorf("couldn't find %s as trading_mode", name)
	}
	return types.Market_TradingMode(ty), nil
}

func (r RowWrapper) Account(name string) types.AccountType {
	return account(r.Str(name))
}

func (r RowWrapper) Price(name string) *types.Price {
	n := r.U64(name)
	// nil instead of zero value of Price is expected by APIs
	if n == 0 {
		return nil
	}
	return Price(n)
}

func (r RowWrapper) Duration(name string) time.Duration {
	return time.Duration(r.U64(name))
}

func (r RowWrapper) DurationSec(name string) time.Duration {
	n := r.U64(name)
	if n == 0 {
		return 0
	}
	return time.Duration(n) * time.Second
}

func Price(n uint64) *types.Price {
	return &types.Price{Value: n}
}

func account(name string) types.AccountType {
	value := types.AccountType(types.AccountType_value[name])

	if value == types.AccountType_ACCOUNT_TYPE_UNSPECIFIED {
		panic(fmt.Sprintf("invalid account type %s", name))
	}

	return value
}

func accountID(marketID, partyID, asset string, ty types.AccountType) string {
	idBuf := make([]byte, 256)

	if ty == types.AccountType_ACCOUNT_TYPE_GENERAL {
		marketID = ""
	}

	if partyID == "market" {
		partyID = ""
	}

	if len(marketID) == 0 {
		marketID = "!"
	}

	if len(partyID) == 0 {
		partyID = "*"
	}

	copy(idBuf, marketID)
	ln := len(marketID)
	copy(idBuf[ln:], partyID)
	ln += len(partyID)
	copy(idBuf[ln:], asset)
	ln += len(asset)
	idBuf[ln] = byte(ty + 48)
	return string(idBuf[:ln+1])
}

func panicW(field string, err error) {
	if err != nil {
		panic(fmt.Sprintf("couldn't parse %s: %v", field, err))
	}
}
