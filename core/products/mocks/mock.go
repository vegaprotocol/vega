// Code generated by MockGen. DO NOT EDIT.
// Source: code.vegaprotocol.io/vega/core/products (interfaces: OracleEngine,Broker)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	common "code.vegaprotocol.io/vega/core/datasource/common"
	spec "code.vegaprotocol.io/vega/core/datasource/spec"
	events "code.vegaprotocol.io/vega/core/events"
	gomock "github.com/golang/mock/gomock"
)

// MockOracleEngine is a mock of OracleEngine interface.
type MockOracleEngine struct {
	ctrl     *gomock.Controller
	recorder *MockOracleEngineMockRecorder
}

// MockOracleEngineMockRecorder is the mock recorder for MockOracleEngine.
type MockOracleEngineMockRecorder struct {
	mock *MockOracleEngine
}

// NewMockOracleEngine creates a new mock instance.
func NewMockOracleEngine(ctrl *gomock.Controller) *MockOracleEngine {
	mock := &MockOracleEngine{ctrl: ctrl}
	mock.recorder = &MockOracleEngineMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOracleEngine) EXPECT() *MockOracleEngineMockRecorder {
	return m.recorder
}

// ListensToSigners mocks base method.
func (m *MockOracleEngine) ListensToSigners(arg0 common.Data) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListensToSigners", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// ListensToSigners indicates an expected call of ListensToSigners.
func (mr *MockOracleEngineMockRecorder) ListensToSigners(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListensToSigners", reflect.TypeOf((*MockOracleEngine)(nil).ListensToSigners), arg0)
}

// Subscribe mocks base method.
func (m *MockOracleEngine) Subscribe(arg0 context.Context, arg1 spec.Spec, arg2 spec.OnMatchedData) (spec.SubscriptionID, spec.Unsubscriber, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Subscribe", arg0, arg1, arg2)
	ret0, _ := ret[0].(spec.SubscriptionID)
	ret1, _ := ret[1].(spec.Unsubscriber)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Subscribe indicates an expected call of Subscribe.
func (mr *MockOracleEngineMockRecorder) Subscribe(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Subscribe", reflect.TypeOf((*MockOracleEngine)(nil).Subscribe), arg0, arg1, arg2)
}

// Unsubscribe mocks base method.
func (m *MockOracleEngine) Unsubscribe(arg0 context.Context, arg1 spec.SubscriptionID) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Unsubscribe", arg0, arg1)
}

// Unsubscribe indicates an expected call of Unsubscribe.
func (mr *MockOracleEngineMockRecorder) Unsubscribe(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Unsubscribe", reflect.TypeOf((*MockOracleEngine)(nil).Unsubscribe), arg0, arg1)
}

// MockBroker is a mock of Broker interface.
type MockBroker struct {
	ctrl     *gomock.Controller
	recorder *MockBrokerMockRecorder
}

// MockBrokerMockRecorder is the mock recorder for MockBroker.
type MockBrokerMockRecorder struct {
	mock *MockBroker
}

// NewMockBroker creates a new mock instance.
func NewMockBroker(ctrl *gomock.Controller) *MockBroker {
	mock := &MockBroker{ctrl: ctrl}
	mock.recorder = &MockBrokerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBroker) EXPECT() *MockBrokerMockRecorder {
	return m.recorder
}

// Send mocks base method.
func (m *MockBroker) Send(arg0 events.Event) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Send", arg0)
}

// Send indicates an expected call of Send.
func (mr *MockBrokerMockRecorder) Send(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockBroker)(nil).Send), arg0)
}

// SendBatch mocks base method.
func (m *MockBroker) SendBatch(arg0 []events.Event) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SendBatch", arg0)
}

// SendBatch indicates an expected call of SendBatch.
func (mr *MockBrokerMockRecorder) SendBatch(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendBatch", reflect.TypeOf((*MockBroker)(nil).SendBatch), arg0)
}
