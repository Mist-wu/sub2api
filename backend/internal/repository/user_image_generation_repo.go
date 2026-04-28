package repository

import (
	"context"
	"fmt"
	"math"

	dbent "github.com/Mist-wu/sub2api/ent"
	dbuserimagegeneration "github.com/Mist-wu/sub2api/ent/userimagegeneration"
	"github.com/Mist-wu/sub2api/internal/pkg/pagination"
	"github.com/Mist-wu/sub2api/internal/service"
)

type userImageGenerationRepository struct {
	client *dbent.Client
}

// NewUserImageGenerationRepository creates the image history repository.
func NewUserImageGenerationRepository(client *dbent.Client) service.UserImageGenerationRepository {
	return &userImageGenerationRepository{client: client}
}

func (r *userImageGenerationRepository) Create(ctx context.Context, item *service.UserImageGeneration) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("nil user image generation repository")
	}
	if item == nil {
		return fmt.Errorf("nil user image generation")
	}
	builder := r.client.UserImageGeneration.Create().
		SetUserID(item.UserID).
		SetPrompt(item.Prompt).
		SetModel(item.Model).
		SetMimeType(item.MimeType).
		SetImageData(item.ImageData).
		SetImageSha256(item.ImageSHA256).
		SetCreatedAt(item.CreatedAt)
	if item.RevisedPrompt != nil {
		builder.SetRevisedPrompt(*item.RevisedPrompt)
	}
	if len(item.ThumbnailData) > 0 {
		builder.SetThumbnailData(item.ThumbnailData)
	}
	if item.ThumbnailMimeType != "" {
		builder.SetThumbnailMimeType(item.ThumbnailMimeType)
	}
	created, err := builder.Save(ctx)
	if err != nil {
		return err
	}
	item.ID = created.ID
	item.CreatedAt = created.CreatedAt
	return nil
}

func (r *userImageGenerationRepository) ListByUserID(ctx context.Context, userID int64, params pagination.PaginationParams) ([]service.UserImageGeneration, *pagination.PaginationResult, error) {
	if r == nil || r.client == nil {
		return nil, nil, fmt.Errorf("nil user image generation repository")
	}
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	total, err := r.client.UserImageGeneration.Query().
		Where(dbuserimagegeneration.UserIDEQ(userID)).
		Count(ctx)
	if err != nil {
		return nil, nil, err
	}
	rows, err := r.client.UserImageGeneration.Query().
		Where(dbuserimagegeneration.UserIDEQ(userID)).
		Order(dbent.Desc(dbuserimagegeneration.FieldCreatedAt), dbent.Desc(dbuserimagegeneration.FieldID)).
		Offset(params.Offset()).
		Limit(params.Limit()).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}
	items := make([]service.UserImageGeneration, 0, len(rows))
	for _, row := range rows {
		items = append(items, convertUserImageGeneration(row, false))
	}
	pages := int(math.Ceil(float64(total) / float64(params.PageSize)))
	if pages < 1 {
		pages = 1
	}
	return items, &pagination.PaginationResult{
		Total:    int64(total),
		Page:     params.Page,
		PageSize: params.PageSize,
		Pages:    pages,
	}, nil
}

func (r *userImageGenerationRepository) GetByID(ctx context.Context, id int64) (*service.UserImageGeneration, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("nil user image generation repository")
	}
	row, err := r.client.UserImageGeneration.Query().
		Where(dbuserimagegeneration.IDEQ(id)).
		Only(ctx)
	if dbent.IsNotFound(err) {
		return nil, service.ErrUserImageNotFound
	}
	if err != nil {
		return nil, err
	}
	item := convertUserImageGeneration(row, true)
	return &item, nil
}

func (r *userImageGenerationRepository) DeleteOlderThanUserLimit(ctx context.Context, userID int64, keep int) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("nil user image generation repository")
	}
	if keep <= 0 {
		_, err := r.client.UserImageGeneration.Delete().
			Where(dbuserimagegeneration.UserIDEQ(userID)).
			Exec(ctx)
		return err
	}
	rows, err := r.client.UserImageGeneration.Query().
		Where(dbuserimagegeneration.UserIDEQ(userID)).
		Order(dbent.Desc(dbuserimagegeneration.FieldCreatedAt), dbent.Desc(dbuserimagegeneration.FieldID)).
		Offset(keep).
		Select(dbuserimagegeneration.FieldID).
		All(ctx)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	_, err = r.client.UserImageGeneration.Delete().
		Where(dbuserimagegeneration.IDIn(ids...)).
		Exec(ctx)
	return err
}

func convertUserImageGeneration(row *dbent.UserImageGeneration, includeData bool) service.UserImageGeneration {
	item := service.UserImageGeneration{
		ID:          row.ID,
		UserID:      row.UserID,
		Prompt:      row.Prompt,
		Model:       row.Model,
		MimeType:    row.MimeType,
		ImageSHA256: row.ImageSha256,
		CreatedAt:   row.CreatedAt,
	}
	if row.RevisedPrompt != nil {
		item.RevisedPrompt = row.RevisedPrompt
	}
	if row.ThumbnailMimeType != nil {
		item.ThumbnailMimeType = *row.ThumbnailMimeType
	}
	item.ThumbnailData = append([]byte(nil), row.ThumbnailData...)
	if includeData || len(item.ThumbnailData) == 0 {
		item.ImageData = append([]byte(nil), row.ImageData...)
	}
	return item
}
