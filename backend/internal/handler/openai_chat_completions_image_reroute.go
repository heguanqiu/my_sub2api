package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const defaultOpenAIChatImagePrompt = "Generate an image."

type openAIChatImageRerouteRequest struct {
	Endpoint   string
	Body       []byte
	ImageCount int
}

type openAIChatImageCaptureWriter struct {
	gin.ResponseWriter
	header http.Header
	body   bytes.Buffer
	status int
	size   int
}

type openAIChatImageResult struct {
	URL           string
	B64JSON       string
	MIMEType      string
	RevisedPrompt string
}

func buildOpenAIImagesRequestFromChatCompletion(body []byte, model string) (*openAIChatImageRerouteRequest, error) {
	model = strings.TrimSpace(model)
	if !isOpenAIChatImageModel(model) {
		return nil, nil
	}

	prompt, images := extractOpenAIChatImagePromptAndImages(body)
	if prompt == "" {
		prompt = defaultOpenAIChatImagePrompt
	}

	payload := map[string]any{
		"model":           model,
		"prompt":          prompt,
		"n":               1,
		"response_format": "b64_json",
	}
	copyOpenAIChatImageBoolField(body, payload, "stream")
	copyOpenAIChatImageNumberField(body, payload, "n")
	copyOpenAIChatImageStringField(body, payload, "size")
	copyOpenAIChatImageStringField(body, payload, "quality")
	copyOpenAIChatImageStringField(body, payload, "background")
	copyOpenAIChatImageStringField(body, payload, "output_format")
	copyOpenAIChatImageStringField(body, payload, "moderation")
	copyOpenAIChatImageStringField(body, payload, "input_fidelity")
	copyOpenAIChatImageStringField(body, payload, "style")
	copyOpenAIChatImageNumberField(body, payload, "output_compression")
	copyOpenAIChatImageNumberField(body, payload, "partial_images")

	endpoint := EndpointImagesGenerations
	if len(images) > 0 {
		endpoint = EndpointImagesEdits
		imagePayload := make([]map[string]string, 0, len(images))
		for _, imageURL := range images {
			imagePayload = append(imagePayload, map[string]string{"image_url": imageURL})
		}
		payload["images"] = imagePayload
	}

	imageBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &openAIChatImageRerouteRequest{
		Endpoint:   endpoint,
		Body:       imageBody,
		ImageCount: len(images),
	}, nil
}

func rerouteOpenAIChatCompletionToImages(c *gin.Context, endpoint string, body []byte) {
	if c == nil || c.Request == nil {
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	c.Request.ContentLength = int64(len(body))
	c.Request.Header.Set("Content-Type", "application/json")
	if c.Request.URL != nil {
		c.Request.URL.Path = endpoint
		c.Request.RequestURI = endpoint
	}
	c.Set(ctxKeyInboundEndpoint, endpoint)
}

func (h *OpenAIGatewayHandler) serveOpenAIChatImageReroute(c *gin.Context, request *openAIChatImageRerouteRequest, model string, stream bool) {
	if request == nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "image request is empty")
		return
	}

	originalWriter := c.Writer
	capture := newOpenAIChatImageCaptureWriter(originalWriter)
	c.Writer = capture
	rerouteOpenAIChatCompletionToImages(c, request.Endpoint, request.Body)
	h.Images(c)
	c.Writer = originalWriter

	status := capture.Status()
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		relayOpenAIChatImageCapturedResponse(c, capture)
		return
	}

	results := extractOpenAIChatImageResults(capture.body.Bytes())
	if len(results) == 0 {
		relayOpenAIChatImageCapturedResponse(c, capture)
		return
	}
	content := buildOpenAIChatImageMarkdown(results)
	if stream {
		writeOpenAIChatImageCompletionStream(c, model, content)
		return
	}
	writeOpenAIChatImageCompletionJSON(c, model, content)
}

func newOpenAIChatImageCaptureWriter(rw gin.ResponseWriter) *openAIChatImageCaptureWriter {
	return &openAIChatImageCaptureWriter{
		ResponseWriter: rw,
		header:         make(http.Header),
	}
}

func (w *openAIChatImageCaptureWriter) Header() http.Header {
	return w.header
}

func (w *openAIChatImageCaptureWriter) WriteHeader(code int) {
	if w.Written() {
		return
	}
	w.status = code
}

