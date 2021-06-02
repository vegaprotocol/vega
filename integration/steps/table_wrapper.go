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

func GetFirstRow(table gherkin.DataTable) (RowWrapper, error) {
	rows := TableWrapper(table).Parse()

	if len(rows) > 1 {
		return RowWrapper{}, fmt.Errorf("this table supports only one row")
	}

	for _, r := range rows {
		return r, nil
	}

	return RowWrapper{}, fmt.Errorf("missing row")
}

type TableWrapper gherkin.DataTable

// StrictParse parses and verifies the table integrity.
func (t TableWrapper) StrictParse(columns ...string) []RowWrapper {
	dt := gherkin.DataTable(t)

	tableLen := len(dt.Rows)
	if tableLen < 1 {
		panic("A table is required.")
	}

	verifyTableIntegrity(columns, dt.Rows[0])

	out := make([]RowWrapper, 0, tableLen-1)
	for _, row := range dt.Rows[1:] {
		wrapper := RowWrapper{values: map[string]string{}}
		for i := range row.Cells {
			wrapper.values[dt.Rows[0].Cells[i].Value] = row.Cells[i].Value
		}
		out = append(out, wrapper)
	}

	return out
}

// Parse parses the table without verifying the integrity.
func (t TableWrapper) Parse() []RowWrapper {
	return t.StrictParse()
}

// verifyTableIntegrity ensures the table declares the expected columns and does
// not declared any unexpected columns.
func verifyTableIntegrity(columns []string, header *gherkin.TableRow) {
	if len(columns) == 0 {
		return
	}

	requiredColumnsSet := map[string]interface{}{}
	for _, column := range columns {
		requiredColumnsSet[column] = nil
	}

	declaredColumnsSet := map[string]interface{}{}
	for _, cell := range header.Cells {
		_, ok := requiredColumnsSet[cell.Value]
		if !ok {
			panic(fmt.Errorf("the column \"%s\" is not expected by this table", cell.Value))
		}
		declaredColumnsSet[cell.Value] = nil
	}

	for requiredColumn := range requiredColumnsSet {
		_, ok := declaredColumnsSet[requiredColumn]
		if !ok {
			panic(fmt.Errorf("a column \"%s\" is required by this table", requiredColumn))
		}
	}
}

type RowWrapper struct {
	values map[string]string
}

func (r RowWrapper) mustColumn(name string) string {
	s, ok := r.values[name]
	if !ok {
		panic(fmt.Errorf("column \"%s\" not found", name))
	}
	return s
}

// Has returns whether or not a given column was specified
func (r RowWrapper) Has(col string) bool {
	_, ok := r.values[col]
	return ok
}

func (r RowWrapper) MustStr(name string) string {
	return r.mustColumn(name)
}

// StrB simply returns the string value, but includes the bool indicating whether or not the column was set
func (r RowWrapper) StrB(name string) (string, bool) {
	s, ok := r.values[name]
	// empty strings don't count - this would mess things up with multi-line checks (e.g. price monitoring in market data)
	if ok && s == "" {
		return "", false
	}
	return s, ok
}

func (r RowWrapper) Str(name string) string {
	return r.values[name]
}

