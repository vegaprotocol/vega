// Code generated by MockGen. DO NOT EDIT.
// Source: code.vegaprotocol.io/vega/datanode/api (interfaces: AssetService)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	entities "code.vegaprotocol.io/vega/datanode/entities"
	gomock "github.com/golang/mock/gomock"
)

// MockAssetService is a mock of AssetService interface.
type MockAssetService struct {
	ctrl     *gomock.Controller
	recorder *MockAssetServiceMockRecorder
}

// MockAssetServiceMockRecorder is the mock recorder for MockAssetService.
type MockAssetServiceMockRecorder struct {
	mock *MockAssetService
}

// NewMockAssetService creates a new mock instance.
func NewMockAssetService(ctrl *gomock.Controller) *MockAssetService {
	mock := &MockAssetService{ctrl: ctrl}
	mock.recorder = &MockAssetServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAssetService) EXPECT() *MockAssetServiceMockRecorder {
	return m.recorder
}

// GetAll mocks base method.
func (m *MockAssetService) GetAll(arg0 context.Context) ([]entities.Asset, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAll", arg0)
	ret0, _ := ret[0].([]entities.Asset)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAll indicates an expected call of GetAll.
func (mr *MockAssetServiceMockRecorder) GetAll(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAll", reflect.TypeOf((*MockAssetService)(nil).GetAll), arg0)
}

// GetAllWithCursorPagination mocks base method.
func (m *MockAssetService) GetAllWithCursorPagination(arg0 context.Context, arg1 entities.CursorPagination) ([]entities.Asset, entities.PageInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllWithCursorPagination", arg0, arg1)
	ret0, _ := ret[0].([]entities.Asset)
	ret1, _ := ret[1].(entities.PageInfo)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetAllWithCursorPagination indicates an expected call of GetAllWithCursorPagination.
func (mr *MockAssetServiceMockRecorder) GetAllWithCursorPagination(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllWithCursorPagination", reflect.TypeOf((*MockAssetService)(nil).GetAllWithCursorPagination), arg0, arg1)
}

// GetByID mocks base method.
func (m *MockAssetService) GetByID(arg0 context.Context, arg1 string) (entities.Asset, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByID", arg0, arg1)
	ret0, _ := ret[0].(entities.Asset)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByID indicates an expected call of GetByID.
func (mr *MockAssetServiceMockRecorder) GetByID(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByID", reflect.TypeOf((*MockAssetService)(nil).GetByID), arg0, arg1)
}

// GetByTxHash mocks base method.
func (m *MockAssetService) GetByTxHash(arg0 context.Context, arg1 entities.TxHash) ([]entities.Asset, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByTxHash", arg0, arg1)
	ret0, _ := ret[0].([]entities.Asset)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByTxHash indicates an expected call of GetByTxHash.
func (mr *MockAssetServiceMockRecorder) GetByTxHash(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByTxHash", reflect.TypeOf((*MockAssetService)(nil).GetByTxHash), arg0, arg1)
}
