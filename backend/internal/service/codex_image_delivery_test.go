package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func codexDeliveryPNG(t *testing.T) []byte {
	t.Helper()
	picture := image.NewRGBA(image.Rect(0, 0, 3, 2))
	picture.Set(1, 1, color.RGBA{R: 220, G: 80, B: 20, A: 255})
	var out bytes.Buffer
	require.NoError(t, png.Encode(&out, picture))
	return out.Bytes()
}

func codexDeliveryItem(t *testing.T) map[string]any {
	return map[string]any{"type": "image_generation_call", "id": "ig_test", "status": "completed", "result": base64.StdEncoding.EncodeToString(codexDeliveryPNG(t))}
}

func codexDeliveryEvent(t *testing.T, value any) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	require.NoError(t, err)
	return body
}

func TestCodexImageDeliveryPreservesPayloadAndDeduplicatesAcrossEvents(t *testing.T) {
	store := NewCodexImageStore(t.TempDir())
	d := NewCodexImageDelivery(store, "https://gateway.test")
	item := codexDeliveryItem(t)
	done := codexDeliveryEvent(t, map[string]any{"type": "response.output_item.done", "output_index": 0, "sequence_number": 2, "item": item})
	terminal := codexDeliveryEvent(t, map[string]any{"type": "response.completed", "sequence_number": 3, "response": map[string]any{"id": "resp_test", "output": []any{item}, "usage": map[string]any{"input_tokens": 9, "output_tokens": 4}}})
	var events [][]byte
	write := func(body []byte) error { events = append(events, bytes.Clone(body)); return nil }
	require.NoError(t, d.WriteEvent(done, write))
	require.NoError(t, d.WriteEvent(terminal, write))
	require.Equal(t, done, events[0])
	require.Len(t, events, 8)
	for i, event := range events {
		require.EqualValues(t, i+2, gjson.GetBytes(event, "sequence_number").Int())
	}
	final := events[len(events)-1]
	require.Equal(t, item["result"], gjson.GetBytes(final, "response.output.0.result").String())
	require.EqualValues(t, 9, gjson.GetBytes(final, "response.usage.input_tokens").Int())
	require.Equal(t, "assistant", gjson.GetBytes(final, "response.output.1.role").String())
	require.Equal(t, "final_answer", gjson.GetBytes(final, "response.output.1.phase").String())
	text := gjson.GetBytes(final, "response.output.1.content.0.text").String()
	require.Contains(t, text, "![Generated image 1](https://gateway.test/generated-images/")
	require.Equal(t, text, gjson.GetBytes(events[3], "delta").String())
	require.NoError(t, d.WriteEvent(terminal, write))
	files, err := os.ReadDir(store.directory)
	require.NoError(t, err)
	require.Len(t, files, 1)
	file, mime, err := store.Open(files[0].Name())
	require.NoError(t, err)
	defer file.Close()
	data, err := io.ReadAll(file)
	require.NoError(t, err)
	require.Equal(t, codexDeliveryPNG(t), data)
	require.Equal(t, "image/png", mime)
	decoded, err := png.Decode(bytes.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, image.Rect(0, 0, 3, 2), decoded.Bounds())
}

func TestCodexImageDeliveryHandlesEmptyTerminalAndResetsBetweenTurns(t *testing.T) {
	d := NewCodexImageDelivery(NewCodexImageStore(t.TempDir()), "https://gateway.test")
	var events [][]byte
	write := func(body []byte) error { events = append(events, bytes.Clone(body)); return nil }
	require.NoError(t, d.WriteEvent(codexDeliveryEvent(t, map[string]any{"type": "response.output_item.done", "output_index": 0, "item": codexDeliveryItem(t)}), write))
	empty := []byte(`{"type":"response.completed","response":{"output":[],"usage":{"output_tokens":2}}}`)
	require.NoError(t, d.WriteEvent(empty, write))
	final := events[len(events)-1]
	require.Len(t, gjson.GetBytes(final, "response.output").Array(), 2)
	require.Equal(t, "image_generation_call", gjson.GetBytes(final, "response.output.0.type").String())
	require.NoError(t, d.WriteEvent([]byte(`{"type":"response.created","response":{"id":"resp_next"}}`), write))
	count := len(events)
	require.NoError(t, d.WriteEvent(empty, write))
	require.Len(t, events, count+1)
	require.Equal(t, empty, events[len(events)-1])
}

