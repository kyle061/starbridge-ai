package service

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	CodexImageDownloadPath       = "/generated-images/"
	codexImageDeliveryContextKey = "codex_image_delivery"
	codexImageMaxBytes           = 32 << 20
)

var codexImageFileName = regexp.MustCompile(`^[a-f0-9]{64}\.(png|jpg|webp)$`)

// CodexImageStore keeps private image artifacts behind random, expiring bearer URLs.
// It never downloads upstream URLs or persists request prompts/account credentials.
type CodexImageStore struct {
	directory string
	ttl       time.Duration
	maxBytes  int64
	now       func() time.Time
	mu        sync.Mutex
}

func NewCodexImageStore(directory string) *CodexImageStore {
	return &CodexImageStore{directory: directory, ttl: 24 * time.Hour, maxBytes: 1 << 30, now: time.Now}
}

func (s *CodexImageStore) save(encoded string) (string, error) {
	if len(encoded) > base64.StdEncoding.EncodedLen(codexImageMaxBytes)+128 {
		return "", errors.New("generated image exceeds size limit")
	}
	if strings.HasPrefix(encoded, "data:") {
		header, body, ok := strings.Cut(encoded, ",")
		if !ok || !strings.HasSuffix(header, ";base64") {
			return "", errors.New("invalid image data URL")
		}
		encoded = body
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", errors.New("invalid generated image base64")
	}
	if len(data) > codexImageMaxBytes {
		return "", errors.New("generated image exceeds size limit")
	}
	ext := ""
	if cfg, format, err := image.DecodeConfig(bytes.NewReader(data)); err == nil && cfg.Width > 0 && cfg.Height > 0 {
		switch format {
		case "png":
			ext = "png"
		case "jpeg":
			ext = "jpg"
		case "webp":
			ext = "webp"
		}
	} else if _, _, ok := detectOpenAIWebPDimensions(data); ok {
		ext = "webp"
	}
	if ext == "" {
		return "", errors.New("invalid generated image format")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.directory, 0700); err != nil {
		return "", err
	}
	entries, err := os.ReadDir(s.directory)
	if err != nil {
		return "", err
	}
	var total int64
	files := 0
	for _, entry := range entries {
		if !codexImageFileName.MatchString(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return "", err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if !s.now().Before(info.ModTime().Add(s.ttl)) {
			if err := os.Remove(filepath.Join(s.directory, entry.Name())); err != nil {
				return "", err
			}
			continue
		}
		total += info.Size()
		files++
	}
	if total+int64(len(data)) > s.maxBytes || files >= 4096 {
		return "", errors.New("generated image storage is full")
	}
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", err
	}
	name := hex.EncodeToString(token[:]) + "." + ext
	file, err := os.OpenFile(filepath.Join(s.directory, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		_ = os.Remove(filepath.Join(s.directory, name))
		return "", err
	}
	return name, nil
}

func (s *CodexImageStore) Open(name string) (*os.File, string, error) {
	if s == nil || !codexImageFileName.MatchString(name) {
		return nil, "", os.ErrNotExist
	}
	filePath := filepath.Join(s.directory, name)
	info, err := os.Lstat(filePath)
	if err != nil || !info.Mode().IsRegular() {
		return nil, "", os.ErrNotExist
	}
	if !s.now().Before(info.ModTime().Add(s.ttl)) {
		return nil, "", os.ErrNotExist
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", err
	}
	contentType := map[string]string{".png": "image/png", ".jpg": "image/jpeg", ".webp": "image/webp"}[filepath.Ext(name)]
	return file, contentType, nil
}

// CodexImageDelivery adds ordinary assistant text for clients which retain native
// image_generation_call items in their transcript but do not render those items.
type CodexImageDelivery struct {
	store     *CodexImageStore
	baseURL   string
	mu        sync.Mutex
	seen      map[string]bool
	links     []string
	failed    bool
	emitted   bool
	maxIndex  int64
	doneItems *responsesStreamOutputItems
	doneBytes int
}

func NewCodexImageDelivery(store *CodexImageStore, baseURL string) *CodexImageDelivery {
	d := &CodexImageDelivery{store: store, baseURL: strings.TrimRight(baseURL, "/")}
	d.reset()
	return d
}

func SetCodexImageDelivery(c *gin.Context, delivery *CodexImageDelivery) {
	c.Set(codexImageDeliveryContextKey, delivery)
}

func codexImageDeliveryFromContext(c *gin.Context) *CodexImageDelivery {
	if c == nil {
		return nil
	}
	value, _ := c.Get(codexImageDeliveryContextKey)
	delivery, _ := value.(*CodexImageDelivery)
	return delivery
}

func (d *CodexImageDelivery) reset() {
	d.seen = make(map[string]bool)
	d.links = nil
	d.failed, d.emitted = false, false
	d.maxIndex = -1
	d.doneItems = newResponsesStreamOutputItems()
	d.doneBytes = 0
}

func (d *CodexImageDelivery) observe(item gjson.Result) {
	if item.Get("type").String() != "image_generation_call" {
		return
	}
	result := strings.TrimSpace(item.Get("result").String())
	if result == "" {
		return
	}
	id := item.Get("id").String()
	if id == "" {
		sum := sha256.Sum256([]byte(result))
		id = hex.EncodeToString(sum[:])
	}
	if d.seen[id] {
		return
	}
	d.seen[id] = true
	if len(d.seen) > 16 {
		d.failed = true
		return
	}
	name, err := d.store.save(result)
	if err != nil {
		d.failed = true
		slog.Warn("codex image delivery could not save generated image", "error", err)
		return
	}
	d.links = append(d.links, d.baseURL+CodexImageDownloadPath+name)
}

func (d *CodexImageDelivery) message() map[string]any {
	var text strings.Builder
	for i, link := range d.links {
		fmt.Fprintf(&text, "![Generated image %d](%s)\n\n[Download image %d](%s?download=1)\n\n", i+1, link, i+1, link)
	}
	if len(d.links) > 0 {
		text.WriteString("Image links expire in 24 hours.\n")
	}
	if d.failed {
		text.WriteString("An image was generated, but its download file could not be saved. The original image result is still in the API response; do not regenerate it just to recover the file.\n")
	}
	if text.Len() == 0 {
		return nil
	}
	return map[string]any{"id": "msg_image_" + generateRequestID(), "type": "message", "role": "assistant", "status": "completed", "phase": "final_answer", "content": []any{map[string]any{"type": "output_text", "text": text.String(), "annotations": []any{}}}}
}

func (d *CodexImageDelivery) JSON(body []byte) []byte {
	if d == nil || !gjson.ValidBytes(body) {
		return body
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.reset()
	output := gjson.GetBytes(body, "output")
	if !output.IsArray() {
		return body
	}
	for _, item := range output.Array() {
		d.observe(item)
	}
	message := d.message()
	if message == nil {
		return body
	}
	updated, err := sjson.SetBytes(body, "output.-1", message)
	if err != nil {
		return body
	}
	return updated
}

// WriteEvent preserves native image items and usage. Only completed artifacts
// receive links; previews are ignored and output/terminal duplicates are deduped.
func (d *CodexImageDelivery) WriteEvent(payload []byte, write func([]byte) error) error {
	if d == nil || !gjson.ValidBytes(payload) {
		return write(payload)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	eventType := gjson.GetBytes(payload, "type").String()
	if eventType == "response.created" {
		d.reset()
	}
	if index := gjson.GetBytes(payload, "output_index"); index.Exists() && index.Int() > d.maxIndex {
		d.maxIndex = index.Int()
	}
	if eventType == "response.output_item.done" {
		d.doneBytes += len(payload)
		if d.doneBytes > 64<<20 {
			d.doneItems = nil
		} else {
			d.doneItems.Observe(payload)
		}
		d.observe(gjson.GetBytes(payload, "item"))
	}
	switch eventType {
	case "response.completed", "response.done", "response.failed", "response.incomplete":
	default:
		return write(payload)
	}
	if d.emitted {
		return write(payload)
	}
	output := gjson.GetBytes(payload, "response.output")
	if output.IsArray() {
		for _, item := range output.Array() {
			d.observe(item)
		}
		if index := int64(len(output.Array()) - 1); index > d.maxIndex {
			d.maxIndex = index
		}
	}
	message := d.message()
	if message == nil {
		return write(payload)
	}
	if rebuilt, changed := normalizeResponsesStreamingTerminalOutput(payload, nil, d.doneItems, nil); changed {
		payload = rebuilt
	}
	if !gjson.GetBytes(payload, "response.output").IsArray() {
		var err error
		payload, err = sjson.SetBytes(payload, "response.output", []any{})
		if err != nil {
			return err
		}
	}
	part := message["content"].([]any)[0].(map[string]any)
	id := message["id"]
	index := d.maxIndex + 1
	added := map[string]any{"id": id, "type": "message", "role": "assistant", "status": "in_progress", "phase": "final_answer", "content": []any{}}
	events := []map[string]any{
		{"type": "response.output_item.added", "output_index": index, "item": added},
		{"type": "response.content_part.added", "output_index": index, "item_id": id, "content_index": 0, "part": map[string]any{"type": "output_text", "text": "", "annotations": []any{}}},
		{"type": "response.output_text.delta", "output_index": index, "item_id": id, "content_index": 0, "delta": part["text"]},
		{"type": "response.output_text.done", "output_index": index, "item_id": id, "content_index": 0, "text": part["text"]},
		{"type": "response.content_part.done", "output_index": index, "item_id": id, "content_index": 0, "part": part},
		{"type": "response.output_item.done", "output_index": index, "item": message},
	}
	sequence := gjson.GetBytes(payload, "sequence_number")
	for i, event := range events {
		if sequence.Exists() {
			event["sequence_number"] = sequence.Int() + int64(i)
		}
		encoded, err := json.Marshal(event)
		if err != nil {
			return err
		}
		if err := write(encoded); err != nil {
			return err
		}
	}
	d.emitted = true
	updated, err := sjson.SetBytes(payload, "response.output.-1", message)
	if err != nil {
		return err
	}
	if sequence.Exists() {
		updated, err = sjson.SetBytes(updated, "sequence_number", sequence.Int()+int64(len(events)))
		if err != nil {
			return err
		}
	}
	return write(updated)
}
