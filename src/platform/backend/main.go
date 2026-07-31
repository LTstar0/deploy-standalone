package main

import (
	"crypto/rand"
	"deploy-platform/api"
	"deploy-platform/service"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Data directory
	dataDir := getEnv("DATA_DIR", "./data")
	port := getEnv("PORT", "9090")
	frontendDir := getEnv("FRONTEND_DIR", "")

	os.MkdirAll(dataDir, 0755)

	// ── Token Authentication Initialization ─────────────────────────────────────
	token := os.Getenv("DEPLOY_TOKEN")
	if token == "" {
		tokenPath := filepath.Join(dataDir, "token.txt")
		if data, err := os.ReadFile(tokenPath); err == nil {
			token = string(data)
		} else {
			// Generate random token
			token = "deploy_" + generateRandomToken(16)
			_ = os.WriteFile(tokenPath, []byte(token), 0600)
		}
	}
	api.DeployToken = token

	// Services
	historySvc := service.NewHistoryService(filepath.Join(dataDir, "history.json"))
	packageSvc := service.NewPackageService(dataDir)
	deploySvc := service.NewDeployService(packageSvc, historySvc)

	// Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Deploy-Token"},
		AllowCredentials: true,
	}))

	// API routes
	apiGroup := r.Group("/api")
	apiGroup.Use(api.AuthMiddleware()) // Protected API group

	api.NewPackageAPI(packageSvc).Register(apiGroup)
	api.NewDeployAPI(deploySvc, historySvc).Register(apiGroup)

	// Verification endpoint
	apiGroup.GET("/verify", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// WebSocket
	api.NewWSAPI(deploySvc).Register(r)

	// Serve frontend (if built)
	if frontendDir != "" {
		r.Static("/assets", filepath.Join(frontendDir, "assets"))
		r.StaticFile("/favicon.svg", filepath.Join(frontendDir, "favicon.svg"))
		r.NoRoute(func(c *gin.Context) {
			c.File(filepath.Join(frontendDir, "index.html"))
		})
	}

	fmt.Printf("\n  🚀 Deploy Platform 启动成功\n")
	fmt.Printf("  ➜ 地址: http://localhost:%s\n", port)
	fmt.Printf("  ➜ 数据: %s\n", dataDir)
	if api.DeployToken != "" {
		fmt.Printf("  ➜ 认证: 已启用发布 Token 校验 (Token: %s)\n\n", api.DeployToken)
	} else {
		fmt.Printf("  ➜ 认证: 未启用发布 Token 校验 (免密模式)\n\n")
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func generateRandomToken(length int) string {
	b := make([]byte, length/2)
	if _, err := rand.Read(b); err != nil {
		return "deploy_token_fallback"
	}
	return hex.EncodeToString(b)
}

