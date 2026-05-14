package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"insider-one-assessment/internal/constants"
	"insider-one-assessment/internal/dto/request"
	"insider-one-assessment/internal/dto/resource"
	"insider-one-assessment/internal/model"
	"insider-one-assessment/internal/repository"
	"insider-one-assessment/internal/utils"
	"insider-one-assessment/pkgs/validation"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var (
	ErrInvalidRequest             = errors.New("invalid request")
	ErrTooManyNotifications       = errors.New("batch size exceeds 1000")
	ErrUnsupportedChannel         = errors.New("unsupported channel")
	ErrInvalidPriority            = errors.New("invalid priority")
	ErrEmptyRecipient             = errors.New("recipient is required")
	ErrEmptyContent               = errors.New("content is required")
	ErrContentTooLongSMS          = errors.New("sms content exceeds 160 characters")
	ErrInvalidScheduledAt         = errors.New("scheduled_at must be in the future")
	ErrTemplateNotFound           = errors.New("template not found")
	ErrEmptyTemplateName          = errors.New("template name is required")
	ErrEmptyTemplateContent       = errors.New("template content is required")
	ErrEmptyTemplateUpdate        = errors.New("at least one template field must be provided")
	ErrTemplateChannelMismatch    = errors.New("template channel does not match notification channel")
	ErrInvalidNotificationContent = errors.New("either content or template_id must be provided, but not both")
	ErrMissingTemplateVariable    = errors.New("missing template variable")
	ErrInvalidPagination          = errors.New("invalid pagination")
	ErrInvalidDateRange           = errors.New("invalid date range")
	ErrNotificationNotFound       = errors.New("notification not found")
	ErrNotificationNotCancellable = errors.New("notification is not cancellable")
	ErrBatchNotFound              = errors.New("batch not found")
)

var tracer = otel.Tracer("insider-one-assessment/service")

type INotificationService interface {
	CreateNotification(ctx context.Context, req request.CreateNotificationRequest) (*resource.NotificationResource, error)
	GetNotificationByID(ctx context.Context, notificationID string) (*resource.NotificationResource, error)
	ListNotifications(ctx context.Context, req request.ListNotificationsRequest) (*resource.ListNotificationsResponse, error)
	CancelNotification(ctx context.Context, notificationID string) (*resource.NotificationResource, error)
	CreateBatchNotification(ctx context.Context, req request.CreateNotificationsRequest) (*resource.BatchNotificationsCreateResource, error)
	GetBatchByID(ctx context.Context, batchID string) (*resource.NotificationBatchResource, error)
}

type notificationService struct {
	notificationReadRepo  repository.INotificationReadRepository
	notificationWriteRepo repository.INotificationWriteRepository
	templateRepo          repository.ITemplateRepository
}

func NewNotificationService(notificationReadRepo repository.INotificationReadRepository, notificationWriteRepo repository.INotificationWriteRepository, templateRepo repository.ITemplateRepository) INotificationService {
	return &notificationService{
		notificationReadRepo:  notificationReadRepo,
		notificationWriteRepo: notificationWriteRepo,
		templateRepo:          templateRepo,
	}
}

func (s *notificationService) CreateNotification(ctx context.Context, req request.CreateNotificationRequest) (*resource.NotificationResource, error) {
	ctx, span := tracer.Start(ctx, "notification_service.create_notification")
	defer span.End()
	span.SetAttributes(attribute.String("notification.channel", string(req.Notification.Channel)))

	if err := s.resolveCreateItem(ctx, &req.Notification); err != nil {
		return nil, err
	}

	requestHash, err := repository.HashCreateRequest([]request.NotificationItem{req.Notification})
	if err != nil {
		return nil, utils.NewErrorBag(constants.ProcessingErrCode, err, err.Error())
	}

	notification, _, err := s.notificationWriteRepo.CreateNotification(ctx, req.IdempotencyKey, requestHash, req.Notification)
	if err != nil {
		return nil, mapNotificationError(err)
	}
	return resource.NewNotificationResource(notification), nil
}

func (s *notificationService) GetNotificationByID(ctx context.Context, notificationID string) (*resource.NotificationResource, error) {
	ctx, span := tracer.Start(ctx, "notification_service.get_notification")
	defer span.End()
	span.SetAttributes(attribute.String("notification.id", notificationID))

	notification, err := s.notificationReadRepo.GetNotificationByID(ctx, notificationID)
	if err != nil {
		if errors.Is(err, repository.ErrNotificationNotFound) {
			return nil, utils.NewErrorBag(constants.NotFoundErrCode, ErrNotificationNotFound, ErrNotificationNotFound.Error())
		}
		return nil, utils.NewErrorBag(constants.ProcessingErrCode, err, err.Error())
	}

	return resource.NewNotificationResource(notification), nil
}

