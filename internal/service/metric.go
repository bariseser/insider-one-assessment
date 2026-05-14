package service

import (
	"context"
	"insider-one-assessment/internal/repository"
)

type IMetricService interface {
	GetSnapshot(ctx context.Context) (*repository.MetricSnapshot, error)
}

type metricService struct {
	repo repository.IMetricRepository
}

func NewMetricService(repo repository.IMetricRepository) IMetricService {
	return &metricService{repo: repo}
}

func (s *metricService) GetSnapshot(ctx context.Context) (*repository.MetricSnapshot, error) {
	return s.repo.LoadSnapshot(ctx)
}
