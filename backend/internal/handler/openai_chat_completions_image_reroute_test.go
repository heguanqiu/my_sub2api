package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestBuildOpenAIImagesRequestFromChatCompletion_Generation(t *testing.T) {
	body := []byte(`{
		"model":"gpt-image-2",
		"stream":true,
		"messages":[{"role":"user","content":"画一只坐在屋顶上的猫"}]
	}`)

	request, err := buildOpenAIImagesRequestFromChatCompletion(body, "gpt-image-2")
	require.NoError(t, err)
	require.NotNil(t, request)
	require.Equal(t, EndpointImagesGenerations, request.Endpoint)
	require.Equal(t, 0, request.ImageCount)
	require.Equal(t, "gpt-image-2", gjson.GetBytes(request.Body, "model").String())
	require.Equal(t, "画一只坐在屋顶上的猫", gjson.GetBytes(request.Body, "prompt").String())
	require.True(t, gjson.GetBytes(request.Body, "stream").Bool())
	require.Equal(t, "b64_json", gjson.GetBytes(request.Body, "response_format").String())
	require.False(t, gjson.GetBytes(request.Body, "images").Exists())
}

func TestBuildOpenAIImagesRequestFromChatCompletion_Edit(t *testing.T) {
	body := []byte(`{
		"model":"gpt-image-2",
		"size":"1024x1024",
		"messages":[
			{"role":"user","content":"旧提示"},
			{"role":"user","content":[
				{"type":"text","text":"把背景换成夜景"},
				{"type":"image_url","image_url":{"url":"https://example.com/source.png"}}
			]}
		]
	}`)

	request, err := buildOpenAIImagesRequestFromChatCompletion(body, "gpt-image-2")
	require.NoError(t, err)
	require.NotNil(t, request)
	require.Equal(t, EndpointImagesEdits, request.Endpoint)
	require.Equal(t, 1, request.ImageCount)
	require.Equal(t, "把背景换成夜景", gjson.GetBytes(request.Body, "prompt").String())
	require.Equal(t, "1024x1024", gjson.GetBytes(request.Body, "size").String())
	require.Equal(t, "https://example.com/source.png", gjson.GetBytes(request.Body, "images.0.image_url").String())
}

func TestBuildOpenAIImagesRequestFromChatCompletion_NonImageModel(t *testing.T) {
	request, err := buildOpenAIImagesRequestFromChatCompletion([]byte(`{"model":"gpt-5.4"}`), "gpt-5.4")
	require.NoError(t, err)
	require.Nil(t, request)
}

func TestBuildOpenAIImagesRequestFromChatCompletion_ForcesBase64ForChatDisplay(t *testing.T) {
	body := []byte(`{
		"model":"gpt-image-2",
		"response_format":"url",
		"messages":[{"role":"user","content":"画一张海报"}]
	}`)

	request, err := buildOpenAIImagesRequestFromChatCompletion(body, "gpt-image-2")
	require.NoError(t, err)
	require.NotNil(t, request)
	require.Equal(t, "b64_json", gjson.GetBytes(request.Body, "response_format").String())
}

func TestExtractOpenAIChatImageResultsFromJSONAndMarkdownDownload(t *testing.T) {
	body := []byte(`{"data":[{"b64_json":"aGVsbG8=","revised_prompt":"draw a cat"}]}`)

	results := extractOpenAIChatImageResults(body)
	require.Len(t, results, 1)
	require.Equal(t, "aGVsbG8=", results[0].B64JSON)

	markdown := buildOpenAIChatImageMarkdown(results)
	require.Contains(t, markdown, "![generated image](data:image/png;base64,aGVsbG8=)")
	require.Contains(t, markdown, "[下载图片](data:image/png;base64,aGVsbG8=)")
	require.Contains(t, markdown, "draw a cat")
}

func TestExtractOpenAIChatImageResultsFromStreamingSSE(t *testing.T) {
	body := []byte(
		"data: {\"type\":\"image_generation.partial_image\",\"b64_json\":\"partial\"}\n\n" +
			"data: {\"type\":\"image_generation.completed\",\"b64_json\":\"ZmluYWw=\",\"output_format\":\"webp\"}\n\n" +
			"data: [DONE]\n\n",
	)

	results := extractOpenAIChatImageResults(body)
	require.Len(t, results, 1)
	require.Equal(t, "ZmluYWw=", results[0].B64JSON)

	markdown := buildOpenAIChatImageMarkdown(results)
	require.Contains(t, markdown, "data:image/webp;base64,ZmluYWw=")
	require.NotContains(t, markdown, "partial")
}

func TestRerouteOpenAIChatCompletionToImages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set(ctxKeyInboundEndpoint, EndpointChatCompletions)

	body := []byte(`{"model":"gpt-image-2","prompt":"draw"}`)
	rerouteOpenAIChatCompletionToImages(c, EndpointImagesGenerations, body)

	require.Equal(t, EndpointImagesGenerations, c.Request.URL.Path)
	require.Equal(t, EndpointImagesGenerations, c.Request.RequestURI)
	require.Equal(t, EndpointImagesGenerations, GetInboundEndpoint(c))
	require.Equal(t, int64(len(body)), c.Request.ContentLength)
	require.Equal(t, "application/json", c.Request.Header.Get("Content-Type"))
	gotBody, err := io.ReadAll(c.Request.Body)
	require.NoError(t, err)
	require.JSONEq(t, string(body), string(gotBody))
}
