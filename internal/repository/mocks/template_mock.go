// Code generated manually to mirror GoMock style for ITemplateRepository.
package mocks

import (
	context "context"
	model "insider-one-assessment/internal/model"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

type MockITemplateRepository struct {
	ctrl     *gomock.Controller
	recorder *MockITemplateRepositoryMockRecorder
}

type MockITemplateRepositoryMockRecorder struct {
	mock *MockITemplateRepository
}

func NewMockITemplateRepository(ctrl *gomock.Controller) *MockITemplateRepository {
	mock := &MockITemplateRepository{ctrl: ctrl}
	mock.recorder = &MockITemplateRepositoryMockRecorder{mock}
	return mock
}

func (m *MockITemplateRepository) EXPECT() *MockITemplateRepositoryMockRecorder {
	return m.recorder
}

func (m *MockITemplateRepository) CreateTemplate(ctx context.Context, template *model.MessageTemplate) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateTemplate", ctx, template)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockITemplateRepositoryMockRecorder) CreateTemplate(ctx, template interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateTemplate", reflect.TypeOf((*MockITemplateRepository)(nil).CreateTemplate), ctx, template)
}

func (m *MockITemplateRepository) GetTemplateByID(ctx context.Context, uuid string) (*model.MessageTemplate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTemplateByID", ctx, uuid)
	ret0, _ := ret[0].(*model.MessageTemplate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockITemplateRepositoryMockRecorder) GetTemplateByID(ctx, uuid interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTemplateByID", reflect.TypeOf((*MockITemplateRepository)(nil).GetTemplateByID), ctx, uuid)
}

func (m *MockITemplateRepository) ListTemplates(ctx context.Context) ([]model.MessageTemplate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTemplates", ctx)
	ret0, _ := ret[0].([]model.MessageTemplate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockITemplateRepositoryMockRecorder) ListTemplates(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTemplates", reflect.TypeOf((*MockITemplateRepository)(nil).ListTemplates), ctx)
}

func (m *MockITemplateRepository) UpdateTemplate(ctx context.Context, template *model.MessageTemplate) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateTemplate", ctx, template)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockITemplateRepositoryMockRecorder) UpdateTemplate(ctx, template interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateTemplate", reflect.TypeOf((*MockITemplateRepository)(nil).UpdateTemplate), ctx, template)
}
