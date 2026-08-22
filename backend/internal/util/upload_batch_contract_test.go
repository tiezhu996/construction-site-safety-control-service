package util_test

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"safetyplatform/internal/config"
	"safetyplatform/internal/handler"
	"safetyplatform/internal/util"

	"github.com/gin-gonic/gin"
)

type trackedSource struct {
	name           string
	data           []byte
	active         *atomic.Int32
	failWhenActive bool
	readErr        error
	closeErr       error
}

func (s trackedSource) Name() string     { return s.name }
func (s trackedSource) SizeBytes() int64 { return int64(len(s.data)) }
func (s trackedSource) Open() (io.ReadCloser, error) {
	if s.failWhenActive && s.active.Load() != 0 {
		return nil, errors.New("previous source still open")
	}
	s.active.Add(1)
	return &trackedReader{reader: bytes.NewReader(s.data), active: s.active, readErr: s.readErr, closeErr: s.closeErr}, nil
}

type trackedReader struct {
	reader   *bytes.Reader
	active   *atomic.Int32
	readErr  error
	closeErr error
}

func (r *trackedReader) Read(p []byte) (int, error) {
	if r.readErr != nil {
		return 0, r.readErr
	}
	return r.reader.Read(p)
}
func (r *trackedReader) Close() error {
	r.active.Add(-1)
	return r.closeErr
}

func TestUploadClosesEveryReader(t *testing.T) {
	var active atomic.Int32
	files := []util.UploadSource{
		trackedSource{name: "one.jpg", data: []byte("one"), active: &active},
		trackedSource{name: "two.jpg", data: []byte("two"), active: &active, failWhenActive: true},
	}
	urls, err := util.SaveUploadedImages(t.TempDir(), 1, files)
	if err != nil {
		t.Fatalf("batch kept an earlier source open: %v", err)
	}
	if len(urls) != 2 {
		t.Fatalf("saved %d files", len(urls))
	}
	if active.Load() != 0 {
		t.Fatalf("open source count=%d", active.Load())
	}
}

func TestUploadKeepsPrimaryWriteFailure(t *testing.T) {
	var active atomic.Int32
	writeErr := errors.New("source read failed")
	closeErr := errors.New("source close failed")
	_, err := util.SaveUploadedImages(t.TempDir(), 1, []util.UploadSource{
		trackedSource{name: "broken.jpg", active: &active, readErr: writeErr, closeErr: closeErr},
	})
	if !errors.Is(err, writeErr) {
		t.Fatalf("write error was lost: %v", err)
	}
}

func TestUploadCleansTemporaryBatch(t *testing.T) {
	var active atomic.Int32
	dir := t.TempDir()
	_, err := util.SaveUploadedImages(dir, 1, []util.UploadSource{
		trackedSource{name: "ok.jpg", data: []byte("ok"), active: &active},
		trackedSource{name: "bad.txt", data: []byte("bad"), active: &active},
	})
	if err == nil {
		t.Fatal("invalid batch unexpectedly succeeded")
	}
	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("failed batch left %d files", len(entries))
	}
}

func TestUploadDoesNotExposePartialBatch(t *testing.T) {
	t.Setenv("UPLOAD_BATCH_MAX", "2")
	cfg := config.Load()
	if cfg.UploadBatchMax != 2 {
		t.Errorf("batch max=%d want=2", cfg.UploadBatchMax)
	}
	cfg.UploadDir = t.TempDir()
	cfg.UploadMaxMB = 1
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, name := range []string{"a.jpg", "b.jpg", "c.jpg"} {
		part, err := writer.CreateFormFile("files", name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.Copy(part, strings.NewReader(name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	h := handler.NewUploadHandler(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	r := gin.New()
	r.POST("/upload", h.UploadImage)
	req := httptest.NewRequest(http.MethodPost, "/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	entries, err := os.ReadDir(cfg.UploadDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("rejected batch published %d files", len(entries))
	}
}
