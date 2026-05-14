package resource

import (
	"time"

	"insider-one-assessment/internal/model"
)

type TemplateResponse struct {
	Template TemplateResource `json:"template"`
}

type TemplatesResponse struct {
	Templates []TemplateResource `json:"templates"`
}

type TemplateResource struct {
	ID        string                    `json:"id"`
	Name      string                    `json:"name"`
	Channel   model.NotificationChannel `json:"channel"`
	Content   string                    `json:"content"`
	CreatedAt time.Time                 `json:"created_at"`
	UpdatedAt time.Time                 `json:"updated_at"`
}

func NewTemplateResource(template *model.MessageTemplate) *TemplateResource {
	r := &TemplateResource{
		ID:        template.ID.String(),
		Name:      template.Name,
		Channel:   template.Channel,
		Content:   template.Content,
		CreatedAt: template.CreatedAt,
		UpdatedAt: template.UpdatedAt,
	}
	return r
}

func NewTemplatesResource(templates []model.MessageTemplate) []TemplateResource {
	var templatesResource []TemplateResource

	for i := range templates {
		r := NewTemplateResource(&templates[i])
		templatesResource = append(templatesResource, *r)
	}
	return templatesResource
}
