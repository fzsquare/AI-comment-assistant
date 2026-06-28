package merchant

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"ppk/backend/internal/model"
	"ppk/backend/internal/pkg/response"
	"ppk/backend/internal/pkg/utils"

	"github.com/gin-gonic/gin"
)

const maxImageUploadBytes = 5 << 20 // 5MB

// 按嗅探出的真实 MIME 决定扩展名（不信任前端传来的文件名/扩展名）
var imageExtByType = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

// uploadImageFile 接收商家直接上传的图片文件，存到本地 UploadDir，
// 经 /uploads 静态路由对外访问，并落一条 StoreImage 记录。
func (h *Handler) uploadImageFile(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "请选择要上传的图片")
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > maxImageUploadBytes {
		response.Error(c, http.StatusBadRequest, "图片需在 5MB 以内")
		return
	}

	// 嗅探真实类型
	src, err := fileHeader.Open()
	if err != nil {
		response.Error(c, http.StatusBadRequest, "图片读取失败")
		return
	}
	head := make([]byte, 512)
	n, _ := io.ReadFull(src, head)
	_ = src.Close()
	ext, ok := imageExtByType[http.DetectContentType(head[:n])]
	if !ok {
		response.Error(c, http.StatusBadRequest, "仅支持 jpg / png / webp / gif 图片")
		return
	}

	name := utils.RandomString(24) + ext
	dst := filepath.Join(h.Config.UploadDir, name)
	if err := c.SaveUploadedFile(fileHeader, dst); err != nil {
		response.Error(c, http.StatusInternalServerError, "图片保存失败")
		return
	}

	url := h.publicURL(c, "/uploads/"+name)
	var count int64
	h.DB.Model(&model.StoreImage{}).Where("store_id = ?", store.ID).Count(&count)
	img := model.StoreImage{
		StoreID:      store.ID,
		ImageURL:     url,
		ThumbnailURL: url,
		Status:       model.StatusEnabled,
		SortNo:       int(count) + 1,
	}
	if err := h.DB.Create(&img).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "保存失败")
		return
	}
	response.Success(c, img)
}

// publicURL 构造上传文件对外可访问的绝对地址。
// 配置了 PublicBaseURL 用它，否则按当前请求的 scheme + host 推导。
func (h *Handler) publicURL(c *gin.Context, path string) string {
	if h.Config.PublicBaseURL != "" {
		return h.Config.PublicBaseURL + path
	}
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host + path
}
