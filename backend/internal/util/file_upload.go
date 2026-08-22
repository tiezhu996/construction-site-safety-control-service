package util

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var allowedImageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
}

// UploadSource 描述一个上传源。
type UploadSource interface {
	Name() string
	SizeBytes() int64
	Open() (io.ReadCloser, error)
}

type multipartUpload struct{ header *multipart.FileHeader }

func (m multipartUpload) Name() string                 { return m.header.Filename }
func (m multipartUpload) SizeBytes() int64             { return m.header.Size }
func (m multipartUpload) Open() (io.ReadCloser, error) { return m.header.Open() }

// MultipartUpload 把 multipart 文件转换为上传源。
func MultipartUpload(header *multipart.FileHeader) UploadSource {
	return multipartUpload{header: header}
}

// SaveUploadedImage 保存单张图片。
func SaveUploadedImage(uploadDir string, maxMB int64, file *multipart.FileHeader) (string, error) {
	urls, err := SaveUploadedImages(uploadDir, maxMB, []UploadSource{MultipartUpload(file)})
	if err != nil {
		return "", err
	}
	return urls[0], nil
}

// SaveUploadedImages 保存一批图片。任意一张失败时回滚整批已落盘的文件，
// 保证目录不会残留半成品；每张图片的句柄在写入完成后立即关闭，避免句柄堆积。
func SaveUploadedImages(uploadDir string, maxMB int64, files []UploadSource) (urls []string, err error) {
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	urls = make([]string, 0, len(files))
	written := make([]string, 0, len(files)) // 本批已落盘的文件名，失败时用于回滚
	rollback := func() {
		for _, name := range written {
			os.Remove(filepath.Join(uploadDir, name))
		}
	}
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if !allowedImageExts[ext] {
			rollback()
			return nil, fmt.Errorf("unsupported file type: %s", ext)
		}
		if file.SizeBytes() > maxMB*1024*1024 {
			rollback()
			return nil, fmt.Errorf("file too large: %d bytes", file.SizeBytes())
		}
		src, openErr := file.Open()
		if openErr != nil {
			rollback()
			return nil, fmt.Errorf("open upload file: %w", openErr)
		}
		name := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
		out, createErr := os.Create(filepath.Join(uploadDir, name))
		if createErr != nil {
			src.Close()
			rollback()
			return nil, fmt.Errorf("create destination file: %w", createErr)
		}
		written = append(written, name) // 先登记，写入失败时由 rollback 一并清理
		_, copyErr := io.Copy(out, src)
		src.Close()
		closeErr := out.Close()
		if copyErr != nil {
			// 写入错误为主错误，丢弃 close 错误以免覆盖。
			rollback()
			return nil, fmt.Errorf("write upload file: %w", copyErr)
		}
		if closeErr != nil {
			rollback()
			return nil, fmt.Errorf("close destination file: %w", closeErr)
		}
		urls = append(urls, "/uploads/"+name)
	}
	return urls, nil
}
