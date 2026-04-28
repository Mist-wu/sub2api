package service

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
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
