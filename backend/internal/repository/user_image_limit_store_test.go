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
