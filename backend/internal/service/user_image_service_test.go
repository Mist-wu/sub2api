package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/Mist-wu/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestBuildUserImageOpenAIRequestUsesFixedModelAndNoSize(t *testing.T) {
	body, parsed, err := buildUserImageOpenAIRequest("一只蓝色玻璃杯")
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.Equal(t, UserImageModel, parsed.Model)
	require.Equal(t, 1, parsed.N)
	require.False(t, gjson.GetBytes(body, "size").Exists())
	require.Equal(t, "b64_json", gjson.GetBytes(body, "response_format").String())
	require.True(t, strings.Contains(gjson.GetBytes(body, "prompt").String(), "统一视觉约束"))
}

func TestNormalizeUserImagePromptEnforcesFinalPayloadLength(t *testing.T) {
	overhead := len([]rune(appendUserImageVisualConstraint("")))
	prompt := strings.Repeat("a", UserImagePromptMaxLength-overhead)

	normalized, err := normalizeUserImagePrompt(prompt)
	require.NoError(t, err)
	require.Equal(t, prompt, normalized)

	_, err = normalizeUserImagePrompt(prompt + "a")
	require.ErrorIs(t, err, ErrUserImagePromptTooLong)
}

func TestExtractUserImageFromOpenAIResponse(t *testing.T) {
	rawImage := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	body := []byte(`{"output_format":"png","data":[{"b64_json":"` + base64.StdEncoding.EncodeToString(rawImage) + `","revised_prompt":"soft light"}]}`)

	imageData, mimeType, revisedPrompt, err := extractUserImageFromOpenAIResponse(body)
	require.NoError(t, err)
	require.Equal(t, rawImage, imageData)
	require.Equal(t, "image/png", mimeType)
	require.Equal(t, "soft light", revisedPrompt)
}

func TestBuildUserImageThumbnailCompressesImage(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 720, 360))
	for y := 0; y < 360; y++ {
		for x := 0; x < 720; x++ {
			src.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 180, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, src))

	thumbnail, mimeType := buildUserImageThumbnail(buf.Bytes())
	require.NotEmpty(t, thumbnail)
	require.Equal(t, "image/jpeg", mimeType)

	decoded, format, err := image.Decode(bytes.NewReader(thumbnail))
	require.NoError(t, err)
	require.Equal(t, "jpeg", format)
	require.LessOrEqual(t, decoded.Bounds().Dx(), userImageThumbnailMaxDim)
	require.LessOrEqual(t, decoded.Bounds().Dy(), userImageThumbnailMaxDim)
}

func TestFindActiveUserImageJobLockedDeduplicatesPrompt(t *testing.T) {
	svc := NewUserImageService(nil, nil, nil, nil)
	svc.jobs["img_running"] = &UserImageJob{
		ID:     "img_running",
		UserID: 7,
		Prompt: "小猫",
		Status: UserImageJobStatusRunning,
	}
	svc.jobs["img_other_user"] = &UserImageJob{
		ID:     "img_other_user",
		UserID: 8,
		Prompt: "小猫",
		Status: UserImageJobStatusRunning,
	}
	svc.jobs["img_done"] = &UserImageJob{
		ID:     "img_done",
		UserID: 7,
		Prompt: "小猫",
		Status: UserImageJobStatusSucceeded,
	}

	require.Equal(t, "img_running", svc.findActiveUserImageJobLocked(7, " 小猫 ").ID)
	require.Nil(t, svc.findActiveUserImageJobLocked(7, "小狗"))
}

func TestStartGenerationJobDedupesRunningJobBeforeAcquireConcurrency(t *testing.T) {
	store := &userImageLimitStoreStub{}
	svc := NewUserImageService(&userImageGenerationRepoStub{}, store, &APIKeyService{}, &OpenAIGatewayService{})
	svc.jobs["img_running"] = &UserImageJob{
		ID:        "img_running",
		UserID:    7,
		Prompt:    "小猫",
		Status:    UserImageJobStatusRunning,
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}

	job, err := svc.StartGenerationJob(context.Background(), 7, " 小猫 ")
	require.NoError(t, err)
	require.Equal(t, "img_running", job.ID)
	require.Zero(t, store.acquireCalls)
}

