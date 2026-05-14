package service

import (
	"context"
	"errors"

	"insider-one-assessment/internal/constants"
	"insider-one-assessment/internal/dto/request"
	"insider-one-assessment/internal/dto/resource"
	"insider-one-assessment/internal/model"
	"insider-one-assessment/internal/repository"
	"insider-one-assessment/internal/utils"
)

type ITemplateService interface {
	CreateTemplate(ctx context.Context, req request.CreateTemplateRequest) (*resource.TemplateResource, error)
	GetTemplateByID(ctx context.Context, templateUUID string) (*resource.TemplateResource, error)
	ListTemplates(ctx context.Context) ([]resource.TemplateResource, error)
	UpdateTemplate(ctx context.Context, templateUUID string, body request.UpdateTemplateRequest) (*resource.TemplateResource, error)
}

type templateService struct {
	templateRepo repository.ITemplateRepository
}

func NewTemplateService(templateRepo repository.ITemplateRepository) ITemplateService {
	return &templateService{
		templateRepo: templateRepo,
	}
}

func (s *templateService) CreateTemplate(ctx context.Context, req request.CreateTemplateRequest) (*resource.TemplateResource, error) {
	template := model.MessageTemplate{
		Name:    req.Name,
		Channel: req.Channel,
		Content: req.Content,
	}
	err := s.templateRepo.CreateTemplate(ctx, &template)
	if err != nil {
		if errors.Is(err, repository.ErrTemplateAlreadyExists) {
			return nil, utils.NewErrorBag(constants.ConflictErrCode, err, err.Error())
		}
		return nil, utils.NewErrorBag(constants.ProcessingErrCode, err, err.Error())
	}
	return resource.NewTemplateResource(&template), nil
}

func (s *templateService) GetTemplateByID(ctx context.Context, uuid string) (*resource.TemplateResource, error) {
	template, err := s.templateRepo.GetTemplateByID(ctx, uuid)
	if err != nil {
		if errors.Is(err, repository.ErrTemplateNotFound) {
			return nil, utils.NewErrorBag(constants.NotFoundErrCode, ErrTemplateNotFound, ErrTemplateNotFound.Error())
		}
		return nil, utils.NewErrorBag(constants.ProcessingErrCode, err, err.Error())
	}

	return resource.NewTemplateResource(template), nil
}

func (s *templateService) ListTemplates(ctx context.Context) ([]resource.TemplateResource, error) {
	templates, err := s.templateRepo.ListTemplates(ctx)
	if err != nil {
		return nil, utils.NewErrorBag(constants.ProcessingErrCode, err, err.Error())
	}
	return resource.NewTemplatesResource(templates), nil
}

func (s *templateService) UpdateTemplate(ctx context.Context, uuid string, payload request.UpdateTemplateRequest) (*resource.TemplateResource, error) {
	template, err := s.templateRepo.GetTemplateByID(ctx, uuid)
	if err != nil {
		if errors.Is(err, repository.ErrTemplateNotFound) {
			return nil, utils.NewErrorBag(constants.NotFoundErrCode, ErrTemplateNotFound, ErrTemplateNotFound.Error())
		}
		return nil, utils.NewErrorBag(constants.ProcessingErrCode, err, err.Error())
	}

	if payload.Name != nil {
		template.Name = *payload.Name
	}

	if payload.Channel != nil {
		template.Channel = *payload.Channel
	}

	if payload.Content != nil {
		template.Content = *payload.Content
	}

	err = s.templateRepo.UpdateTemplate(ctx, template)
	if err != nil {
		if errors.Is(err, repository.ErrTemplateAlreadyExists) {
			return nil, utils.NewErrorBag(constants.ConflictErrCode, err, err.Error())
		}
		return nil, utils.NewErrorBag(constants.ProcessingErrCode, err, err.Error())
	}

	return resource.NewTemplateResource(template), nil
}
