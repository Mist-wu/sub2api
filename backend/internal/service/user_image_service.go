package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Mist-wu/sub2api/internal/pkg/errors"
	"github.com/Mist-wu/sub2api/internal/pkg/pagination"
	"github.com/tidwall/gjson"
)

const (
	UserImageModel            = "gpt-image-2"
	UserImageDailyLimit       = 100
	UserImageConcurrencyLimit = 8
	UserImageHistoryLimit     = 100
	UserImagePromptMaxLength  = 4000

	userImageGenerationLimitTTL = 15 * time.Minute
)

var (
	ErrUserImagePromptRequired = infraerrors.BadRequest("IMAGE_PROMPT_REQUIRED", "请输入生图提示词")
	ErrUserImagePromptTooLong  = infraerrors.BadRequest("IMAGE_PROMPT_TOO_LONG", "生图提示词最多 4000 个字符")
	ErrUserImageDailyLimit     = infraerrors.TooManyRequests("IMAGE_DAILY_LIMIT_EXCEEDED", "今日免费绘图次数已用完")
	ErrUserImageConcurrency    = infraerrors.TooManyRequests("IMAGE_CONCURRENCY_LIMIT_EXCEEDED", "当前绘图任务过多，请稍后再试")
	ErrUserImageNoGroup        = infraerrors.ServiceUnavailable("IMAGE_OPENAI_GROUP_UNAVAILABLE", "当前没有可用的 OpenAI 绘图分组")
	ErrUserImageNoAccount      = infraerrors.ServiceUnavailable("IMAGE_OPENAI_ACCOUNT_UNAVAILABLE", "当前没有可用的 OpenAI 绘图账号，请稍后再试")
	ErrUserImageNoOutput       = infraerrors.ServiceUnavailable("IMAGE_UPSTREAM_EMPTY_OUTPUT", "上游没有返回可保存的图片")
	ErrUserImageNotFound       = infraerrors.NotFound("IMAGE_HISTORY_NOT_FOUND", "图片历史不存在")
)

// UserImageGeneration is the persisted free image generation record.
type UserImageGeneration struct {
	ID            int64
	UserID        int64
	Prompt        string
	RevisedPrompt *string
	Model         string
	MimeType      string
	ImageData     []byte
	ImageSHA256   string
	CreatedAt     time.Time
}

// UserImageGenerationRepository stores user-side image generations.
type UserImageGenerationRepository interface {
	Create(ctx context.Context, item *UserImageGeneration) error
	ListByUserID(ctx context.Context, userID int64, params pagination.PaginationParams) ([]UserImageGeneration, *pagination.PaginationResult, error)
	GetByID(ctx context.Context, id int64) (*UserImageGeneration, error)
	DeleteOlderThanUserLimit(ctx context.Context, userID int64, keep int) error
}

// UserImageLimitStore enforces free image-generation quota and concurrency.
type UserImageLimitStore interface {
	ReserveDaily(ctx context.Context, userID int64, day string, limit int, ttl time.Duration) error
	AcquireConcurrency(ctx context.Context, userID int64, limit int, ttl time.Duration) (func(), error)
}

// UserImageService handles JWT-only free image generation and history.
type UserImageService struct {
	repo          UserImageGenerationRepository
	limitStore    UserImageLimitStore
	apiKeyService *APIKeyService
	openaiGateway *OpenAIGatewayService
}

// NewUserImageService creates a user image service.
func NewUserImageService(
	repo UserImageGenerationRepository,
	limitStore UserImageLimitStore,
	apiKeyService *APIKeyService,
	openaiGateway *OpenAIGatewayService,
) *UserImageService {
	return &UserImageService{
		repo:          repo,
		limitStore:    limitStore,
		apiKeyService: apiKeyService,
		openaiGateway: openaiGateway,
	}
}