func TestCodexImageDeliverySkipsPreviewsAndPropagatesWriteErrors(t *testing.T) {
	d := NewCodexImageDelivery(NewCodexImageStore(t.TempDir()), "https://gateway.test")
	preview := []byte(`{"type":"response.image_generation_call.partial_image","partial_image_b64":"preview"}`)
	require.NoError(t, d.WriteEvent(preview, func(payload []byte) error { require.Equal(t, preview, payload); return nil }))
	require.Empty(t, d.links)
	terminal := codexDeliveryEvent(t, map[string]any{"type": "response.done", "response": map[string]any{"output": []any{codexDeliveryItem(t)}}})
	want := errors.New("client closed")
	require.ErrorIs(t, d.WriteEvent(terminal, func([]byte) error { return want }), want)
}

func TestCodexImageStoreRejectsInvalidExpiredAndOverBudgetFiles(t *testing.T) {
	store := NewCodexImageStore(t.TempDir())
	_, err := store.save("not-base64")
	require.Error(t, err)
	_, err = store.save(base64.StdEncoding.EncodeToString([]byte("not an image")))
	require.Error(t, err)
	_, _, err = store.Open("../config.yaml")
	require.ErrorIs(t, err, os.ErrNotExist)
	encoded := base64.StdEncoding.EncodeToString(codexDeliveryPNG(t))
	name, err := store.save(encoded)
	require.NoError(t, err)
	_, _, err = store.Open(strings.Repeat("0", 64) + ".png")
	require.ErrorIs(t, err, os.ErrNotExist)
	store.maxBytes = 1
	_, err = store.save(encoded)
	require.ErrorContains(t, err, "storage is full")
	store.maxBytes = 1 << 30
	store.now = func() time.Time { return time.Now().Add(25 * time.Hour) }
	_, _, err = store.Open(name)
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = store.save(encoded)
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(store.directory, name))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestCodexImageDeliveryStorageFailureKeepsOriginalResult(t *testing.T) {
	store := NewCodexImageStore(t.TempDir())
	store.maxBytes = 1
	d := NewCodexImageDelivery(store, "https://gateway.test")
	body := codexDeliveryEvent(t, map[string]any{"output": []any{codexDeliveryItem(t)}})
	result := d.JSON(body)
	require.Equal(t, gjson.GetBytes(body, "output.0.result").String(), gjson.GetBytes(result, "output.0.result").String())
	require.Contains(t, gjson.GetBytes(result, "output.1.content.0.text").String(), "could not be saved")
	require.NotContains(t, string(result), "https://gateway.test")
}

func TestCodexImageDeliveryWebSocketWritesVisibleMessage(t *testing.T) {
	store := NewCodexImageStore(t.TempDir())
	payload := codexDeliveryEvent(t, map[string]any{"type": "response.completed", "response": map[string]any{"output": []any{codexDeliveryItem(t)}}})
	serverErr := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, nil)
		if err != nil {
			serverErr <- err
			return
		}
		defer conn.CloseNow()
		frame := &openAIWSClientFrameConn{conn: conn, imageDelivery: NewCodexImageDelivery(store, "https://gateway.test")}
		serverErr <- frame.WriteFrame(r.Context(), coderws.MessageText, payload)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := coderws.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), nil)
	require.NoError(t, err)
	defer conn.CloseNow()
	var text string
	for {
		_, body, err := conn.Read(ctx)
		require.NoError(t, err)
		if gjson.GetBytes(body, "type").String() == "response.output_text.delta" {
			text += gjson.GetBytes(body, "delta").String()
		}
		if gjson.GetBytes(body, "type").String() == "response.completed" {
			break
		}
	}
	require.Contains(t, text, "![Generated image 1]")
	require.NoError(t, <-serverErr)
}
