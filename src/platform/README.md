# Deploy Platform — 轻量部署平台

> 上传产品包（`.tar.gz` / `.zip`），在浏览器中选择应用、一键完成部署或升级。

## 技术栈

- **后端**: Go + Gin + gorilla/websocket
- **前端**: Vue 3 + Vite + Pinia
- **数据**: 本地 JSON 文件（零数据库依赖）
- **部署脚本**: Shell（复用产品包内的 deploy.sh 等脚本）

## 快速开始

### 1. 安装依赖

```bash
# 前端
cd frontend && npm install

# 后端依赖自动管理
cd backend && go mod tidy
```

### 2. 开发模式

终端 1 — 启动后端（默认 9090 端口）：
```bash
cd backend && go run main.go
```

终端 2 — 启动前端（默认 3000 端口，自动代理到后端）：
```bash
cd frontend && npm run dev
```

打开浏览器访问 `http://localhost:3000`

### 3. 使用流程

1. **上传产品包**：拖拽或选择 `.tar.gz` / `.zip` 文件上传
2. **查看应用列表**：系统自动解析 `config/apps.json`，展示所有应用
3. **选择应用**：勾选要部署的应用（自动补全依赖）
4. **一键部署**：点击部署，实时查看终端日志
5. **查看历史**：部署记录可展开查看日志、触发回滚

## 产品包格式

平台兼容的产品包需包含 `config/apps.json`，示例结构：

```
my-product.tar.gz/
├── config/apps.json      # 应用配置（必需）
├── apps/
│   ├── database/deploy.sh
│   ├── api-backend/deploy.sh
│   └── web-frontend/deploy.sh
└── scripts/common.sh     # 可选公共脚本
```

## API

| Method | Path | 说明 |
|--------|------|------|
| POST | `/api/packages/upload` | 上传产品包 |
| GET | `/api/packages` | 列出所有包 |
| GET | `/api/packages/:id` | 包详情 |
| DELETE | `/api/packages/:id` | 删除包 |
| POST | `/api/deploy/start` | 开始部署 |
| GET | `/api/deploy/tasks/:id` | 任务状态 |
| GET | `/api/deploy/history` | 部署历史 |
| POST | `/api/deploy/rollback/:id` | 触发回滚 |
| WS | `/ws/deploy/:taskId` | 实时日志 |
