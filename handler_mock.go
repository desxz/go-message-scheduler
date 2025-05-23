// Code generated by MockGen. DO NOT EDIT.
// Source: handler.go
//
// Generated by this command:
//
//	mockgen --source=handler.go --destination=handler_mock.go --package=main
//

// Package main is a generated GoMock package.
package main

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockMessageService is a mock of MessageService interface.
type MockMessageService struct {
	ctrl     *gomock.Controller
	recorder *MockMessageServiceMockRecorder
	isgomock struct{}
}

// MockMessageServiceMockRecorder is the mock recorder for MockMessageService.
type MockMessageServiceMockRecorder struct {
	mock *MockMessageService
}

// NewMockMessageService creates a new mock instance.
func NewMockMessageService(ctrl *gomock.Controller) *MockMessageService {
	mock := &MockMessageService{ctrl: ctrl}
	mock.recorder = &MockMessageServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMessageService) EXPECT() *MockMessageServiceMockRecorder {
	return m.recorder
}

// RetrieveSentMessages mocks base method.
func (m *MockMessageService) RetrieveSentMessages() ([]Message, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RetrieveSentMessages")
	ret0, _ := ret[0].([]Message)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RetrieveSentMessages indicates an expected call of RetrieveSentMessages.
func (mr *MockMessageServiceMockRecorder) RetrieveSentMessages() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RetrieveSentMessages", reflect.TypeOf((*MockMessageService)(nil).RetrieveSentMessages))
}