func (s *notificationService) ListNotifications(ctx context.Context, req request.ListNotificationsRequest) (*resource.ListNotificationsResponse, error) {
	ctx, span := tracer.Start(ctx, "notification_service.list_notifications")
	defer span.End()

	page := req.Page
	if page == 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize == 0 {
		pageSize = 20
	}

	params, err := s.buildListParams(req, page, pageSize)
	if err != nil {
		return nil, utils.NewErrorBag(constants.ValidationErrCode, err, err.Error())
	}

	notifications, total, err := s.notificationReadRepo.ListNotifications(ctx, params)
	if err != nil {
		return nil, utils.NewErrorBag(constants.ProcessingErrCode, err, err.Error())
	}

	return &resource.ListNotificationsResponse{
		Notifications: resource.NewNotificationsResource(notifications),
		Page:          page,
		PageSize:      pageSize,
		Total:         total,
	}, nil
}

func (s *notificationService) CancelNotification(ctx context.Context, notificationID string) (*resource.NotificationResource, error) {
	ctx, span := tracer.Start(ctx, "notification_service.cancel_notification")
	defer span.End()
	span.SetAttributes(attribute.String("notification.id", notificationID))

	notification, err := s.notificationWriteRepo.CancelNotification(ctx, notificationID, time.Now().UTC())
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotificationNotFound):
			return nil, utils.NewErrorBag(constants.NotFoundErrCode, ErrNotificationNotFound, ErrNotificationNotFound.Error())
		case errors.Is(err, repository.ErrNotificationNotCancellable):
			return nil, utils.NewErrorBag(constants.ConflictErrCode, ErrNotificationNotCancellable, ErrNotificationNotCancellable.Error())
		default:
			return nil, utils.NewErrorBag(constants.ProcessingErrCode, err, err.Error())
		}
	}
	return resource.NewNotificationResource(notification), nil
}

func (s *notificationService) CreateBatchNotification(ctx context.Context, req request.CreateNotificationsRequest) (*resource.BatchNotificationsCreateResource, error) {
	ctx, span := tracer.Start(ctx, "notification_service.create_batch")
	defer span.End()
	span.SetAttributes(attribute.Int("notification.batch_size", len(req.Notifications)))

	batch, notifications, replayed, err := s.createNotifications(ctx, req.IdempotencyKey, req.Notifications, true)
	if err != nil {
		return nil, mapNotificationError(err)
	}

	notificationIDs := make([]string, 0, len(notifications))
	for _, notification := range notifications {
		notificationIDs = append(notificationIDs, notification.ID.String())
	}

	return &resource.BatchNotificationsCreateResource{
		BatchID:         batch.ID.String(),
		NotificationIDs: notificationIDs,
		Status:          model.StatusPending,
		Replayed:        replayed,
	}, nil
}

func (s *notificationService) GetBatchByID(ctx context.Context, batchID string) (*resource.NotificationBatchResource, error) {
	ctx, span := tracer.Start(ctx, "notification_service.get_batch")
	defer span.End()
	span.SetAttributes(attribute.String("notification.batch_id", batchID))

	batch, notifications, err := s.notificationReadRepo.GetBatchByID(ctx, batchID)
	if err != nil {
		if errors.Is(err, repository.ErrNotificationNotFound) {
			return nil, utils.NewErrorBag(constants.NotFoundErrCode, ErrBatchNotFound, ErrBatchNotFound.Error())
		}
		return nil, utils.NewErrorBag(constants.ProcessingErrCode, err, err.Error())
	}

	return &resource.NotificationBatchResource{
		ID:             batch.ID.String(),
		IdempotencyKey: batch.IdempotencyKey,
		TotalCount:     batch.TotalCount,
		CreatedAt:      batch.CreatedAt,
		Notifications:  resource.NewNotificationsResource(notifications),
	}, nil
}

func (s *notificationService) createNotifications(ctx context.Context, idempotencyKey string, rawItems []request.NotificationItem, enforceBatchLimit bool) (model.NotificationBatch, []model.Notification, bool, error) {
	if len(rawItems) == 0 {
		return model.NotificationBatch{}, nil, false, utils.NewErrorBag(constants.ValidationErrCode, ErrInvalidRequest, "at least one notification is required")
	}
	if enforceBatchLimit && len(rawItems) > 1000 {
		return model.NotificationBatch{}, nil, false, utils.NewErrorBag(constants.ValidationErrCode, ErrTooManyNotifications, ErrTooManyNotifications.Error())
	}

	for i := range rawItems {
		if err := s.resolveCreateItem(ctx, &rawItems[i]); err != nil {
			return model.NotificationBatch{}, nil, false, mapNotificationError(err)
		}
	}

	requestHash, err := repository.HashCreateRequest(rawItems)
	if err != nil {
		return model.NotificationBatch{}, nil, false, utils.NewErrorBag(constants.ProcessingErrCode, err, fmt.Sprintf("hash request: %v", err))
	}

	batch, notifications, replayed, err := s.notificationWriteRepo.CreateNotifications(ctx, repository.CreateNotificationsParams{
		IdempotencyKey: idempotencyKey,
		RequestHash:    requestHash,
		Notifications:  rawItems,
	})
	if err != nil {
		return model.NotificationBatch{}, nil, false, mapNotificationError(err)
	}

	return batch, notifications, replayed, nil
}