func TestStartGenerationJobConcurrencyLimitDoesNotCreateJob(t *testing.T) {
	store := &userImageLimitStoreStub{acquireErr: ErrUserImageConcurrency}
	svc := NewUserImageService(&userImageGenerationRepoStub{}, store, &APIKeyService{}, &OpenAIGatewayService{})

	_, err := svc.StartGenerationJob(context.Background(), 7, "小猫")
	require.ErrorIs(t, err, ErrUserImageConcurrency)
	require.Equal(t, 1, store.acquireCalls)
	require.Empty(t, svc.jobs)
	require.Nil(t, store.savedJob)
}

func TestStartGenerationJobStoreFailureRollsBackAndReleasesConcurrency(t *testing.T) {
	store := &userImageLimitStoreStub{storeErr: errors.New("store failed")}
	svc := NewUserImageService(&userImageGenerationRepoStub{}, store, &APIKeyService{}, &OpenAIGatewayService{})

	_, err := svc.StartGenerationJob(context.Background(), 7, "小猫")
	require.Error(t, err)
	require.Equal(t, 1, store.acquireCalls)
	require.Equal(t, 1, store.released)
	require.Empty(t, svc.jobs)
}

func TestGetGenerationJobFallsBackToStoredJob(t *testing.T) {
	store := &userImageLimitStoreStub{
		jobs: map[string]*UserImageJob{
			"img_stored": {
				ID:        "img_stored",
				UserID:    7,
				Prompt:    "小猫",
				Status:    UserImageJobStatusRunning,
				CreatedAt: time.Now(),
				StartedAt: time.Now(),
			},
		},
	}
	svc := NewUserImageService(nil, store, nil, nil)

	job, err := svc.GetGenerationJob(context.Background(), 7, "img_stored")
	require.NoError(t, err)
	require.Equal(t, "img_stored", job.ID)
	require.Equal(t, UserImageJobStatusRunning, job.Status)
}

func TestGetGenerationJobMarksStoredStaleRunningJobFailed(t *testing.T) {
	store := &userImageLimitStoreStub{
		jobs: map[string]*UserImageJob{
			"img_stale": {
				ID:        "img_stale",
				UserID:    7,
				Prompt:    "小猫",
				Status:    UserImageJobStatusRunning,
				CreatedAt: time.Now().Add(-2 * userImageJobTimeout),
				StartedAt: time.Now().Add(-2 * userImageJobTimeout),
			},
		},
	}
	svc := NewUserImageService(nil, store, nil, nil)

	job, err := svc.GetGenerationJob(context.Background(), 7, "img_stale")
	require.NoError(t, err)
	require.Equal(t, UserImageJobStatusFailed, job.Status)
	require.Equal(t, "IMAGE_UPSTREAM_TIMEOUT", job.ErrorReason)
	require.Equal(t, UserImageJobStatusFailed, store.savedJob.Status)
}

func TestCloneUserImageJobForStoreOmitsFullImageData(t *testing.T) {
	job := &UserImageJob{
		ID:     "img_done",
		UserID: 7,
		Result: &UserImageGeneration{
			ID:            11,
			UserID:        7,
			ImageData:     []byte("full-image"),
			ThumbnailData: []byte("thumb"),
		},
	}

	stored := cloneUserImageJobForStore(job)
	require.NotNil(t, stored.Result)
	require.Empty(t, stored.Result.ImageData)
	require.Equal(t, []byte("thumb"), stored.Result.ThumbnailData)
}

