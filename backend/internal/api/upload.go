package api

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"crm-backend/internal/storage"
	"crm-backend/pkg/response"
)

// UploadResp 单文件上传响应
type UploadResp struct {
	Path     string `json:"path"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	URL      string `json:"url"`
}

// uploadFiles 多文件上传
// 表单字段名: files
func (r *Router) uploadFiles(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "表单解析失败")
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		response.Fail(c, 1, "没有文件")
		return
	}

	// 业务子目录（默认 screenshots，可由 form.sub_dir 指定）
	subDir := c.DefaultPostForm("sub_dir", "screenshots")

	var results []UploadResp
	for _, file := range files {
		res, err := r.Uploader.Save(file, storage.FileTypeImage, subDir)
		if err != nil {
			response.Fail(c, 1, "上传失败: "+err.Error())
			return
		}
		results = append(results, UploadResp{
			Path:     res.Path,
			Filename: res.Filename,
			Size:     res.Size,
			URL:      res.URL,
		})
	}

	response.OK(c, gin.H{
		"files": results,
	})
}

// uploadSingleFile 单文件上传（保留兼容）
func (r *Router) uploadSingleFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "未找到文件")
		return
	}

	subDir := c.DefaultPostForm("sub_dir", "screenshots")
	res, err := r.Uploader.Save(file, storage.FileTypeImage, subDir)
	if err != nil {
		response.Fail(c, 1, "上传失败: "+err.Error())
		return
	}

	response.OK(c, UploadResp{
		Path:     res.Path,
		Filename: res.Filename,
		Size:     res.Size,
		URL:      res.URL,
	})
}

// validateImageExt 校验图片扩展名
func validateImageExt(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	allowed := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	}
	return allowed[ext]
}

// 占位：避免 unused 警告
var _ = http.StatusOK
var _ = validateImageExt