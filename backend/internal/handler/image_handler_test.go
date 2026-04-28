package handler

import (
	"testing"
	"time"

	"github.com/Mist-wu/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestImageGenerationJobResponseOmitsFullImageBase64(t *testing.T) {
	job := &service.UserImageJob{
		ID:        "img_test",
		UserID:    7,
		Prompt:    "glass star",
		Status:    service.UserImageJobStatusSucceeded,
		CreatedAt: time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC),
		Result: &service.UserImageGeneration{
			ID:                12,
			UserID:            7,
			Prompt:            "glass star",
			Model:             service.UserImageModel,
			MimeType:          "image/png",
			ImageData:         []byte("full-image-data"),
			ThumbnailData:     []byte("thumbnail-data"),
			ThumbnailMimeType: "image/jpeg",
			CreatedAt:         time.Date(2026, 4, 28, 10, 1, 0, 0, time.UTC),
		},
	}

	resp := toImageGenerationJobResponse(job)
	require.NotNil(t, resp.Result)
	require.Empty(t, resp.Result.ImageBase64)
	require.NotEmpty(t, resp.Result.ThumbnailBase64)
}
