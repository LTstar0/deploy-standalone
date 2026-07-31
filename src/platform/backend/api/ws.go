package api

import (
	"deploy-platform/service"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type WSAPI struct {
	deploySvc *service.DeployService
}

func NewWSAPI(deploySvc *service.DeployService) *WSAPI {
	return &WSAPI{deploySvc: deploySvc}
}

func (a *WSAPI) Register(r *gin.Engine) {
	r.GET("/ws/deploy/:taskId", a.Handle)
}

func (a *WSAPI) Handle(c *gin.Context) {
	// Verify Token
	if DeployToken != "" {
		token := c.GetHeader("X-Deploy-Token")
		if token == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
					token = authHeader[7:]
				} else {
					token = authHeader
				}
			}
		}
		if token == "" {
			token = c.Query("token")
		}

		if token != DeployToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权，请输入正确的发布 Token"})
			return
		}
	}

	taskID := c.Param("taskId")
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)

	if err != nil {
		return
	}
	defer conn.Close()

	subID := uuid.New().String()[:8]
	ch, err := a.deploySvc.Subscribe(taskID, subID)
	if err != nil {
		conn.WriteJSON(gin.H{"type": "error", "data": err.Error()})
		return
	}
	defer a.deploySvc.Unsubscribe(taskID, subID)

	// Read goroutine (handle close)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if err := conn.WriteJSON(msg); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}
