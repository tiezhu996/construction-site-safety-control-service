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

// SaveUploadedImages 保存一批图片。
func SaveUploadedImages(uploadDir string, maxMB int64, files []UploadSource) (urls []string, err error) {
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	urls = make([]string, 0, len(files))
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if !allowedImageExts[ext] {
			return nil, fmt.Errorf("unsupported file type: %s", ext)
		}
		if file.SizeBytes() > maxMB*1024*1024 {
			return nil, fmt.Errorf("file too large: %d bytes", file.SizeBytes())
		}
		var src io.ReadCloser
		src, err = file.Open()
		if err != nil {
			return nil, fmt.Errorf("open upload file: %w", err)
		}
		defer func() { err = src.Close() }()
		name := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
		var out *os.File
		out, err = os.Create(filepath.Join(uploadDir, name))
		if err != nil {
			return nil, fmt.Errorf("create destination file: %w", err)
		}
		defer func() { err = out.Close() }()
		if _, err = io.Copy(out, src); err != nil {
			return nil, fmt.Errorf("write upload file: %w", err)
		}
		urls = append(urls, "/uploads/"+name)
	}
	return urls, nil
}