func (r RowWrapper) MustStrSlice(name, sep string) []string {
	return StrSlice(r.mustColumn(name), sep)
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

func (r RowWrapper) MustU64(name string) uint64 {
	value, err := U64(r.mustColumn(name))
	panicW(name, err)
	return value
}

// U64B does the same as U64, but returns a bool indicating whether or not an explicit 0 was set
// or the column simply doesn't exist
func (r RowWrapper) U64B(name string) (uint64, bool) {
	if v, ok := r.values[name]; !ok || v == "" {
		return 0, false
	}
	v := r.U64(name)
	return v, true
}

func (r RowWrapper) U64(name string) uint64 {
	value, err := U64(r.values[name])
	panicW(name, err)
	return value
}

func U64(value string) (uint64, error) {
	return strconv.ParseUint(value, 10, 0)
}

func (r RowWrapper) MustU32(name string) uint32 {
	value, err := U64(r.mustColumn(name))
	panicW(name, err)
	return uint32(value)
}

func (r RowWrapper) U32(name string) uint32 {
	value, err := U64(r.values[name])
	panicW(name, err)
	return uint32(value)
}

func (r RowWrapper) MustU64Slice(name, sep string) []uint64 {
	value, err := U64Slice(r.mustColumn(name), sep)
	panicW(name, err)
	return value
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

func (r RowWrapper) MustI64(name string) int64 {
	value, err := I64(r.mustColumn(name))
	panicW(name, err)
	return value
}

// I64B does the same as U64B (ie same as I64, but returns a bool for empty/missing columns)
func (r RowWrapper) I64B(name string) (int64, bool) {
	if v, ok := r.values[name]; !ok || v == "" {
		return 0, false
	}
	v := r.I64(name)
	return v, true
}

func (r RowWrapper) I64(name string) int64 {
	value, err := I64(r.values[name])
	panicW(name, err)
	return value
}

func I64(rawValue string) (int64, error) {
	return strconv.ParseInt(rawValue, 10, 0)
}

func (r RowWrapper) MustI64Slice(name, sep string) []int64 {
	value, err := I64Slice(r.mustColumn(name), sep)
	panicW(name, err)
	return value
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

func (r RowWrapper) MustF64(name string) float64 {
	value, err := F64(r.mustColumn(name))
	panicW(name, err)
	return value
}

func (r RowWrapper) F64(name string) float64 {
	value, err := F64(r.values[name])
	panicW(name, err)
	return value
}

func F64(rawValue string) (float64, error) {
	return strconv.ParseFloat(rawValue, 10)
}

func (r RowWrapper) MustF64Slice(name, sep string) []float64 {
	value, err := F64Slice(r.mustColumn(name), sep)
	panicW(name, err)
	return value
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

func (r RowWrapper) MustBool(name string) bool {
	b, err := Bool(r.mustColumn(name))
	panicW(name, err)
	return b
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

func (r RowWrapper) MustTime(name string) time.Time {
	t, err := Time(r.mustColumn(name))
	panicW(name, err)
	return t
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

func (r RowWrapper) MustOrderType(name string) types.Order_Type {
	orderType, err := OrderType(r.MustStr(name))
	panicW(name, err)
	return orderType
}

func (r RowWrapper) OrderType(name string) types.Order_Type {
	orderType, err := OrderType(r.Str(name))
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

func (r RowWrapper) MustOrderStatus(name string) types.Order_Status {
	s, err := OrderStatus(r.MustStr(name))
	panicW(name, err)
	return s
}

func (r RowWrapper) OrderStatus(name string) types.Order_Status {
	s, err := OrderStatus(r.Str(name))
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

func (r RowWrapper) MustLiquidityStatus(name string) types.LiquidityProvision_Status {
	s, err := LiquidityStatus(r.MustStr(name))
	panicW(name, err)
	return s
}

func (r RowWrapper) LiquidityStatus(name string) types.LiquidityProvision_Status {
	s, err := LiquidityStatus(r.Str(name))
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

func (r RowWrapper) MustTIF(name string) types.Order_TimeInForce {
	tif, err := TIF(r.MustStr(name))
	panicW(name, err)
	return tif
}

func (r RowWrapper) TIF(name string) types.Order_TimeInForce {
	tif, err := TIF(r.Str(name))
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

func (r RowWrapper) MustSide(name string) types.Side {
	side, err := Side(r.MustStr(name))
	panicW(name, err)
	return side
}

func (r RowWrapper) Side(name string) types.Side {
	side, err := Side(r.Str(name))
	panicW(name, err)
	return side
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

func (r RowWrapper) MustPeggedReference(name string) types.PeggedReference {
	return peggedReference(r.MustStr(name))
}

func (r RowWrapper) PeggedReference(name string) types.PeggedReference {
	return peggedReference(r.Str(name))
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

func (r RowWrapper) MustOracleSpecPropertyType(name string) oraclesv1.PropertyKey_Type {
	ty, err := OracleSpecPropertyType(r.MustStr(name))
	panicW(name, err)
	return ty
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

func (r RowWrapper) MustAuctionTrigger(name string) types.AuctionTrigger {
	at, err := AuctionTrigger(r.MustStr(name))
	panicW(name, err)
	return at
}

func (r RowWrapper) AuctionTrigger(name string) types.AuctionTrigger {
	at, err := AuctionTrigger(r.Str(name))
	panicW(name, err)
	return at
}

func AuctionTrigger(name string) (types.AuctionTrigger, error) {
	at, ok := types.AuctionTrigger_value[name]
	if !ok {
		return types.AuctionTrigger_AUCTION_TRIGGER_UNSPECIFIED, fmt.Errorf("couldn't find %s as auction trigger", name)
	}
	return types.AuctionTrigger(at), nil
}

func (r RowWrapper) MustTradingMode(name string) types.Market_TradingMode {
	ty, err := TradingMode(r.MustStr(name))
	panicW(name, err)
	return ty
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

func (r RowWrapper) MustAccount(name string) types.AccountType {
	return account(r.MustStr(name))
}

func (r RowWrapper) Account(name string) types.AccountType {
	return account(r.Str(name))
}

func (r RowWrapper) MustPrice(name string) *types.Price {
	n := r.MustU64(name)
	// nil instead of zero value of Price is expected by APIs
	if n == 0 {
		return nil
	}
	return Price(n)
}
func (r RowWrapper) Price(name string) *types.Price {
	n := r.U64(name)
	// nil instead of zero value of Price is expected by APIs
	if n == 0 {
		return nil
	}
	return Price(n)
}

func (r RowWrapper) MustDuration(name string) time.Duration {
	return time.Duration(r.MustU64(name))
}

func (r RowWrapper) MustDurationStr(name string) time.Duration {
	s := r.MustStr(name)
	d, err := time.ParseDuration(s)
	panicW(name, err)
	return d
}

func (r RowWrapper) Duration(name string) time.Duration {
	return time.Duration(r.U64(name))
}

func (r RowWrapper) MustDurationSec(name string) time.Duration {
	n := r.MustU64(name)
	if n == 0 {
		return 0
	}
	return time.Duration(n) * time.Second
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
