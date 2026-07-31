package api

import (
	"deploy-platform/model"
	"deploy-platform/service"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

type PackageAPI struct {
	svc *service.PackageService
}

func NewPackageAPI(svc *service.PackageService) *PackageAPI {
	return &PackageAPI{svc: svc}
}

func (a *PackageAPI) Register(r *gin.RouterGroup) {
	r.POST("/packages/upload", a.Upload)
	r.GET("/packages", a.List)
	r.GET("/packages/:id", a.Get)
	r.DELETE("/packages/:id", a.Delete)
}

func (a *PackageAPI) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择文件上传"})
		return
	}

	ext := filepath.Ext(file.Filename)
	if ext != ".gz" && ext != ".tgz" && ext != ".zip" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 .tar.gz / .tgz / .zip 格式"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "打开文件失败"})
		return
	}
	defer f.Close()

	pkg, err := a.svc.Upload(file.Filename, f, file.Size)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "上传成功",
		"package": toPackageResp(pkg),
	})
}

func (a *PackageAPI) List(c *gin.Context) {
	packages := a.svc.List()
	var resp []gin.H
	for _, p := range packages {
		resp = append(resp, toPackageResp(p))
	}
	if resp == nil {
		resp = []gin.H{}
	}
	c.JSON(http.StatusOK, gin.H{"packages": resp})
}

func (a *PackageAPI) Get(c *gin.Context) {
	pkg, err := a.svc.Get(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "包不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"package": toPackageResp(pkg)})
}

func (a *PackageAPI) Delete(c *gin.Context) {
	if err := a.svc.Delete(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "包不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

func toPackageResp(p *model.Package) gin.H {
	return gin.H{
		"id":           p.ID,
		"file_name":    p.FileName,
		"size_bytes":   p.SizeBytes,
		"uploaded_at":  p.UploadedAt,
		"project_name": p.ProjectName,
		"version":      p.Version,
		"environment":  p.Environment,
		"apps":         p.Apps,
	}
}
