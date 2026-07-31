#!/usr/bin/env bash
# =============================================================================
# apps/web-frontend/health-check.sh — 前端健康检查脚本（示例）
# =============================================================================
set -euo pipefail

TARGET_URL="${HEALTH_CHECK_URL:-http://localhost:80}"
TIMEOUT="${HEALTH_CHECK_TIMEOUT:-30}"
INTERVAL=5
WAITED=0

echo "[web-frontend] 健康检查: $TARGET_URL"

while (( WAITED < TIMEOUT )); do
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "$TARGET_URL" 2>/dev/null || echo "000")
    if [[ "$HTTP_CODE" == "200" ]]; then
        echo "  ✔ 服务正常 (HTTP $HTTP_CODE)"
        exit 0
    fi
    echo "  ⏳ 等待服务就绪... ${WAITED}/${TIMEOUT}s (HTTP $HTTP_CODE)"
    sleep $INTERVAL
    (( WAITED += INTERVAL ))
done

echo "  ✖ 健康检查超时 (${TIMEOUT}s)"
exit 1
