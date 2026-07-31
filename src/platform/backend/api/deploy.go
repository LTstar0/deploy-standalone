package api

import (
	"deploy-platform/model"
	"deploy-platform/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DeployAPI struct {
	svc *service.DeployService
	hist *service.HistoryService
}

func NewDeployAPI(svc *service.DeployService, hist *service.HistoryService) *DeployAPI {
	return &DeployAPI{svc: svc, hist: hist}
}

func (a *DeployAPI) Register(r *gin.RouterGroup) {
	r.POST("/deploy/start", a.Start)
	r.GET("/deploy/tasks/:id", a.GetTask)
	r.GET("/deploy/history", a.History)
	r.GET("/deploy/history/:id", a.HistoryDetail)
	r.POST("/deploy/rollback/:id", a.Rollback)
}

func (a *DeployAPI) Start(c *gin.Context) {
	var req model.StartDeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	task, err := a.svc.StartDeploy(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "部署已启动",
		"task_id": task.ID,
		"status":  task.Status,
	})
}

func (a *DeployAPI) GetTask(c *gin.Context) {
	id := c.Param("id")
	task := a.svc.GetActiveTask(id)
	if task != nil {
		c.JSON(http.StatusOK, gin.H{"task": task})
		return
	}
	// Try history
	hist, err := a.hist.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"task": hist})
}

func (a *DeployAPI) History(c *gin.Context) {
	tasks, _ := a.hist.List()
	if tasks == nil {
		tasks = []*model.DeployTask{}
	}
	c.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

func (a *DeployAPI) HistoryDetail(c *gin.Context) {
	task, err := a.hist.Get(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"task": task})
}

func (a *DeployAPI) Rollback(c *gin.Context) {
	task, err := a.svc.Rollback(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "回滚已启动",
		"task_id": task.ID,
	})
}