// Generate creates one free image for a logged-in user.
func (s *UserImageService) Generate(ctx context.Context, userID int64, prompt string) (*UserImageGeneration, error) {
	if s == nil || s.repo == nil || s.apiKeyService == nil || s.openaiGateway == nil {
		return nil, infraerrors.ServiceUnavailable("IMAGE_SERVICE_UNAVAILABLE", "绘图服务暂不可用")
	}

	normalizedPrompt := strings.TrimSpace(prompt)
	if normalizedPrompt == "" {
		return nil, ErrUserImagePromptRequired
	}
	if len([]rune(normalizedPrompt)) > UserImagePromptMaxLength {
		return nil, ErrUserImagePromptTooLong
	}

	groups, err := s.openAIUserImageGroups(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, ErrUserImageNoGroup
	}

	now := time.Now()
	day, ttl := userImageShanghaiDayAndTTL(now)
	if s.limitStore != nil {
		release, err := s.limitStore.AcquireConcurrency(ctx, userID, UserImageConcurrencyLimit, userImageGenerationLimitTTL)
		if err != nil {
			return nil, err
		}
		defer release()
		if err := s.limitStore.ReserveDaily(ctx, userID, day, UserImageDailyLimit, ttl); err != nil {
			return nil, err
		}
	}

	body, parsed, err := buildUserImageOpenAIRequest(normalizedPrompt)
	if err != nil {
		return nil, infraerrors.BadRequest("IMAGE_REQUEST_INVALID", "绘图请求格式无效")
	}

	direct, err := s.forwardUserImage(ctx, userID, groups, body, parsed)
	if err != nil {
		return nil, err
	}
	imageData, mimeType, revisedPrompt, err := extractUserImageFromOpenAIResponse(direct.Body)
	if err != nil {
		return nil, err
	}

	sum := sha256.Sum256(imageData)
	item := &UserImageGeneration{
		UserID:      userID,
		Prompt:      normalizedPrompt,
		Model:       UserImageModel,
		MimeType:    mimeType,
		ImageData:   imageData,
		ImageSHA256: hex.EncodeToString(sum[:]),
		CreatedAt:   time.Now(),
	}
	if revisedPrompt != "" {
		item.RevisedPrompt = &revisedPrompt
	}
	if direct.ForwardResult != nil && strings.TrimSpace(direct.ForwardResult.Model) != "" {
		item.Model = strings.TrimSpace(direct.ForwardResult.Model)
	}

	if err := s.repo.Create(ctx, item); err != nil {
		return nil, fmt.Errorf("create user image generation: %w", err)
	}
	if err := s.repo.DeleteOlderThanUserLimit(ctx, userID, UserImageHistoryLimit); err != nil {
		return nil, fmt.Errorf("prune user image generation history: %w", err)
	}
	return item, nil
}

// ListHistory returns current user's image history metadata.
func (s *UserImageService) ListHistory(ctx context.Context, userID int64, params pagination.PaginationParams) ([]UserImageGeneration, *pagination.PaginationResult, error) {
	if s == nil || s.repo == nil {
		return nil, nil, infraerrors.ServiceUnavailable("IMAGE_SERVICE_UNAVAILABLE", "绘图服务暂不可用")
	}
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.PageSize > 50 {
		params.PageSize = 50
	}
	params.SortBy = "created_at"
	params.SortOrder = pagination.SortOrderDesc
	return s.repo.ListByUserID(ctx, userID, params)
}

// GetFile returns a history image owned by current user.
func (s *UserImageService) GetFile(ctx context.Context, userID int64, id int64) (*UserImageGeneration, error) {
	if s == nil || s.repo == nil {
		return nil, infraerrors.ServiceUnavailable("IMAGE_SERVICE_UNAVAILABLE", "绘图服务暂不可用")
	}
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil || item.UserID != userID {
		return nil, ErrUserImageNotFound
	}
	return item, nil
}

func (s *UserImageService) openAIUserImageGroups(ctx context.Context, userID int64) ([]Group, error) {
	groups, err := s.apiKeyService.GetAvailableGroups(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get available groups: %w", err)
	}
	openaiGroups := make([]Group, 0, len(groups))
	for _, group := range groups {
		if group.Platform == PlatformOpenAI && group.IsActive() {
			openaiGroups = append(openaiGroups, group)
		}
	}
	sort.SliceStable(openaiGroups, func(i, j int) bool {
		if openaiGroups[i].SortOrder == openaiGroups[j].SortOrder {
			return openaiGroups[i].ID < openaiGroups[j].ID
		}
		return openaiGroups[i].SortOrder < openaiGroups[j].SortOrder
	})
	return openaiGroups, nil
}

func (s *UserImageService) forwardUserImage(
	ctx context.Context,
	userID int64,
	groups []Group,
	body []byte,
	parsed *OpenAIImagesRequest,
) (*OpenAIImagesDirectResult, error) {
	sessionHash := fmt.Sprintf("user-image:%d", userID)
	var lastErr error
	for _, group := range groups {
		groupID := group.ID
		excluded := make(map[int64]struct{})
		for attempts := 0; attempts < 16; attempts++ {
			selection, _, err := s.openaiGateway.SelectAccountWithSchedulerForImages(
				ctx,
				&groupID,
				sessionHash,
				UserImageModel,
				excluded,
				parsed.RequiredCapability,
			)
			if err != nil {
				lastErr = err
				break
			}
			if selection == nil || selection.Account == nil {
				lastErr = ErrNoAvailableAccounts
				break
			}

			account := selection.Account
			direct, forwardErr := s.openaiGateway.ForwardImagesDirect(ctx, account, body, parsed, "")
			if selection.ReleaseFunc != nil {
				selection.ReleaseFunc()
			}
			if forwardErr == nil {
				return direct, nil
			}
			lastErr = mapUserImageForwardError(forwardErr, direct)
			var failoverErr *UpstreamFailoverError
			if stderrors.As(forwardErr, &failoverErr) {
				excluded[account.ID] = struct{}{}
				continue
			}
			return nil, lastErr
		}
	}
	if lastErr == nil || stderrors.Is(lastErr, ErrNoAvailableAccounts) {
		return nil, ErrUserImageNoAccount
	}
	return nil, mapUserImageForwardError(lastErr, nil)
}