func TestListHistoryThumbnailBackfillIsolatesLoadFailure(t *testing.T) {
	imageData := testUserImagePNG(t)
	repo := &userImageGenerationRepoStub{
		items: []UserImageGeneration{
			{ID: 1, UserID: 7, Prompt: "broken", Model: UserImageModel, MimeType: "image/png", CreatedAt: time.Now()},
			{ID: 2, UserID: 7, Prompt: "ok", Model: UserImageModel, MimeType: "image/png", CreatedAt: time.Now()},
		},
		get: map[int64]*UserImageGeneration{
			2: {ID: 2, UserID: 7, ImageData: imageData, MimeType: "image/png"},
		},
		getErr: map[int64]error{
			1: errors.New("load failed"),
		},
	}
	svc := NewUserImageService(repo, nil, nil, nil)

	items, pag, err := svc.ListHistory(context.Background(), 7, pagination.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.NotNil(t, pag)
	require.Len(t, items, 2)
	require.Empty(t, items[0].ThumbnailData)
	require.NotEmpty(t, items[1].ThumbnailData)
	require.Empty(t, items[0].ImageData)
	require.Empty(t, items[1].ImageData)
}

func TestListHistoryThumbnailUpdateFailureDoesNotFailList(t *testing.T) {
	imageData := testUserImagePNG(t)
	repo := &userImageGenerationRepoStub{
		items: []UserImageGeneration{
			{ID: 3, UserID: 7, Prompt: "ok", Model: UserImageModel, MimeType: "image/png", CreatedAt: time.Now()},
		},
		get: map[int64]*UserImageGeneration{
			3: {ID: 3, UserID: 7, ImageData: imageData, MimeType: "image/png"},
		},
		updateErr: errors.New("write failed"),
	}
	svc := NewUserImageService(repo, nil, nil, nil)

	items, _, err := svc.ListHistory(context.Background(), 7, pagination.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.NotEmpty(t, items[0].ThumbnailData)
	require.Equal(t, 1, repo.updateCalls)
}

type userImageLimitStoreStub struct {
	acquireCalls int
	acquireErr   error
	storeErr     error
	released     int
	jobs         map[string]*UserImageJob
	savedJob     *UserImageJob
}

func (s *userImageLimitStoreStub) ReserveDaily(context.Context, int64, string, int, time.Duration) error {
	return nil
}

func (s *userImageLimitStoreStub) AcquireConcurrency(context.Context, int64, int, time.Duration) (func(), error) {
	s.acquireCalls++
	if s.acquireErr != nil {
		return nil, s.acquireErr
	}
	return func() { s.released++ }, nil
}

func (s *userImageLimitStoreStub) StoreJob(_ context.Context, job *UserImageJob, _ time.Duration) error {
	if s.storeErr != nil {
		return s.storeErr
	}
	s.savedJob = cloneUserImageJob(job)
	if s.jobs == nil {
		s.jobs = make(map[string]*UserImageJob)
	}
	s.jobs[job.ID] = cloneUserImageJob(job)
	return nil
}

func (s *userImageLimitStoreStub) GetJob(_ context.Context, userID int64, jobID string) (*UserImageJob, error) {
	job := s.jobs[jobID]
	if job == nil || job.UserID != userID {
		return nil, ErrUserImageNotFound
	}
	return cloneUserImageJob(job), nil
}

type userImageGenerationRepoStub struct {
	items       []UserImageGeneration
	get         map[int64]*UserImageGeneration
	getErr      map[int64]error
	updateErr   error
	updateCalls int
}

func (r *userImageGenerationRepoStub) Create(context.Context, *UserImageGeneration) error {
	return nil
}

func (r *userImageGenerationRepoStub) ListByUserID(context.Context, int64, pagination.PaginationParams) ([]UserImageGeneration, *pagination.PaginationResult, error) {
	items := append([]UserImageGeneration(nil), r.items...)
	return items, &pagination.PaginationResult{Total: int64(len(items)), Page: 1, PageSize: 20, Pages: 1}, nil
}

func (r *userImageGenerationRepoStub) GetByID(_ context.Context, id int64) (*UserImageGeneration, error) {
	if err := r.getErr[id]; err != nil {
		return nil, err
	}
	item := r.get[id]
	if item == nil {
		return nil, ErrUserImageNotFound
	}
	return cloneUserImageGeneration(item), nil
}

func (r *userImageGenerationRepoStub) UpdateThumbnail(context.Context, int64, int64, []byte, string) error {
	r.updateCalls++
	return r.updateErr
}

func (r *userImageGenerationRepoStub) DeleteOlderThanUserLimit(context.Context, int64, int) error {
	return nil
}

func testUserImagePNG(t *testing.T) []byte {
	t.Helper()
	src := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			src.Set(x, y, color.RGBA{R: uint8(x * 4), G: uint8(y * 4), B: 180, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, src))
	return buf.Bytes()
}
