package handler

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Mist-wu/sub2api/internal/pkg/pagination"
	"github.com/Mist-wu/sub2api/internal/pkg/response"
	"github.com/Mist-wu/sub2api/internal/server/middleware"
	"github.com/Mist-wu/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ImageHandler handles JWT-only user image generation APIs.
type ImageHandler struct {
	imageService *service.UserImageService
}

// NewImageHandler creates an ImageHandler.
func NewImageHandler(imageService *service.UserImageService) *ImageHandler {
	return &ImageHandler{imageService: imageService}
}

type imageGenerationRequest struct {
	Prompt string `json:"prompt"`
}

type imageGenerationResponse struct {
	ID                int64   `json:"id"`
	Prompt            string  `json:"prompt"`
	RevisedPrompt     *string `json:"revised_prompt,omitempty"`
	Model             string  `json:"model"`
	MimeType          string  `json:"mime_type"`
	ImageBase64       string  `json:"image_base64,omitempty"`
	ThumbnailMimeType string  `json:"thumbnail_mime_type,omitempty"`
	ThumbnailBase64   string  `json:"thumbnail_base64,omitempty"`
	CreatedAt         string  `json:"created_at"`
}

type imageHistoryItemResponse struct {
	ID                int64   `json:"id"`
	Prompt            string  `json:"prompt"`
	RevisedPrompt     *string `json:"revised_prompt,omitempty"`
	Model             string  `json:"model"`
	MimeType          string  `json:"mime_type"`
	ImageSHA256       string  `json:"image_sha256"`
	ThumbnailMimeType string  `json:"thumbnail_mime_type,omitempty"`
	ThumbnailBase64   string  `json:"thumbnail_base64,omitempty"`
	CreatedAt         string  `json:"created_at"`
}

type imageGenerationJobResponse struct {
	JobID        string                   `json:"job_id"`
	Prompt       string                   `json:"prompt"`
	Status       string                   `json:"status"`
	ErrorMessage string                   `json:"error_message,omitempty"`
	ErrorReason  string                   `json:"error_reason,omitempty"`
	CreatedAt    string                   `json:"created_at"`
	StartedAt    string                   `json:"started_at,omitempty"`
	CompletedAt  string                   `json:"completed_at,omitempty"`
	Result       *imageGenerationResponse `json:"result,omitempty"`
}

// Generate creates one image for the authenticated user.
func (h *ImageHandler) Generate(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}
	var req imageGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	job, err := h.imageService.StartGenerationJob(c.Request.Context(), subject.UserID, req.Prompt)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Accepted(c, toImageGenerationJobResponse(job))
}

// GetGeneration returns one asynchronous generation job for the authenticated user.
func (h *ImageHandler) GetGeneration(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}
	job, err := h.imageService.GetGenerationJob(c.Request.Context(), subject.UserID, c.Param("job_id"))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, toImageGenerationJobResponse(job))
}

// ListHistory returns current user's image generation history metadata.
func (h *ImageHandler) ListHistory(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, pag, err := h.imageService.ListHistory(c.Request.Context(), subject.UserID, pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    "created_at",
		SortOrder: pagination.SortOrderDesc,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	out := make([]imageHistoryItemResponse, 0, len(items))
	for _, item := range items {
		out = append(out, toImageHistoryItemResponse(&item))
	}
	if pag == nil {
		response.Paginated(c, out, 0, page, pageSize)
		return
	}
	response.Paginated(c, out, pag.Total, pag.Page, pag.PageSize)
}

// GetHistoryFile returns a history image file for preview/download.
func (h *ImageHandler) GetHistoryFile(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid image id")
		return
	}
	item, err := h.imageService.GetFile(c.Request.Context(), subject.UserID, id)
	if response.ErrorFrom(c, err) {
		return
	}
	mimeType := strings.TrimSpace(item.MimeType)
	if mimeType == "" {
		mimeType = "image/png"
	}
	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="image-%d%s"`, item.ID, extensionForImageMimeType(mimeType)))
	c.Data(http.StatusOK, mimeType, item.ImageData)
}

func toImageGenerationResponse(item *service.UserImageGeneration) imageGenerationResponse {
	return toImageGenerationResponseWithOptions(item, false)
}

func toImageGenerationResponseWithOptions(item *service.UserImageGeneration, includeImageBase64 bool) imageGenerationResponse {
	if item == nil {
		return imageGenerationResponse{}
	}
	out := imageGenerationResponse{
		ID:                item.ID,
		Prompt:            item.Prompt,
		RevisedPrompt:     item.RevisedPrompt,
		Model:             item.Model,
		MimeType:          item.MimeType,
		ThumbnailMimeType: item.ThumbnailMimeType,
		CreatedAt:         item.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if includeImageBase64 && len(item.ImageData) > 0 {
		out.ImageBase64 = base64.StdEncoding.EncodeToString(item.ImageData)
	}
	if len(item.ThumbnailData) > 0 {
		out.ThumbnailBase64 = base64.StdEncoding.EncodeToString(item.ThumbnailData)
	}
	return out
}

func toImageHistoryItemResponse(item *service.UserImageGeneration) imageHistoryItemResponse {
	if item == nil {
		return imageHistoryItemResponse{}
	}
	out := imageHistoryItemResponse{
		ID:                item.ID,
		Prompt:            item.Prompt,
		RevisedPrompt:     item.RevisedPrompt,
		Model:             item.Model,
		MimeType:          item.MimeType,
		ImageSHA256:       item.ImageSHA256,
		ThumbnailMimeType: item.ThumbnailMimeType,
		CreatedAt:         item.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if len(item.ThumbnailData) > 0 {
		out.ThumbnailBase64 = base64.StdEncoding.EncodeToString(item.ThumbnailData)
	}
	return out
}

func toImageGenerationJobResponse(job *service.UserImageJob) imageGenerationJobResponse {
	if job == nil {
		return imageGenerationJobResponse{}
	}
	out := imageGenerationJobResponse{
		JobID:        job.ID,
		Prompt:       job.Prompt,
		Status:       string(job.Status),
		ErrorMessage: job.ErrorMessage,
		ErrorReason:  job.ErrorReason,
		CreatedAt:    job.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if !job.StartedAt.IsZero() {
		out.StartedAt = job.StartedAt.UTC().Format(time.RFC3339Nano)
	}
	if !job.CompletedAt.IsZero() {
		out.CompletedAt = job.CompletedAt.UTC().Format(time.RFC3339Nano)
	}
	if job.Result != nil {
		result := toImageGenerationResponse(job.Result)
		out.Result = &result
	}
	return out
}

func extensionForImageMimeType(mimeType string) string {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".png"
	}
}
