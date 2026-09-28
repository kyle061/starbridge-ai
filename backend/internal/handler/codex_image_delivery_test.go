package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func imageDeliveryResponse(t *testing.T, picture []byte) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{"id": "resp_image", "output": []any{map[string]any{"type": "image_generation_call", "id": "ig_delivery", "status": "completed", "result": base64.StdEncoding.EncodeToString(picture)}}, "usage": map[string]any{"input_tokens": 7, "output_tokens": 3}})
	require.NoError(t, err)
	return body
}

func imageDeliveryFixture(t *testing.T) []byte {
	t.Helper()
	var out bytes.Buffer
	require.NoError(t, png.Encode(&out, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	return out.Bytes()
}

func replayCodexImageDelivery(t *testing.T, picture []byte, stream bool, prefix string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	store := service.NewCodexImageStore(t.TempDir())
	r := gin.New()
	r.GET(service.CodexImageDownloadPath+":name", CodexImageDownload(store))
	r.HEAD(service.CodexImageDownloadPath+":name", CodexImageDownload(store))
	r.Use(CodexImageDeliveryMiddleware(store, &config.Config{}))
	response := imageDeliveryResponse(t, picture)
	r.POST(prefix+"/responses", func(c *gin.Context) {
		if !stream {
			c.Header("Content-Length", strconv.Itoa(len(response)))
			c.Data(200, "application/json", response)
			return
		}
		c.Header("Content-Type", "text/event-stream")
		body := "data: {\"type\":\"response.completed\",\"sequence_number\":12,\"response\":" + string(response) + "}\r\n\r\ndata: [DONE]\r\n\r\n"
		// Network chunk boundaries must not split or lose the base64 image.
		for start := 0; start < len(body); start += 137 {
			end := start + 137
			if end > len(body) {
				end = len(body)
			}
			_, err := c.Writer.Write([]byte(body[start:end]))
			require.NoError(t, err)
			c.Writer.Flush()
		}
	})
	req := httptest.NewRequest(http.MethodPost, "https://gateway.test"+prefix+"/responses", nil)
	req.Header.Set("User-Agent", "codex_cli_rs/0.144.1")
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusOK, recorder.Code)
	var text string
	if stream {
		for _, line := range strings.Split(recorder.Body.String(), "\n") {
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
			if gjson.Get(payload, "type").String() == "response.output_text.delta" {
				text += gjson.Get(payload, "delta").String()
			}
			if gjson.Get(payload, "type").String() == "response.completed" {
				require.Equal(t, base64.StdEncoding.EncodeToString(picture), gjson.Get(payload, "response.output.0.result").String())
				require.EqualValues(t, 7, gjson.Get(payload, "response.usage.input_tokens").Int())
			}
		}
		require.Contains(t, recorder.Body.String(), "data: [DONE]")
	} else {
		text = gjson.GetBytes(recorder.Body.Bytes(), "output.1.content.0.text").String()
		require.Equal(t, base64.StdEncoding.EncodeToString(picture), gjson.GetBytes(recorder.Body.Bytes(), "output.0.result").String())
	}
	match := regexp.MustCompile(`!\[Generated image 1\]\((https://gateway\.test/images/generated/[a-f0-9]+\.png)\)`).FindStringSubmatch(text)
	require.Len(t, match, 2)
	link, err := url.Parse(match[1])
	require.NoError(t, err)
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		download := httptest.NewRecorder()
		r.ServeHTTP(download, httptest.NewRequest(method, link.String(), nil))
		require.Equal(t, http.StatusOK, download.Code)
		require.Equal(t, "image/png", download.Header().Get("Content-Type"))
		require.Equal(t, "private, no-store", download.Header().Get("Cache-Control"))
		if method == http.MethodGet {
			require.Equal(t, picture, download.Body.Bytes())
			_, err := png.Decode(bytes.NewReader(download.Body.Bytes()))
			require.NoError(t, err)
		} else {
			require.Empty(t, download.Body.Bytes())
		}
	}
	download := httptest.NewRecorder()
	r.ServeHTTP(download, httptest.NewRequest(http.MethodGet, link.String()+"?download=1", nil))
	require.Contains(t, download.Header().Get("Content-Disposition"), "attachment")
	missing := httptest.NewRecorder()
	r.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "https://gateway.test/images/generated/"+strings.Repeat("0", 64)+".png", nil))
	require.Equal(t, http.StatusNotFound, missing.Code)
}

func TestCodexImageDeliveryHTTPJSONAndSSE(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, prefix := range []string{"/v1", "/openai/v1"} {
			t.Run(strconv.FormatBool(stream)+prefix, func(t *testing.T) { replayCodexImageDelivery(t, imageDeliveryFixture(t), stream, prefix) })
		}
	}
}

func TestCodexImageDeliveryRealImageReplay(t *testing.T) {
	file := os.Getenv("CODEX_IMAGE_REPLAY_FILE")
	if file == "" {
		t.Skip("set CODEX_IMAGE_REPLAY_FILE to replay a locally recovered image")
	}
	picture, err := os.ReadFile(file)
	require.NoError(t, err)
	replayCodexImageDelivery(t, picture, false, "/v1")
	replayCodexImageDelivery(t, picture, true, "/v1")
}

func TestCodexImageDeliveryLeavesOtherClientsAndTextUnchanged(t *testing.T) {
	for _, test := range []struct {
		name, path, userAgent, body string
		status                      int
	}{
		{"standard-client", "/v1/responses", "OpenAI/Python", string(imageDeliveryResponse(t, imageDeliveryFixture(t))), 200},
		{"text", "/v1/responses", "codex_cli_rs/0.144.1", `{"output":[{"type":"message","content":[{"type":"output_text","text":"hello"}]}]}`, 200},
		{"compact", "/v1/responses/compact", "codex_cli_rs/0.144.1", `{"output":[{"type":"compaction","encrypted_content":"secret"}]}`, 200},
		{"error", "/v1/responses", "codex_cli_rs/0.144.1", `{"error":{"message":"no account"}}`, 503},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			r := gin.New()
			r.Use(CodexImageDeliveryMiddleware(service.NewCodexImageStore(dir), &config.Config{}))
			r.POST(test.path, func(c *gin.Context) { c.Data(test.status, "application/json", []byte(test.body)) })
			req := httptest.NewRequest(http.MethodPost, "https://gateway.test"+test.path, nil)
			req.Header.Set("User-Agent", test.userAgent)
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, req)
			require.Equal(t, test.status, recorder.Code)
			require.Equal(t, test.body, recorder.Body.String())
			files, err := os.ReadDir(dir)
			require.NoError(t, err)
			require.Empty(t, files)
		})
	}
}
