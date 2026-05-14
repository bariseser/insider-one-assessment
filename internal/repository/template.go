package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"insider-one-assessment/internal/model"
	postgresdb "insider-one-assessment/pkgs/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type ITemplateRepository interface {
	CreateTemplate(ctx context.Context, template *model.MessageTemplate) error
	GetTemplateByID(ctx context.Context, uuid string) (*model.MessageTemplate, error)
	ListTemplates(ctx context.Context) ([]model.MessageTemplate, error)
	UpdateTemplate(ctx context.Context, template *model.MessageTemplate) error
}

type templateRepository struct {
	db     *sql.DB
	gormDB *gorm.DB
}

func NewTemplateRepository(client postgresdb.IPostgresInstance) ITemplateRepository {
	return &templateRepository{
		db:     client.SQLDatabase(),
		gormDB: client.Database(),
	}
}

func (r *templateRepository) CreateTemplate(ctx context.Context, template *model.MessageTemplate) error {
	template.ID = uuid.New()
	err := r.gormDB.WithContext(ctx).Create(template).Error
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrTemplateAlreadyExists
		}
		return fmt.Errorf("create template: %w", err)
	}

	return nil
}

func (r *templateRepository) GetTemplateByID(ctx context.Context, uuid string) (*model.MessageTemplate, error) {
	var template model.MessageTemplate
	err := r.gormDB.WithContext(ctx).Where("id", uuid).First(&template).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("get template by id: %w", err)
	}
	return &template, nil
}

func (r *templateRepository) ListTemplates(ctx context.Context) ([]model.MessageTemplate, error) {
	var templates []model.MessageTemplate
	err := r.gormDB.WithContext(ctx).Find(&templates).Error
	if err != nil {
		return nil, err
	}
	return templates, nil
}

func (r *templateRepository) UpdateTemplate(ctx context.Context, template *model.MessageTemplate) error {
	return r.gormDB.WithContext(ctx).
		Model(&model.MessageTemplate{}).
		Where("id", template.ID).
		Updates(template).Error
}
