#!/usr/bin/env bash
# =============================================================================
# scripts/rollback.sh — 通用回滚编排脚本
# 可独立使用，对指定的多个应用按逆序执行回滚
# 用法: ./scripts/rollback.sh --apps "web-frontend api-backend database"
#       ./scripts/rollback.sh  （无参数则回滚 apps.json 中所有启用的应用）
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
source "${PROJECT_ROOT}/scripts/common.sh"

mkdir -p "${PROJECT_ROOT}/logs"
export LOG_FILE="${PROJECT_ROOT}/logs/rollback-$(date +%Y%m%d-%H%M%S).log"
echo "===== 回滚日志开始: $(date) =====" > "$LOG_FILE"

CONFIG_FILE="${PROJECT_ROOT}/config/apps.json"
ROLLBACK_APPS_RAW=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --apps|-a) ROLLBACK_APPS_RAW="$2"; shift 2 ;;
        --config|-c) CONFIG_FILE="$2"; shift 2 ;;
        --help|-h)
            echo "用法: $0 [--apps 'app1 app2 ...'] [--config path]"
            exit 0 ;;
        *) echo "未知参数: $1"; exit 1 ;;
    esac
done

check_dependencies

log_step "开始独立回滚流程"
log_info "日志文件: $LOG_FILE"

if [[ -n "$ROLLBACK_APPS_RAW" ]]; then
    read -ra ROLLBACK_APPS <<< "$ROLLBACK_APPS_RAW"
else
    # 默认回滚所有启用应用（按 deploy_order 逆序）
    ROLLBACK_APPS=()
    while IFS= read -r key; do
        if is_app_enabled "$CONFIG_FILE" "$key"; then
            ROLLBACK_APPS+=("$key")
        fi
    done < <(get_all_app_keys "$CONFIG_FILE")

    # 按 deploy_order 排序后逆序
    local_sorted=($(sort_by_deploy_order "$CONFIG_FILE" "${ROLLBACK_APPS[@]}"))
    ROLLBACK_APPS=()
    for (( i=${#local_sorted[@]}-1; i>=0; i-- )); do
        ROLLBACK_APPS+=("${local_sorted[$i]}")
    done
fi

echo "" | tee -a "$LOG_FILE"
echo -e "  ${BOLD_WHITE}即将回滚（按以下顺序）：${RESET}" | tee -a "$LOG_FILE"
for app in "${ROLLBACK_APPS[@]}"; do
    name=$(get_app_field "$CONFIG_FILE" "$app" "name")
    echo -e "  ${YELLOW}→ $app${RESET} ($name)" | tee -a "$LOG_FILE"
done
echo "" | tee -a "$LOG_FILE"

confirm "确认执行回滚？" "n" || { log_info "取消回滚"; exit 0; }

success=0
fail=0
for app_key in "${ROLLBACK_APPS[@]}"; do
    name=$(get_app_field "$CONFIG_FILE" "$app_key" "name")
    rollback_support=$(get_app_field "$CONFIG_FILE" "$app_key" "rollback_support")
    script_path=$(get_app_field "$CONFIG_FILE" "$app_key" "scripts.rollback")

    log_info "回滚: $name ($app_key)"

    if [[ "$rollback_support" != "true" ]]; then
        log_warn "  该应用不支持回滚，跳过"
        continue
    fi

    if [[ -n "$script_path" && -f "${PROJECT_ROOT}/${script_path}" ]]; then
        if run_script "${PROJECT_ROOT}/${script_path}" "rollback[$app_key]"; then
            log_success "  ✔ 回滚成功: $app_key"
            ((success++))
        else
            log_error "  ✖ 回滚失败: $app_key"
            ((fail++))
        fi
    else
        log_warn "  无回滚脚本配置，跳过: $app_key"
    fi
done

print_separator
log_info "回滚完成  成功: $success  失败: $fail"
[[ $fail -gt 0 ]] && exit 1 || exit 0
