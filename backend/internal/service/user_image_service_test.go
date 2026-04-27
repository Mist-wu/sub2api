package service

import (
	"encoding/base64"
	"strings"
	"testing"

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

func TestExtractUserImageFromOpenAIResponse(t *testing.T) {
	rawImage := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	body := []byte(`{"output_format":"png","data":[{"b64_json":"` + base64.StdEncoding.EncodeToString(rawImage) + `","revised_prompt":"soft light"}]}`)

	imageData, mimeType, revisedPrompt, err := extractUserImageFromOpenAIResponse(body)
	require.NoError(t, err)
	require.Equal(t, rawImage, imageData)
	require.Equal(t, "image/png", mimeType)
	require.Equal(t, "soft light", revisedPrompt)
}
