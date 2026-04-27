package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
)

// OpenAIImagesDirectResult is the captured upstream-compatible images response
// used by internal JWT-only features that should not write to the client directly.
type OpenAIImagesDirectResult struct {
	ForwardResult *OpenAIForwardResult
	StatusCode    int
	ContentType   string
	Body          []byte
}

// ForwardImagesDirect reuses the normal OpenAI images forwarding path while
// capturing its response body instead of writing it to the caller's HTTP response.
func (s *OpenAIGatewayService) ForwardImagesDirect(
	ctx context.Context,
	account *Account,
	body []byte,
	parsed *OpenAIImagesRequest,
	channelMappedModel string,
) (*OpenAIImagesDirectResult, error) {
	if s == nil {
		return nil, fmt.Errorf("openai gateway service is not configured")
	}
	if parsed == nil {
		return nil, fmt.Errorf("parsed images request is required")
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	path := openAIImagesGenerationsEndpoint
	if parsed.IsEdits() {
		path = openAIImagesEditsEndpoint
	}
	req := httptest.NewRequest("POST", path, bytes.NewReader(body))
	if parsed.ContentType != "" {
		req.Header.Set("Content-Type", parsed.ContentType)
	}
	c.Request = req.WithContext(ctx)

	forwardResult, err := s.ForwardImages(ctx, c, account, body, parsed, channelMappedModel)
	responseBody := append([]byte(nil), recorder.Body.Bytes()...)
	statusCode := recorder.Code
	if statusCode == 0 {
		statusCode = 200
	}
	if err != nil {
		return &OpenAIImagesDirectResult{
			ForwardResult: forwardResult,
			StatusCode:    statusCode,
			ContentType:   recorder.Header().Get("Content-Type"),
			Body:          responseBody,
		}, err
	}
	if statusCode >= 400 {
		return &OpenAIImagesDirectResult{
			ForwardResult: forwardResult,
			StatusCode:    statusCode,
			ContentType:   recorder.Header().Get("Content-Type"),
			Body:          responseBody,
		}, fmt.Errorf("openai images upstream returned status %d", statusCode)
	}

	return &OpenAIImagesDirectResult{
		ForwardResult: forwardResult,
		StatusCode:    statusCode,
		ContentType:   recorder.Header().Get("Content-Type"),
		Body:          responseBody,
	}, nil
}
