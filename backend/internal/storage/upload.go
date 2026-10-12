package storage

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FileType 文件类型
type FileType string

const (
	FileTypeImage FileType = "image"
	FileTypeFile  FileType = "file"
)

// UploadResult 上传结果
type UploadResult struct {
	Path     string `json:"path"`      // 相对路径（存数据库）
	Filename string `json:"filename"`  // 文件名
	Size     int64  `json:"size"`      // 字节
	URL      string `json:"url"`       // 访问 URL
}

// Uploader 文件上传器（本地存储实现）
type Uploader struct {
	rootDir     string // 项目根目录/uploads
	urlPrefix   string // HTTP URL 前缀
	maxFileSize int64  // 最大文件大小（字节）
}

// NewUploader 创建上传器
// rootDir: 物理存储根目录（如 ./uploads）
// urlPrefix: 静态文件服务 URL 前缀（如 /uploads）
func NewUploader(rootDir, urlPrefix string) *Uploader {
	if !strings.HasSuffix(urlPrefix, "/") {
		urlPrefix += "/"
	}
	return &Uploader{
		rootDir:     rootDir,
		urlPrefix:   urlPrefix,
		maxFileSize: 10 << 20, // 10 MB
	}
}

// Save 保存文件
// fileType: image / file
// subDir: 二级目录（如 screenshots/）
func (u *Uploader) Save(file *multipart.FileHeader, fileType FileType, subDir string) (*UploadResult, error) {
	if file.Size > u.maxFileSize {
		return nil, fmt.Errorf("文件过大，最大 %d MB", u.maxFileSize/(1<<20))
	}

	// 校验扩展名
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if fileType == FileTypeImage {
		allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
		if !allowed[ext] {
			return nil, errors.New("图片格式不支持，仅支持 jpg/png/gif/webp")
		}
	}

	// 生成文件名
	now := time.Now()
	dateDir := now.Format("2006-01-02")
	uuidStr := uuid.New().String()
	newName := fmt.Sprintf("%s_%d%s", uuidStr, now.UnixNano(), ext)

	// 目标目录
	relDir := filepath.Join(string(fileType), subDir, dateDir)
	absDir := filepath.Join(u.rootDir, relDir)
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	relPath := filepath.Join(relDir, newName)
	absPath := filepath.Join(u.rootDir, relPath)

	// 打开并保存
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	dst, err := os.Create(absPath)
	if err != nil {
		return nil, err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return nil, err
	}

	return &UploadResult{
		Path:     filepath.ToSlash(relPath),
		Filename: file.Filename,
		Size:     file.Size,
		URL:      u.urlPrefix + filepath.ToSlash(relPath),
	}, nil
}

// Delete 删除文件
func (u *Uploader) Delete(relPath string) error {
	absPath := filepath.Join(u.rootDir, relPath)
	if !strings.HasPrefix(absPath, u.rootDir) {
		return errors.New("非法路径")
	}
	return os.Remove(absPath)
}

// Read 读取文件
func (u *Uploader) Read(relPath string) (*os.File, error) {
	absPath := filepath.Join(u.rootDir, relPath)
	if !strings.HasPrefix(absPath, u.rootDir) {
		return nil, errors.New("非法路径")
	}
	return os.Open(absPath)
}