func (w *openAIChatImageCaptureWriter) WriteHeaderNow() {
	if !w.Written() {
		w.status = http.StatusOK
	}
}

func (w *openAIChatImageCaptureWriter) Write(data []byte) (int, error) {
	w.WriteHeaderNow()
	n, err := w.body.Write(data)
	w.size += n
	return n, err
}

func (w *openAIChatImageCaptureWriter) WriteString(s string) (int, error) {
	w.WriteHeaderNow()
	n, err := w.body.WriteString(s)
	w.size += n
	return n, err
}

func (w *openAIChatImageCaptureWriter) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *openAIChatImageCaptureWriter) Size() int {
	return w.size
}

func (w *openAIChatImageCaptureWriter) Written() bool {
	return w.status != 0
}

func (w *openAIChatImageCaptureWriter) Flush() {}

func relayOpenAIChatImageCapturedResponse(c *gin.Context, capture *openAIChatImageCaptureWriter) {
	if c == nil || capture == nil {
		return
	}
	for key, values := range capture.header {
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}
	body := capture.body.Bytes()
	contentType := capture.header.Get("Content-Type")
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/json"
	}
	c.Data(capture.Status(), contentType, body)
}

func extractOpenAIChatImageResults(body []byte) []openAIChatImageResult {
	var results []openAIChatImageResult
	if len(bytes.TrimSpace(body)) == 0 {
		return results
	}
	if gjson.ValidBytes(body) {
		appendOpenAIChatImageResultsFromJSON(&results, gjson.ParseBytes(body))
		return dedupeOpenAIChatImageResults(results)
	}

	var dataLines []string
	flush := func() {
		if len(dataLines) == 0 {
			return
		}
		if len(dataLines) == 1 {
			appendOpenAIChatImageResultFromSSEData(&results, dataLines[0])
		} else {
			joined := strings.Join(dataLines, "\n")
			if gjson.Valid(joined) {
				appendOpenAIChatImageResultFromSSEData(&results, joined)
			} else {
				for _, line := range dataLines {
					appendOpenAIChatImageResultFromSSEData(&results, line)
				}
			}
		}
		dataLines = dataLines[:0]
	}

	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		if data, ok := extractOpenAIChatImageSSEDataLine(line); ok {
			dataLines = append(dataLines, data)
		}
	}
	flush()
	return dedupeOpenAIChatImageResults(results)
}

func appendOpenAIChatImageResultFromSSEData(results *[]openAIChatImageResult, data string) {
	data = strings.TrimSpace(data)
	if data == "" || data == "[DONE]" || !gjson.Valid(data) {
		return
	}
	appendOpenAIChatImageResultsFromJSON(results, gjson.Parse(data))
}

func appendOpenAIChatImageResultsFromJSON(results *[]openAIChatImageResult, root gjson.Result) {
	if results == nil || !root.Exists() {
		return
	}
	if root.IsArray() {
		root.ForEach(func(_, item gjson.Result) bool {
			appendOpenAIChatImageResultsFromJSON(results, item)
			return true
		})
		return
	}
	if !root.IsObject() {
		return
	}

	eventType := strings.TrimSpace(root.Get("type").String())
	if eventType == "image_generation.partial_image" {
		return
	}
	appendOpenAIChatImageResult(results, root)
	if data := root.Get("data"); data.IsArray() {
		data.ForEach(func(_, item gjson.Result) bool {
			appendOpenAIChatImageResult(results, item)
			return true
		})
	}
	appendOpenAIChatImageResultsFromJSON(results, root.Get("item"))
	appendOpenAIChatImageResultsFromJSON(results, root.Get("output"))
	appendOpenAIChatImageResultsFromJSON(results, root.Get("response.output"))
}

func appendOpenAIChatImageResult(results *[]openAIChatImageResult, item gjson.Result) {
	if results == nil || !item.Exists() || !item.IsObject() {
		return
	}
	itemType := strings.TrimSpace(item.Get("type").String())
	if itemType != "" && itemType != "image_generation_call" && itemType != "image_generation.completed" {
		return
	}

	imageURL := strings.TrimSpace(item.Get("url").String())
	b64JSON := strings.TrimSpace(item.Get("b64_json").String())
	if b64JSON == "" {
		b64JSON = strings.TrimSpace(item.Get("result").String())
	}
	if imageURL == "" && b64JSON == "" {
		return
	}
	*results = append(*results, openAIChatImageResult{
		URL:           imageURL,
		B64JSON:       b64JSON,
		MIMEType:      openAIChatImageMimeType(item.Get("output_format").String()),
		RevisedPrompt: strings.TrimSpace(item.Get("revised_prompt").String()),
	})
}

