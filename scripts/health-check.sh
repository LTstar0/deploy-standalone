#!/usr/bin/env bash
# =============================================================================
# scripts/health-check.sh — 通用健康检查脚本
# 可单独使用检查一个 URL 或命令是否就绪
# 用法: ./scripts/health-check.sh --url http://localhost:8080/health --timeout 60
#       ./scripts/health-check.sh --cmd "redis-cli ping" --timeout 30
# =============================================================================
set -euo pipefail

TARGET_URL=""
TARGET_CMD=""
TIMEOUT=30
INTERVAL=5
LABEL="服务"

# ── 解析参数 ──────────────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case "$1" in
        --url|-u)     TARGET_URL="$2"; shift 2 ;;
        --cmd|-c)     TARGET_CMD="$2"; shift 2 ;;
        --timeout|-t) TIMEOUT="$2";   shift 2 ;;
        --label|-l)   LABEL="$2";     shift 2 ;;
        --help|-h)
            echo "用法: $0 [--url URL | --cmd CMD] [--timeout N] [--label 名称]"
            exit 0
            ;;
        *) echo "未知参数: $1"; exit 1 ;;
    esac
done

if [[ -z "$TARGET_URL" && -z "$TARGET_CMD" ]]; then
    echo "错误: 需要 --url 或 --cmd 参数"
    exit 1
fi

WAITED=0
echo "[health-check] 目标: ${TARGET_URL:-$TARGET_CMD}  超时: ${TIMEOUT}s"

while (( WAITED < TIMEOUT )); do
    if [[ -n "$TARGET_URL" ]]; then
        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "$TARGET_URL" 2>/dev/null || echo "000")
        if [[ "$HTTP_CODE" =~ ^2[0-9][0-9]$ ]]; then
            echo "  ✔ $LABEL 健康 (HTTP $HTTP_CODE)"
            exit 0
        fi
        echo "  ⏳ 等待中... ${WAITED}/${TIMEOUT}s (HTTP $HTTP_CODE)"
    else
        if eval "$TARGET_CMD" &>/dev/null; then
            echo "  ✔ $LABEL 健康"
            exit 0
        fi
        echo "  ⏳ 等待中... ${WAITED}/${TIMEOUT}s"
    fi
    sleep $INTERVAL
    (( WAITED += INTERVAL ))
done

echo "  ✖ 健康检查超时（${TIMEOUT}s）"
exit 1
