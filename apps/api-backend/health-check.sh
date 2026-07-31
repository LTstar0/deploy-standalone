#!/usr/bin/env bash
# =============================================================================
# apps/api-backend/health-check.sh — 后端健康检查脚本（示例）
# =============================================================================
set -euo pipefail

TARGET_URL="${HEALTH_CHECK_URL:-http://localhost:8080/actuator/health}"
TIMEOUT="${HEALTH_CHECK_TIMEOUT:-60}"
INTERVAL=5
WAITED=0

echo "[api-backend] 健康检查: $TARGET_URL"

while (( WAITED < TIMEOUT )); do
    RESPONSE=$(curl -s --max-time 5 "$TARGET_URL" 2>/dev/null || echo "")
    STATUS=$(echo "$RESPONSE" | grep -o '"status":"UP"' || echo "")
    if [[ -n "$STATUS" ]]; then
        echo "  ✔ 服务正常 (status: UP)"
        exit 0
    fi
    echo "  ⏳ 等待服务就绪... ${WAITED}/${TIMEOUT}s"
    sleep $INTERVAL
    (( WAITED += INTERVAL ))
done

echo "  ✖ 健康检查超时 (${TIMEOUT}s)"
exit 1