func extractOpenAIChatImageSSEDataLine(line string) (string, bool) {
	line = strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(line, "data:") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(line, "data:")), true
}

func dedupeOpenAIChatImageResults(results []openAIChatImageResult) []openAIChatImageResult {
	out := make([]openAIChatImageResult, 0, len(results))
	seen := make(map[string]struct{}, len(results))
	for _, result := range results {
		key := strings.TrimSpace(result.URL)
		if key == "" {
			key = strings.TrimSpace(result.B64JSON)
		}
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, result)
	}
	return out
}

func openAIChatImageMimeType(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "image/jpeg", "image/jpg":
		return "image/jpeg"
	case "image/png":
		return "image/png"
	case "image/webp":
		return "image/webp"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "webp":
		return "image/webp"
	default:
		return "image/png"
	}
}

func buildOpenAIChatImageMarkdown(results []openAIChatImageResult) string {
	lines := make([]string, 0, len(results)*3)
	for index, result := range results {
		src := strings.TrimSpace(result.URL)
		if src == "" && strings.TrimSpace(result.B64JSON) != "" {
			src = "data:" + openAIChatImageMimeType(result.MIMEType) + ";base64," + strings.TrimSpace(result.B64JSON)
		}
		if src == "" {
			continue
		}
		alt := "generated image"
		if len(results) > 1 {
			alt = fmt.Sprintf("generated image %d", index+1)
		}
		lines = append(lines, fmt.Sprintf("![%s](%s)", alt, src))
		downloadLabel := "下载图片"
		if len(results) > 1 {
			downloadLabel = fmt.Sprintf("下载图片 %d", index+1)
		}
		lines = append(lines, fmt.Sprintf("[%s](%s)", downloadLabel, src))
		if result.RevisedPrompt != "" {
			lines = append(lines, result.RevisedPrompt)
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n\n"))
}

func writeOpenAIChatImageCompletionJSON(c *gin.Context, model string, content string) {
	id := "chatcmpl-img-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []gin.H{
			{
				"index": 0,
				"message": gin.H{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": "stop",
			},
		},
	})
}

func writeOpenAIChatImageCompletionStream(c *gin.Context, model string, content string) {
	id := "chatcmpl-img-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	created := time.Now().Unix()
	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	writeOpenAIChatImageSSE(c, gin.H{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []gin.H{{"index": 0, "delta": gin.H{"role": "assistant"}, "finish_reason": nil}},
	})
	writeOpenAIChatImageSSE(c, gin.H{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []gin.H{{"index": 0, "delta": gin.H{"content": content}, "finish_reason": nil}},
	})
	writeOpenAIChatImageSSE(c, gin.H{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []gin.H{{"index": 0, "delta": gin.H{}, "finish_reason": "stop"}},
	})
	_, _ = io.WriteString(c.Writer, "data: [DONE]\n\n")
	c.Writer.Flush()
}

func writeOpenAIChatImageSSE(c *gin.Context, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_, _ = c.Writer.Write([]byte("data: "))
	_, _ = c.Writer.Write(raw)
	_, _ = c.Writer.Write([]byte("\n\n"))
	c.Writer.Flush()
}

func isOpenAIChatImageModel(model string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "gpt-image-")
}

func extractOpenAIChatImagePromptAndImages(body []byte) (string, []string) {
	var lastText []string
	var lastImages []string
	messages := gjson.GetBytes(body, "messages")
	if messages.IsArray() {
		messages.ForEach(func(_, message gjson.Result) bool {
			role := strings.ToLower(strings.TrimSpace(message.Get("role").String()))
			if role != "user" {
				return true
			}
			var textParts []string
			var imageURLs []string
			collectOpenAIChatImageContent(message.Get("content"), &textParts, &imageURLs)
			collectOpenAIChatImageContent(message.Get("files"), &textParts, &imageURLs)
			collectOpenAIChatImageContent(message.Get("attachments"), &textParts, &imageURLs)
			if normalizeOpenAIChatImagePrompt(strings.Join(textParts, "\n")) != "" || len(imageURLs) > 0 {
				lastText = textParts
				lastImages = imageURLs
			}
			return true
		})
	}

	if len(lastText) == 0 {
		collectOpenAIChatImageContent(gjson.GetBytes(body, "input"), &lastText, &lastImages)
		collectOpenAIChatImageContent(gjson.GetBytes(body, "prompt"), &lastText, &lastImages)
	}

	return normalizeOpenAIChatImagePrompt(strings.Join(lastText, "\n")), dedupeOpenAIChatImageURLs(lastImages)
}

