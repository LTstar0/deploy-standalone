.PHONY: dev-backend dev-frontend build clean

# 开发模式 — 后端
dev-backend:
	cd backend && go run main.go

# 开发模式 — 前端
dev-frontend:
	cd frontend && npm run dev

# 构建前端
build-frontend:
	cd frontend && npm run build

# 构建后端（Linux amd64）
build-backend:
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../dist/deploy-platform main.go

# 完整构建
build: build-frontend build-backend
	cp -r frontend/dist backend/data 2>/dev/null || true
	@echo "✔ 构建完成: dist/deploy-platform"

# 清理
clean:
	rm -rf dist/ frontend/dist/ backend/data/

# 安装前端依赖
install:
	cd frontend && npm install