func buildUserImageOpenAIRequest(prompt string) ([]byte, *OpenAIImagesRequest, error) {
	bodyPrompt := appendUserImageVisualConstraint(prompt)
	body, err := json.Marshal(map[string]any{
		"model":           UserImageModel,
		"prompt":          bodyPrompt,
		"n":               1,
		"response_format": "b64_json",
	})
	if err != nil {
		return nil, nil, err
	}
	parsed := &OpenAIImagesRequest{
		Endpoint:       openAIImagesGenerationsEndpoint,
		ContentType:    "application/json",
		Multipart:      false,
		Model:          UserImageModel,
		ExplicitModel:  true,
		Prompt:         bodyPrompt,
		N:              1,
		ResponseFormat: "b64_json",
		Body:           body,
	}
	parsed.SizeTier = normalizeOpenAIImageSizeTier(parsed.Size)
	parsed.RequiredCapability = classifyOpenAIImagesCapability(parsed)
	return body, parsed, nil
}

func appendUserImageVisualConstraint(prompt string) string {
	return strings.TrimSpace(prompt) + "\n\n统一视觉约束：在不覆盖以上主体要求的前提下，保持整体视觉风格统一、构图完整、光影和配色协调、细节清晰。如果用户明确指定视觉风格，以用户指定为准。"
}

func extractUserImageFromOpenAIResponse(body []byte) ([]byte, string, string, error) {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return nil, "", "", ErrUserImageNoOutput
	}
	data := gjson.GetBytes(body, "data")
	if !data.Exists() || !data.IsArray() || len(data.Array()) == 0 {
		return nil, "", "", ErrUserImageNoOutput
	}
	first := data.Array()[0]
	revisedPrompt := strings.TrimSpace(first.Get("revised_prompt").String())

	if b64 := normalizeOpenAIImageBase64(first.Get("b64_json").String()); b64 != "" {
		imageData, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil, "", "", ErrUserImageNoOutput
		}
		return imageData, detectUserImageMimeType(body, imageData), revisedPrompt, nil
	}
	if url := strings.TrimSpace(first.Get("url").String()); strings.HasPrefix(strings.ToLower(url), "data:image/") {
		imageData, mimeType, ok := decodeUserImageDataURL(url)
		if ok {
			return imageData, mimeType, revisedPrompt, nil
		}
	}
	return nil, "", "", ErrUserImageNoOutput
}

func detectUserImageMimeType(body []byte, imageData []byte) string {
	if format := strings.TrimSpace(gjson.GetBytes(body, "output_format").String()); format != "" {
		return openAIImageOutputMIMEType(format)
	}
	detected := http.DetectContentType(imageData)
	if strings.HasPrefix(detected, "image/") {
		return detected
	}
	return "image/png"
}

func decodeUserImageDataURL(raw string) ([]byte, string, bool) {
	raw = strings.TrimSpace(raw)
	idx := strings.Index(raw, ",")
	if idx <= 0 || idx+1 >= len(raw) {
		return nil, "", false
	}
	header := strings.ToLower(raw[:idx])
	if !strings.HasPrefix(header, "data:image/") || !strings.Contains(header, ";base64") {
		return nil, "", false
	}
	mimeType := strings.TrimPrefix(strings.Split(header, ";")[0], "data:")
	decoded, err := base64.StdEncoding.DecodeString(raw[idx+1:])
	if err != nil {
		return nil, "", false
	}
	return decoded, mimeType, true
}

func mapUserImageForwardError(err error, direct *OpenAIImagesDirectResult) error {
	if err == nil {
		return nil
	}
	msg := strings.TrimSpace(err.Error())
	if direct != nil {
		if upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(direct.Body)); upstreamMsg != "" {
			msg = upstreamMsg
		}
	}
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "content_policy") ||
		strings.Contains(lower, "safety") ||
		strings.Contains(lower, "moderation") ||
		strings.Contains(lower, "policy"):
		return infraerrors.BadRequest("IMAGE_POLICY_REJECTED", "提示词触发上游安全策略，请调整后重试")
	case strings.Contains(lower, "invalid") ||
		strings.Contains(lower, "unsupported") ||
		strings.Contains(lower, "parameter"):
		return infraerrors.BadRequest("IMAGE_UPSTREAM_INVALID_REQUEST", "上游拒绝了本次绘图请求，请调整提示词后重试")
	case strings.Contains(lower, "rate limit") || strings.Contains(lower, "429"):
		return infraerrors.TooManyRequests("IMAGE_UPSTREAM_RATE_LIMITED", "上游绘图服务繁忙，请稍后再试")
	}
	if stderrors.Is(err, context.DeadlineExceeded) {
		return infraerrors.GatewayTimeout("IMAGE_UPSTREAM_TIMEOUT", "图片生成超时，请稍后重试")
	}
	return infraerrors.ServiceUnavailable("IMAGE_UPSTREAM_FAILED", "图片生成失败，请稍后重试")
}

func userImageShanghaiDayAndTTL(now time.Time) (string, time.Duration) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	localNow := now.In(loc)
	startOfNextDay := time.Date(localNow.Year(), localNow.Month(), localNow.Day()+1, 0, 0, 0, 0, loc)
	ttl := startOfNextDay.Sub(localNow)
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return localNow.Format("2006-01-02"), ttl
}
