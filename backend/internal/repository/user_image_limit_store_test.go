package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Mist-wu/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserImageLimitStoreReserveDaily(t *testing.T) {
	store := NewUserImageLimitStore(nil)
	ctx := context.Background()

	for i := 0; i < service.UserImageDailyLimit; i++ {
		require.NoError(t, store.ReserveDaily(ctx, 7, "2026-04-28", service.UserImageDailyLimit, 24*time.Hour))
	}

	err := store.ReserveDaily(ctx, 7, "2026-04-28", service.UserImageDailyLimit, 24*time.Hour)
	require.ErrorIs(t, err, service.ErrUserImageDailyLimit)
}

func TestUserImageLimitStoreAcquireConcurrency(t *testing.T) {
	store := NewUserImageLimitStore(nil)
	ctx := context.Background()

	var releases []func()
	for i := 0; i < service.UserImageConcurrencyLimit; i++ {
		release, err := store.AcquireConcurrency(ctx, 9, service.UserImageConcurrencyLimit, time.Minute)
		require.NoError(t, err)
		releases = append(releases, release)
	}

	_, err := store.AcquireConcurrency(ctx, 9, service.UserImageConcurrencyLimit, time.Minute)
	require.ErrorIs(t, err, service.ErrUserImageConcurrency)

	releases[0]()
	releases[0]()
	release, err := store.AcquireConcurrency(ctx, 9, service.UserImageConcurrencyLimit, time.Minute)
	require.NoError(t, err)
	_, err = store.AcquireConcurrency(ctx, 9, service.UserImageConcurrencyLimit, time.Minute)
	require.ErrorIs(t, err, service.ErrUserImageConcurrency)
	release()
	for _, release := range releases[1:] {
		release()
	}
}

func TestUserImageLimitStoreJobRoundTrip(t *testing.T) {
	store := NewUserImageLimitStore(nil)
	ctx := context.Background()
	revisedPrompt := "soft light"
	job := &service.UserImageJob{
		ID:        "img_roundtrip",
		UserID:    7,
		Prompt:    "小猫",
		Status:    service.UserImageJobStatusSucceeded,
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
		Result: &service.UserImageGeneration{
			ID:                11,
			UserID:            7,
			Prompt:            "小猫",
			RevisedPrompt:     &revisedPrompt,
			Model:             service.UserImageModel,
			MimeType:          "image/png",
			ImageData:         []byte("full-image"),
			ThumbnailData:     []byte("thumb"),
			ThumbnailMimeType: "image/jpeg",
			CreatedAt:         time.Now(),
		},
	}

	require.NoError(t, store.StoreJob(ctx, job, time.Hour))
	got, err := store.GetJob(ctx, 7, "img_roundtrip")
	require.NoError(t, err)
	require.Equal(t, job.ID, got.ID)
	require.Equal(t, service.UserImageJobStatusSucceeded, got.Status)
	require.NotNil(t, got.Result)
	require.Empty(t, got.Result.ImageData)
	require.Equal(t, []byte("thumb"), got.Result.ThumbnailData)

	_, err = store.GetJob(ctx, 8, "img_roundtrip")
	require.ErrorIs(t, err, service.ErrUserImageNotFound)
}
