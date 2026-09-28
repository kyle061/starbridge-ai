package handler

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// CodexImageDeliveryMiddleware covers HTTP and WS Responses entrypoints, including
// provider aliases. Other clients and compact/counting requests remain byte-identical.
func CodexImageDeliveryMiddleware(store *service.CodexImageStore, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.HasSuffix(strings.TrimRight(c.Request.URL.Path, "/"), "/responses") ||
			!openai.IsCodexOfficialClientByHeaders(c.GetHeader("User-Agent"), c.GetHeader("originator")) {
			c.Next()
			return
		}
		baseURL := ""
		if cfg != nil {
			baseURL = strings.TrimRight(cfg.Server.FrontendURL, "/")
		}
		if baseURL == "" {
			scheme := "https"
			if c.Request.TLS == nil && c.GetHeader("X-Forwarded-Proto") != "https" {
				scheme = "http"
			}
			baseURL = scheme + "://" + c.Request.Host
		}
		parsed, err := url.Parse(baseURL)
		if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			c.Next()
			return
		}
		delivery := service.NewCodexImageDelivery(store, baseURL)
		service.SetCodexImageDelivery(c, delivery)
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}
		writer := &codexImageDeliveryWriter{ResponseWriter: c.Writer, delivery: delivery}
		c.Writer = writer
		defer writer.finish()
		c.Next()
	}
}

func CodexImageDownload(store *service.CodexImageStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		file, contentType, err := store.Open(c.Param("name"))
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Header("Content-Type", contentType)
		c.Header("Cache-Control", "private, no-store")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("X-Content-Type-Options", "nosniff")
		disposition := "inline"
		if c.Query("download") == "1" {
			disposition = "attachment"
		}
		c.Header("Content-Disposition", disposition+`; filename="generated`+filepath.Ext(c.Param("name"))+`"`)
		http.ServeContent(c.Writer, c.Request, "generated", info.ModTime(), file)
	}
}

type codexImageDeliveryWriter struct {
	gin.ResponseWriter
	delivery *service.CodexImageDelivery
	pending  []byte
	bypass   bool
	err      error
}

func (w *codexImageDeliveryWriter) WriteHeader(status int) {
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(status)
}

func (w *codexImageDeliveryWriter) WriteHeaderNow() {
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeaderNow()
}

func (w *codexImageDeliveryWriter) WriteString(value string) (int, error) {
	return w.Write([]byte(value))
}

func (w *codexImageDeliveryWriter) Write(data []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	contentType := w.Header().Get("Content-Type")
	stream := strings.Contains(contentType, "text/event-stream")
	if w.bypass || w.Status() >= 400 || (!stream && !strings.Contains(contentType, "application/json")) {
		return w.ResponseWriter.Write(data)
	}
	w.Header().Del("Content-Length")
	w.pending = append(w.pending, data...)
	// Stop buffering malformed/oversized responses without truncating their original data.
	if len(w.pending) > 64<<20 {
		_, w.err = w.ResponseWriter.Write(w.pending)
		w.pending = nil
		w.bypass = true
	} else if stream {
		w.err = w.drainSSE()
	}
	if w.err != nil {
		return 0, w.err
	}
	return len(data), nil
}

func (w *codexImageDeliveryWriter) Flush() {
	if w.err == nil {
		w.ResponseWriter.Flush()
	}
}

func (w *codexImageDeliveryWriter) finish() {
	if w.err != nil || len(w.pending) == 0 {
		return
	}
	body := w.pending
	w.pending = nil
	if strings.Contains(w.Header().Get("Content-Type"), "application/json") && w.Status() < 400 && !w.bypass {
		body = w.delivery.JSON(body)
	}
	w.Header().Del("Content-Length")
	_, w.err = w.ResponseWriter.Write(body)
}

func (w *codexImageDeliveryWriter) drainSSE() error {
	for {
		end, separator := bytes.Index(w.pending, []byte("\n\n")), 2
		if crlf := bytes.Index(w.pending, []byte("\r\n\r\n")); crlf >= 0 && (end < 0 || crlf < end) {
			end, separator = crlf, 4
		}
		if end < 0 {
			return nil
		}
		frame := w.pending[:end+separator]
		var data []byte
		eventType := ""
		for _, line := range bytes.Split(bytes.ReplaceAll(frame, []byte("\r\n"), []byte("\n")), []byte("\n")) {
			if bytes.HasPrefix(line, []byte("event:")) {
				eventType = strings.TrimSpace(string(line[6:]))
			}
			if bytes.HasPrefix(line, []byte("data:")) {
				if len(data) > 0 {
					data = append(data, '\n')
				}
				data = append(data, bytes.TrimPrefix(line[5:], []byte(" "))...)
			}
		}
		if gjson.ValidBytes(data) {
			if gjson.GetBytes(data, "type").String() == "" && eventType != "" {
				data, _ = sjson.SetBytes(data, "type", eventType)
			}
			if err := w.delivery.WriteEvent(data, func(payload []byte) error {
				if bytes.Equal(payload, data) {
					_, err := w.ResponseWriter.Write(frame)
					return err
				}
				if _, err := io.WriteString(w.ResponseWriter, "event: "+gjson.GetBytes(payload, "type").String()+"\ndata: "); err != nil {
					return err
				}
				if _, err := w.ResponseWriter.Write(payload); err != nil {
					return err
				}
				_, err := io.WriteString(w.ResponseWriter, "\n\n")
				return err
			}); err != nil {
				return err
			}
		} else if _, err := w.ResponseWriter.Write(frame); err != nil {
			return err
		}
		w.pending = w.pending[end+separator:]
		if len(w.pending) == 0 {
			w.pending = nil
		}
	}
}
