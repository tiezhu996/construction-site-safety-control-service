package handler

import (
	"log/slog"
	"net/http"

	"safetyplatform/internal/config"
	"safetyplatform/internal/constants"
	"safetyplatform/internal/util"

	"github.com/gin-gonic/gin"
)

// UploadHandler 图片上传处理器。
type UploadHandler struct {
	cfg    *config.Config
	logger *slog.Logger
}

// NewUploadHandler 构造上传处理器。
func NewUploadHandler(cfg *config.Config, logger *slog.Logger) *UploadHandler {
	return &UploadHandler{cfg: cfg, logger: logger}
}

// UploadImage 上传图片。
func (h *UploadHandler) UploadImage(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "Upload image: missing file")
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		files = form.File["file"]
	}
	if len(files) == 0 || len(files) > h.cfg.UploadBatchMax {
		Fail(c, http.StatusRequestEntityTooLarge, constants.CodeUploadTooLarge, "Upload image: invalid batch size")
		return
	}
	sources := make([]util.UploadSource, 0, len(files))
	for _, file := range files {
		sources = append(sources, util.MultipartUpload(file))
	}
	urls, err := util.SaveUploadedImages(h.cfg.UploadDir, h.cfg.UploadMaxMB, sources)
	if err != nil {
		h.logger.Error(constants.LogUploadImageFailed, "error", err.Error())
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "Upload image failed: "+err.Error())
		return
	}
	h.logger.Info(constants.LogUploadImageSuccess, "count", len(urls))
	OK(c, gin.H{"urls": urls})
}
