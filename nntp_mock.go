// Code generated by MockGen. DO NOT EDIT.
// Source: ./nntp.go

// Package nntpcli is a generated GoMock package.
package nntpcli

import (
	context "context"
	reflect "reflect"
	time "time"

	gomock "github.com/golang/mock/gomock"
)

// MockClient is a mock of Client interface.
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *MockClientMockRecorder
}

// MockClientMockRecorder is the mock recorder for MockClient.
type MockClientMockRecorder struct {
	mock *MockClient
}

// NewMockClient creates a new mock instance.
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &MockClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClient) EXPECT() *MockClientMockRecorder {
	return m.recorder
}

// Dial mocks base method.
func (m *MockClient) Dial(ctx context.Context, host string, port int, maxAgeTime time.Time) (Connection, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Dial", ctx, host, port, maxAgeTime)
	ret0, _ := ret[0].(Connection)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Dial indicates an expected call of Dial.
func (mr *MockClientMockRecorder) Dial(ctx, host, port, maxAgeTime interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Dial", reflect.TypeOf((*MockClient)(nil).Dial), ctx, host, port, maxAgeTime)
}

// DialTLS mocks base method.
func (m *MockClient) DialTLS(ctx context.Context, host string, port int, insecureSSL bool, maxAgeTime time.Time) (Connection, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DialTLS", ctx, host, port, insecureSSL, maxAgeTime)
	ret0, _ := ret[0].(Connection)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DialTLS indicates an expected call of DialTLS.
func (mr *MockClientMockRecorder) DialTLS(ctx, host, port, insecureSSL, maxAgeTime interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DialTLS", reflect.TypeOf((*MockClient)(nil).DialTLS), ctx, host, port, insecureSSL, maxAgeTime)
}