func mapNotificationError(err error) error {
	switch {
	case err == nil:
		return nil
	case isNotificationValidationError(err):
		return utils.NewErrorBag(constants.ValidationErrCode, err, err.Error())
	case errors.Is(err, repository.ErrIdempotencyConflict),
		errors.Is(err, ErrNotificationNotCancellable):
		return utils.NewErrorBag(constants.ConflictErrCode, err, err.Error())
	case errors.Is(err, repository.ErrNotificationNotFound),
		errors.Is(err, ErrNotificationNotFound),
		errors.Is(err, ErrBatchNotFound),
		errors.Is(err, repository.ErrTemplateNotFound),
		errors.Is(err, ErrTemplateNotFound):
		return utils.NewErrorBag(constants.NotFoundErrCode, err, err.Error())
	default:
		return utils.NewErrorBag(constants.ProcessingErrCode, err, err.Error())
	}
}

func isNotificationValidationError(err error) bool {
	return errors.Is(err, ErrInvalidRequest) ||
		errors.Is(err, ErrTooManyNotifications) ||
		errors.Is(err, ErrUnsupportedChannel) ||
		errors.Is(err, ErrInvalidPriority) ||
		errors.Is(err, ErrEmptyRecipient) ||
		errors.Is(err, ErrEmptyContent) ||
		errors.Is(err, ErrContentTooLongSMS) ||
		errors.Is(err, ErrInvalidScheduledAt) ||
		errors.Is(err, ErrTemplateChannelMismatch) ||
		errors.Is(err, ErrMissingTemplateVariable) ||
		errors.Is(err, ErrInvalidNotificationContent) ||
		errors.Is(err, ErrInvalidPagination) ||
		errors.Is(err, ErrInvalidDateRange)
}

func (s *notificationService) resolveCreateItem(ctx context.Context, item *request.NotificationItem) error {
	if item.TemplateID == nil {
		return nil
	}
	if err := validation.ValidateUUID("template id", *item.TemplateID); err != nil {
		return err
	}

	template, err := s.templateRepo.GetTemplateByID(ctx, *item.TemplateID)
	if err != nil {
		if errors.Is(err, repository.ErrTemplateNotFound) {
			return ErrTemplateNotFound
		}
		return err
	}

	if template.Channel != item.Channel {
		return ErrTemplateChannelMismatch
	}

	renderedContent, err := renderTemplateContent(template.Content, item.Variables)
	if err != nil {
		return err
	}

	item.Content = renderedContent
	if item.Channel == model.ChannelSMS && len(item.Content) > 160 {
		return ErrContentTooLongSMS
	}
	return nil
}

var templateVariablePattern = regexp.MustCompile(`{{\s*([a-zA-Z0-9_]+)\s*}}`)

func renderTemplateContent(templateContent string, variables map[string]string) (string, error) {
	matches := templateVariablePattern.FindAllStringSubmatch(templateContent, -1)
	if len(matches) == 0 {
		return templateContent, nil
	}

	rendered := templateContent
	for _, match := range matches {
		key := match[1]
		value, ok := variables[key]
		if !ok {
			return "", fmt.Errorf("%w: %s", ErrMissingTemplateVariable, key)
		}
		rendered = strings.ReplaceAll(rendered, match[0], value)
	}

	return rendered, nil
}

func (s *notificationService) buildListParams(req request.ListNotificationsRequest, page, pageSize int) (repository.ListNotificationsParams, error) {
	params := repository.ListNotificationsParams{
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	}

	if req.Status != "" {
		status := model.NotificationStatus(req.Status)
		params.Status = &status
	}

	if req.Channel != "" {
		channel := model.NotificationChannel(req.Channel)
		params.Channel = &channel
	}

	if req.DateFrom != "" {
		parsed, err := time.Parse(time.RFC3339, req.DateFrom)
		if err != nil {
			return repository.ListNotificationsParams{}, ErrInvalidDateRange
		}
		params.DateFrom = &parsed
	}

	if req.DateTo != "" {
		parsed, err := time.Parse(time.RFC3339, req.DateTo)
		if err != nil {
			return repository.ListNotificationsParams{}, ErrInvalidDateRange
		}
		params.DateTo = &parsed
	}

	return params, nil
}
