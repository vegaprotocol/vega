// Code generated by MockGen. DO NOT EDIT.
// Source: code.vegaprotocol.io/vega/core/execution/liquidation (interfaces: Book,MarketLiquidity,IDGen,Positions,Settlement)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	events "code.vegaprotocol.io/vega/core/events"
	positions "code.vegaprotocol.io/vega/core/positions"
	types "code.vegaprotocol.io/vega/core/types"
	num "code.vegaprotocol.io/vega/libs/num"
	vega "code.vegaprotocol.io/vega/protos/vega"
	gomock "github.com/golang/mock/gomock"
)

// MockBook is a mock of Book interface.
type MockBook struct {
	ctrl     *gomock.Controller
	recorder *MockBookMockRecorder
}

// MockBookMockRecorder is the mock recorder for MockBook.
type MockBookMockRecorder struct {
	mock *MockBook
}

// NewMockBook creates a new mock instance.
func NewMockBook(ctrl *gomock.Controller) *MockBook {
	mock := &MockBook{ctrl: ctrl}
	mock.recorder = &MockBookMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBook) EXPECT() *MockBookMockRecorder {
	return m.recorder
}

// GetVolumeAtPrice mocks base method.
func (m *MockBook) GetVolumeAtPrice(arg0 *num.Uint, arg1 vega.Side) uint64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVolumeAtPrice", arg0, arg1)
	ret0, _ := ret[0].(uint64)
	return ret0
}

// GetVolumeAtPrice indicates an expected call of GetVolumeAtPrice.
func (mr *MockBookMockRecorder) GetVolumeAtPrice(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVolumeAtPrice", reflect.TypeOf((*MockBook)(nil).GetVolumeAtPrice), arg0, arg1)
}

// MockMarketLiquidity is a mock of MarketLiquidity interface.
type MockMarketLiquidity struct {
	ctrl     *gomock.Controller
	recorder *MockMarketLiquidityMockRecorder
}

// MockMarketLiquidityMockRecorder is the mock recorder for MockMarketLiquidity.
type MockMarketLiquidityMockRecorder struct {
	mock *MockMarketLiquidity
}

// NewMockMarketLiquidity creates a new mock instance.
func NewMockMarketLiquidity(ctrl *gomock.Controller) *MockMarketLiquidity {
	mock := &MockMarketLiquidity{ctrl: ctrl}
	mock.recorder = &MockMarketLiquidityMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMarketLiquidity) EXPECT() *MockMarketLiquidityMockRecorder {
	return m.recorder
}

// ValidOrdersPriceRange mocks base method.
func (m *MockMarketLiquidity) ValidOrdersPriceRange() (*num.Uint, *num.Uint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidOrdersPriceRange")
	ret0, _ := ret[0].(*num.Uint)
	ret1, _ := ret[1].(*num.Uint)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// ValidOrdersPriceRange indicates an expected call of ValidOrdersPriceRange.
func (mr *MockMarketLiquidityMockRecorder) ValidOrdersPriceRange() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidOrdersPriceRange", reflect.TypeOf((*MockMarketLiquidity)(nil).ValidOrdersPriceRange))
}

// MockIDGen is a mock of IDGen interface.
type MockIDGen struct {
	ctrl     *gomock.Controller
	recorder *MockIDGenMockRecorder
}

// MockIDGenMockRecorder is the mock recorder for MockIDGen.
type MockIDGenMockRecorder struct {
	mock *MockIDGen
}

// NewMockIDGen creates a new mock instance.
func NewMockIDGen(ctrl *gomock.Controller) *MockIDGen {
	mock := &MockIDGen{ctrl: ctrl}
	mock.recorder = &MockIDGenMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIDGen) EXPECT() *MockIDGenMockRecorder {
	return m.recorder
}

// NextID mocks base method.
func (m *MockIDGen) NextID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NextID")
	ret0, _ := ret[0].(string)
	return ret0
}

// NextID indicates an expected call of NextID.
func (mr *MockIDGenMockRecorder) NextID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NextID", reflect.TypeOf((*MockIDGen)(nil).NextID))
}

// MockPositions is a mock of Positions interface.
type MockPositions struct {
	ctrl     *gomock.Controller
	recorder *MockPositionsMockRecorder
}

// MockPositionsMockRecorder is the mock recorder for MockPositions.
type MockPositionsMockRecorder struct {
	mock *MockPositions
}

// NewMockPositions creates a new mock instance.
func NewMockPositions(ctrl *gomock.Controller) *MockPositions {
	mock := &MockPositions{ctrl: ctrl}
	mock.recorder = &MockPositionsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPositions) EXPECT() *MockPositionsMockRecorder {
	return m.recorder
}

// RegisterOrder mocks base method.
func (m *MockPositions) RegisterOrder(arg0 context.Context, arg1 *types.Order) *positions.MarketPosition {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RegisterOrder", arg0, arg1)
	ret0, _ := ret[0].(*positions.MarketPosition)
	return ret0
}

// RegisterOrder indicates an expected call of RegisterOrder.
func (mr *MockPositionsMockRecorder) RegisterOrder(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RegisterOrder", reflect.TypeOf((*MockPositions)(nil).RegisterOrder), arg0, arg1)
}

// Update mocks base method.
func (m *MockPositions) Update(arg0 context.Context, arg1 *types.Trade, arg2, arg3 *types.Order) []events.MarketPosition {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].([]events.MarketPosition)
	return ret0
}

// Update indicates an expected call of Update.
func (mr *MockPositionsMockRecorder) Update(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockPositions)(nil).Update), arg0, arg1, arg2, arg3)
}

// MockSettlement is a mock of Settlement interface.
type MockSettlement struct {
	ctrl     *gomock.Controller
	recorder *MockSettlementMockRecorder
}

// MockSettlementMockRecorder is the mock recorder for MockSettlement.
type MockSettlementMockRecorder struct {
	mock *MockSettlement
}

// NewMockSettlement creates a new mock instance.
func NewMockSettlement(ctrl *gomock.Controller) *MockSettlement {
	mock := &MockSettlement{ctrl: ctrl}
	mock.recorder = &MockSettlementMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSettlement) EXPECT() *MockSettlementMockRecorder {
	return m.recorder
}

// AddTrade mocks base method.
func (m *MockSettlement) AddTrade(arg0 *types.Trade) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddTrade", arg0)
}

// AddTrade indicates an expected call of AddTrade.
func (mr *MockSettlementMockRecorder) AddTrade(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddTrade", reflect.TypeOf((*MockSettlement)(nil).AddTrade), arg0)
}