func collectOpenAIChatImageContent(value gjson.Result, textParts *[]string, imageURLs *[]string) {
	switch {
	case !value.Exists():
		return
	case value.Type == gjson.String:
		addOpenAIChatImageText(textParts, value.String())
	case value.IsArray():
		value.ForEach(func(_, item gjson.Result) bool {
			collectOpenAIChatImageContent(item, textParts, imageURLs)
			return true
		})
	case value.IsObject():
		typ := strings.ToLower(strings.TrimSpace(value.Get("type").String()))
		addOpenAIChatImageURL(imageURLs, value.Get("image_url.url").String())
		addOpenAIChatImageURL(imageURLs, value.Get("image_url").String())
		if typ == "image" || typ == "input_image" || typ == "image_url" {
			addOpenAIChatImageURL(imageURLs, value.Get("url").String())
		}
		addOpenAIChatImageDataURL(imageURLs, value.Get("source.media_type").String(), value.Get("source.data").String())
		addOpenAIChatImageDataURL(imageURLs, value.Get("source.mediaType").String(), value.Get("source.data").String())
		addOpenAIChatImageDataURL(imageURLs, value.Get("media_type").String(), value.Get("data").String())
		addOpenAIChatImageDataURL(imageURLs, value.Get("mime_type").String(), value.Get("data").String())
		addOpenAIChatImageDataURL(imageURLs, value.Get("mimeType").String(), value.Get("data").String())
		addOpenAIChatImageURL(imageURLs, value.Get("download_url").String())

		switch typ {
		case "", "text", "input_text", "message":
			if value.Get("text").Exists() {
				addOpenAIChatImageText(textParts, value.Get("text").String())
			}
			if value.Get("content").Exists() {
				collectOpenAIChatImageContent(value.Get("content"), textParts, imageURLs)
			}
		case "image", "input_image", "image_url":
		default:
			if value.Get("text").Exists() {
				addOpenAIChatImageText(textParts, value.Get("text").String())
			}
			if value.Get("content").Exists() {
				collectOpenAIChatImageContent(value.Get("content"), textParts, imageURLs)
			}
		}
	}
}

func addOpenAIChatImageText(parts *[]string, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	*parts = append(*parts, text)
}

func addOpenAIChatImageURL(images *[]string, imageURL string) {
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return
	}
	lower := strings.ToLower(imageURL)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "data:image/") {
		*images = append(*images, imageURL)
	}
}

func addOpenAIChatImageDataURL(images *[]string, mimeType string, data string) {
	mimeType = strings.TrimSpace(mimeType)
	data = strings.TrimSpace(data)
	if mimeType == "" || data == "" {
		return
	}
	addOpenAIChatImageURL(images, "data:"+mimeType+";base64,"+data)
}

func normalizeOpenAIChatImagePrompt(prompt string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(prompt)), " ")
}

func dedupeOpenAIChatImageURLs(images []string) []string {
	out := make([]string, 0, len(images))
	seen := make(map[string]struct{}, len(images))
	for _, image := range images {
		image = strings.TrimSpace(image)
		if image == "" {
			continue
		}
		if _, ok := seen[image]; ok {
			continue
		}
		seen[image] = struct{}{}
		out = append(out, image)
	}
	return out
}

func copyOpenAIChatImageStringField(body []byte, payload map[string]any, field string) {
	value := strings.TrimSpace(gjson.GetBytes(body, field).String())
	if value != "" {
		payload[field] = value
	}
}

func copyOpenAIChatImageBoolField(body []byte, payload map[string]any, field string) {
	value := gjson.GetBytes(body, field)
	if value.Type == gjson.True || value.Type == gjson.False {
		payload[field] = value.Bool()
	}
}

func copyOpenAIChatImageNumberField(body []byte, payload map[string]any, field string) {
	value := gjson.GetBytes(body, field)
	if value.Type == gjson.Number {
		payload[field] = value.Int()
	}
}
