#!/usr/bin/env bash
# =============================================================================
# apps/web-frontend/rollback.sh — 前端回滚脚本（示例）
# =============================================================================
set -euo pipefail

echo "====== [web-frontend] 开始回滚 ======"

echo "[1/3] 停止当前版本..."
# 实际命令示例：
# systemctl stop nginx
# docker stop web-frontend && docker rm web-frontend
echo "  → 已停止（示例）"

echo "[2/3] 恢复旧版本..."
# 实际命令示例：
# LATEST_BACKUP=$(ls -td /opt/backups/web-frontend/*/ | head -1)
# cp -r "$LATEST_BACKUP"/* /opt/web-frontend/
echo "  → 旧版本已恢复（示例）"

echo "[3/3] 重启服务..."
# 实际命令示例：
# systemctl start nginx
echo "  → 服务已重启（示例）"

echo "====== [web-frontend] 回滚完成 ======"
exit 0
