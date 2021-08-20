// Code generated by MockGen. DO NOT EDIT.
// Source: cookie.go

// Package main is a generated GoMock package.
package main

import (
	http "net/http"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockCookieManager is a mock of CookieManager interface.
type MockCookieManager struct {
	ctrl     *gomock.Controller
	recorder *MockCookieManagerMockRecorder
}

// MockCookieManagerMockRecorder is the mock recorder for MockCookieManager.
type MockCookieManagerMockRecorder struct {
	mock *MockCookieManager
}

// NewMockCookieManager creates a new mock instance.
func NewMockCookieManager(ctrl *gomock.Controller) *MockCookieManager {
	mock := &MockCookieManager{ctrl: ctrl}
	mock.recorder = &MockCookieManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCookieManager) EXPECT() *MockCookieManagerMockRecorder {
	return m.recorder
}

// Delete mocks base method.
func (m *MockCookieManager) Delete(sessionId string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Delete", sessionId)
}

// Delete indicates an expected call of Delete.
func (mr *MockCookieManagerMockRecorder) Delete(sessionId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockCookieManager)(nil).Delete), sessionId)
}

// Get mocks base method.
func (m *MockCookieManager) Get(sessionId string) http.CookieJar {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", sessionId)
	ret0, _ := ret[0].(http.CookieJar)
	return ret0
}

// Get indicates an expected call of Get.
func (mr *MockCookieManagerMockRecorder) Get(sessionId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockCookieManager)(nil).Get), sessionId)
}

// NewSession mocks base method.
func (m *MockCookieManager) NewSession(sessionId string) http.CookieJar {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewSession", sessionId)
	ret0, _ := ret[0].(http.CookieJar)
	return ret0
}

// NewSession indicates an expected call of NewSession.
func (mr *MockCookieManagerMockRecorder) NewSession(sessionId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewSession", reflect.TypeOf((*MockCookieManager)(nil).NewSession), sessionId)
}