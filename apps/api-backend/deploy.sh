#!/usr/bin/env bash
# =============================================================================
# apps/api-backend/deploy.sh — 后端 API 部署脚本（示例）
# =============================================================================
set -euo pipefail

echo "====== [api-backend] 开始部署 ======"

echo "[1/4] 停止旧的 API 服务..."
# 实际命令示例：
# systemctl stop api-backend
# docker stop api-backend || true
echo "  → 旧服务已停止（示例）"

echo "[2/4] 备份旧版本..."
# 实际命令示例：
# BACKUP_DIR="/opt/backups/api-backend/$(date +%Y%m%d-%H%M%S)"
# cp -r /opt/api-backend "$BACKUP_DIR"
echo "  → 备份完成（示例）"

echo "[3/4] 部署新版本..."
# 实际命令示例：
# cp ./target/api-backend.jar /opt/api-backend/
# docker pull myapp/api-backend:latest
# docker run -d --name api-backend -p 8080:8080 myapp/api-backend:latest
echo "  → 新版本已部署（示例）"

echo "[4/4] 启动服务..."
# 实际命令示例：
# systemctl start api-backend
echo "  → API 服务已启动（示例）"

echo "====== [api-backend] 部署完成 ======"
exit 0
