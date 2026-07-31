#!/usr/bin/env bash
# =============================================================================
# apps/api-backend/rollback.sh — 后端回滚脚本（示例）
# =============================================================================
set -euo pipefail

echo "====== [api-backend] 开始回滚 ======"

echo "[1/3] 停止当前版本..."
# 实际命令示例：
# systemctl stop api-backend
echo "  → 已停止（示例）"

echo "[2/3] 恢复旧版本..."
# 实际命令示例：
# LATEST_BACKUP=$(ls -td /opt/backups/api-backend/*/ | head -1)
# cp -r "$LATEST_BACKUP"/* /opt/api-backend/
echo "  → 旧版本已恢复（示例）"

echo "[3/3] 重启服务..."
# 实际命令示例：
# systemctl start api-backend
echo "  → 服务已重启（示例）"

echo "====== [api-backend] 回滚完成 ======"
exit 0
