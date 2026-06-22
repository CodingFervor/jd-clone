package handler

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// UploadImage handles a generic image upload (multipart form, field "file").
// It validates the file is an image, stores it under data/images/, and returns
// the public URL. Used by the admin product form, review photos, etc.
func (h *Handler) UploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传图片文件"})
		return
	}
	// Validate it looks like an image by extension.
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	if !allowed[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 jpg/png/gif/webp 格式"})
		return
	}
	// Cap file size at 10MB.
	if file.Size > 10<<20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片大小不能超过10MB"})
		return
	}
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	dst := filepath.Join("data", "images", filename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	url := "/images/" + filename
	c.JSON(http.StatusOK, gin.H{"url": url, "message": "上传成功"})
}
