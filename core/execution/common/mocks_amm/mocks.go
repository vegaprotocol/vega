// Code generated by MockGen. DO NOT EDIT.
// Source: code.vegaprotocol.io/vega/core/execution/common (interfaces: AMMPool,AMM)

// Package mocks_amm is a generated GoMock package.
package mocks_amm

import (
	reflect "reflect"

	common "code.vegaprotocol.io/vega/core/execution/common"
	types "code.vegaprotocol.io/vega/core/types"
	num "code.vegaprotocol.io/vega/libs/num"
	gomock "github.com/golang/mock/gomock"
)

// MockAMMPool is a mock of AMMPool interface.
type MockAMMPool struct {
	ctrl     *gomock.Controller
	recorder *MockAMMPoolMockRecorder
}

// MockAMMPoolMockRecorder is the mock recorder for MockAMMPool.
type MockAMMPoolMockRecorder struct {
	mock *MockAMMPool
}

// NewMockAMMPool creates a new mock instance.
func NewMockAMMPool(ctrl *gomock.Controller) *MockAMMPool {
	mock := &MockAMMPool{ctrl: ctrl}
	mock.recorder = &MockAMMPoolMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAMMPool) EXPECT() *MockAMMPoolMockRecorder {
	return m.recorder
}

// OrderbookShape mocks base method.
func (m *MockAMMPool) OrderbookShape(arg0, arg1 *num.Uint) ([]*types.Order, []*types.Order) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "OrderbookShape", arg0, arg1)
	ret0, _ := ret[0].([]*types.Order)
	ret1, _ := ret[1].([]*types.Order)
	return ret0, ret1
}

// OrderbookShape indicates an expected call of OrderbookShape.
func (mr *MockAMMPoolMockRecorder) OrderbookShape(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OrderbookShape", reflect.TypeOf((*MockAMMPool)(nil).OrderbookShape), arg0, arg1)
}

// MockAMM is a mock of AMM interface.
type MockAMM struct {
	ctrl     *gomock.Controller
	recorder *MockAMMMockRecorder
}

// MockAMMMockRecorder is the mock recorder for MockAMM.
type MockAMMMockRecorder struct {
	mock *MockAMM
}

// NewMockAMM creates a new mock instance.
func NewMockAMM(ctrl *gomock.Controller) *MockAMM {
	mock := &MockAMM{ctrl: ctrl}
	mock.recorder = &MockAMMMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAMM) EXPECT() *MockAMMMockRecorder {
	return m.recorder
}

// GetAMMPoolsBySubAccount mocks base method.
func (m *MockAMM) GetAMMPoolsBySubAccount() map[string]common.AMMPool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAMMPoolsBySubAccount")
	ret0, _ := ret[0].(map[string]common.AMMPool)
	return ret0
}

// GetAMMPoolsBySubAccount indicates an expected call of GetAMMPoolsBySubAccount.
func (mr *MockAMMMockRecorder) GetAMMPoolsBySubAccount() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAMMPoolsBySubAccount", reflect.TypeOf((*MockAMM)(nil).GetAMMPoolsBySubAccount))
}