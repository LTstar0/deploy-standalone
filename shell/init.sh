#!/usr/bin/env bash
# =============================================================================
# shell/init.sh — 主发布脚本
# 用法: ./shell/init.sh [--config path/to/apps.json]
# =============================================================================
set -euo pipefail

# ── 路径解析 ──────────────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# ── 加载公共函数库 ────────────────────────────────────────────────────────────
source "${PROJECT_ROOT}/scripts/common.sh"

# ── 初始化日志目录和文件 ──────────────────────────────────────────────────────
mkdir -p "${PROJECT_ROOT}/logs"
export LOG_FILE="${PROJECT_ROOT}/logs/deploy-$(date +%Y%m%d-%H%M%S).log"
echo "===== 部署日志开始: $(date) =====" > "$LOG_FILE"

# ── 默认配置 ──────────────────────────────────────────────────────────────────
CONFIG_FILE="${PROJECT_ROOT}/config/apps.json"

# ── 解析命令行参数 ────────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case "$1" in
        --config|-c)
            CONFIG_FILE="$2"
            shift 2
            ;;
        --help|-h)
            echo "用法: $0 [--config <apps.json路径>]"
            exit 0
            ;;
        *)
            log_warn "未知参数: $1"
            shift
            ;;
    esac
done

# ── 验证配置文件 ──────────────────────────────────────────────────────────────
if [[ ! -f "$CONFIG_FILE" ]]; then
    log_error "配置文件不存在: $CONFIG_FILE"
    exit 1
fi

# ── 全局状态 ──────────────────────────────────────────────────────────────────
SELECTED_APPS=()          # 用户选中的 app key（含自动补全的依赖）
DEPLOYED_APPS=()          # 已成功部署的 app key（用于回滚）
FAILED_APP=""             # 首个失败的 app key

# =============================================================================
# 函数：显示应用选择菜单，填充 SELECTED_APPS
# =============================================================================
select_apps() {
    log_step "选择要发布的应用"
    echo "" | tee -a "$LOG_FILE"

    # 读取所有启用的应用
    local all_keys=()
    while IFS= read -r key; do
        if is_app_enabled "$CONFIG_FILE" "$key"; then
            all_keys+=("$key")
        fi
    done < <(get_all_app_keys "$CONFIG_FILE")

    if [[ ${#all_keys[@]} -eq 0 ]]; then
        log_error "配置文件中没有可用的应用（enabled: true）"
        exit 1
    fi

    # 显示应用列表
    echo -e "  ${BOLD_WHITE}可用应用列表：${RESET}" | tee -a "$LOG_FILE"
    echo "" | tee -a "$LOG_FILE"
    local idx=1
    for key in "${all_keys[@]}"; do
        local name desc order
        name=$(get_app_field "$CONFIG_FILE" "$key" "name")
        desc=$(get_app_field "$CONFIG_FILE" "$key" "description")
        order=$(get_app_field "$CONFIG_FILE" "$key" "deploy_order")
        printf "  ${BOLD_CYAN}[%d]${RESET} ${BOLD_WHITE}%-20s${RESET} ${DIM}(order:%s)${RESET} — %s\n" \
            "$idx" "${name}" "${order}" "${desc}" | tee -a "$LOG_FILE"
        ((idx++))
    done

    echo "" | tee -a "$LOG_FILE"
    echo -e "  ${DIM}输入选项：数字（如 1 3）、all（全选）${RESET}" | tee -a "$LOG_FILE"
    echo "" | tee -a "$LOG_FILE"

    local chosen_keys=()
    while true; do
        echo -en "  ${BOLD_WHITE}请选择 > ${RESET}"
        read -r input
        input="${input,,}"  # 转小写

        if [[ "$input" == "all" ]]; then
            chosen_keys=("${all_keys[@]}")
            break
        fi

        # 解析数字列表
        local valid=true
        local temp_keys=()
        for token in $input; do
            if [[ "$token" =~ ^[0-9]+$ ]]; then
                local n=$((token - 1))
                if (( n >= 0 && n < ${#all_keys[@]} )); then
                    local chosen_key="${all_keys[$n]}"
                    # 去重
                    local already=false
                    for ck in "${temp_keys[@]:-}"; do
                        [[ "$ck" == "$chosen_key" ]] && already=true && break
                    done
                    $already || temp_keys+=("$chosen_key")
                else
                    log_warn "数字 $token 超出范围（1-${#all_keys[@]}）"
                    valid=false
                    break
                fi
            else
                log_warn "无效输入: $token（请输入数字或 all）"
                valid=false
                break
            fi
        done

        if $valid && [[ ${#temp_keys[@]} -gt 0 ]]; then
            chosen_keys=("${temp_keys[@]}")
            break
        elif $valid; then
            log_warn "未选择任何应用，请重新输入"
        fi
    done

    echo "" | tee -a "$LOG_FILE"
    log_info "初始选择: ${chosen_keys[*]}"

    # ── 依赖解析 ──────────────────────────────────────────────────────────────
    local expanded_keys=()
    expanded_keys=($(resolve_dependencies "$CONFIG_FILE" "${chosen_keys[@]}"))

    # 找出自动补全的依赖
    local auto_added=()
    for key in "${expanded_keys[@]}"; do
        local in_chosen=false
        for ck in "${chosen_keys[@]}"; do
            [[ "$ck" == "$key" ]] && in_chosen=true && break
        done
        $in_chosen || auto_added+=("$key")
    done

    if [[ ${#auto_added[@]} -gt 0 ]]; then
        echo "" | tee -a "$LOG_FILE"
        log_warn "以下依赖未被选中，将自动加入部署列表："
        for dep in "${auto_added[@]}"; do
            local dep_name
            dep_name=$(get_app_field "$CONFIG_FILE" "$dep" "name")
            echo -e "  ${YELLOW}+ $dep${RESET} ($dep_name)" | tee -a "$LOG_FILE"
        done
        echo "" | tee -a "$LOG_FILE"
        if ! confirm "是否确认加入这些依赖？" "y"; then
            log_error "用户取消，退出"
            exit 1
        fi
    fi

    SELECTED_APPS=($(sort_by_deploy_order "$CONFIG_FILE" "${expanded_keys[@]}"))
}

# =============================================================================
# 函数：拓扑展开依赖（递归，去重）
# 输出到 stdout（每行一个 key）
# =============================================================================
resolve_dependencies() {
    local config="$1"
    shift
    local input_keys=("$@")
    local visited=()

    _visit() {
        local key="$1"
        # 已访问则跳过
        for v in "${visited[@]:-}"; do
            [[ "$v" == "$key" ]] && return
        done
        visited+=("$key")

        # 递归处理依赖
        while IFS= read -r dep; do
            [[ -z "$dep" ]] && continue
            _visit "$dep"
        done < <(get_app_deps "$config" "$key")
    }

    for k in "${input_keys[@]}"; do
        _visit "$k"
    done

    printf '%s\n' "${visited[@]}"
}

# =============================================================================
# 函数：显示部署计划并请求用户确认
# =============================================================================
confirm_plan() {
    log_step "部署计划确认"
    echo "" | tee -a "$LOG_FILE"
    echo -e "  ${BOLD_WHITE}以下应用将按顺序部署：${RESET}" | tee -a "$LOG_FILE"
    echo "" | tee -a "$LOG_FILE"

    local idx=1
    for app in "${SELECTED_APPS[@]}"; do
        local name deps_str
        name=$(get_app_field "$CONFIG_FILE" "$app" "name")
        local deps
        deps=$(get_app_deps "$CONFIG_FILE" "$app" | tr '\n' ',' | sed 's/,$//')
        deps_str="${deps:-无}"
        printf "  ${BOLD_GREEN}%d.${RESET} ${BOLD_WHITE}%-22s${RESET} ${DIM}依赖: %s${RESET}\n" \
            "$idx" "${name} (${app})" "${deps_str}" | tee -a "$LOG_FILE"
        ((idx++))
    done

    echo "" | tee -a "$LOG_FILE"
    print_separator
    echo "" | tee -a "$LOG_FILE"

    if ! confirm "确认开始部署？" "y"; then
        log_info "用户取消部署，退出"
        exit 0
    fi
}

# =============================================================================
# 函数：执行健康检查（URL 或 命令 或 脚本）
# 返回: 0=健康 1=不健康
# =============================================================================
do_health_check() {
    local config="$1" app_key="$2"

    local timeout url cmd script

    timeout=$(jq -r ".apps[\"${app_key}\"].health_check.timeout // 30" "$config")
    url=$(jq -r ".apps[\"${app_key}\"].health_check.url // empty" "$config")
    cmd=$(jq -r ".apps[\"${app_key}\"].health_check.command // empty" "$config")
    script=$(jq -r ".apps[\"${app_key}\"].scripts.health_check // empty" "$config")

    # 优先级：脚本 > URL > 命令
    if [[ -n "$script" && -f "${PROJECT_ROOT}/${script}" ]]; then
        log_info "健康检查（脚本）: ${script}"
        if bash "${PROJECT_ROOT}/${script}" >> "$LOG_FILE" 2>&1; then
            return 0
        fi
        return 1
    fi

    if [[ -n "$url" ]]; then
        log_info "健康检查（URL）: ${url}  超时: ${timeout}s"
        local waited=0
        local interval=5
        while (( waited < timeout )); do
            if curl -sf --max-time 5 "$url" &>/dev/null; then
                return 0
            fi
            sleep $interval
            (( waited += interval ))
            log_info "  等待服务就绪... ${waited}/${timeout}s"
        done
        log_error "健康检查超时（URL 未响应）: $url"
        return 1
    fi

    if [[ -n "$cmd" ]]; then
        log_info "健康检查（命令）: ${cmd}"
        local waited=0
        local interval=5
        while (( waited < timeout )); do
            if eval "$cmd" &>/dev/null; then
                return 0
            fi
            sleep $interval
            (( waited += interval ))
            log_info "  等待服务就绪... ${waited}/${timeout}s"
        done
        log_error "健康检查超时: $cmd"
        return 1
    fi

    log_info "无健康检查配置，跳过"
    return 0
}

# =============================================================================
# 函数：部署单个应用
# 返回: 0=成功 1=失败
# =============================================================================
deploy_app() {
    local app_key="$1"
    local name
    name=$(get_app_field "$CONFIG_FILE" "$app_key" "name")

    log_step "部署应用: ${name} (${app_key})"

    local pre_script deploy_script post_script
    pre_script=$(get_app_field "$CONFIG_FILE" "$app_key" "scripts.pre_deploy")
    deploy_script=$(get_app_field "$CONFIG_FILE" "$app_key" "scripts.deploy")
    post_script=$(get_app_field "$CONFIG_FILE" "$app_key" "scripts.post_deploy")

    # Pre-deploy
    if [[ -n "$pre_script" ]]; then
        log_info "执行 pre-deploy..."
        if ! run_script "${PROJECT_ROOT}/${pre_script}" "pre_deploy"; then
            log_error "pre_deploy 失败: $app_key"
            return 1
        fi
    fi

    # Deploy（必需）
    if [[ -z "$deploy_script" ]]; then
        log_warn "未配置 deploy 脚本，跳过: $app_key"
    else
        log_info "执行 deploy..."
        if ! run_script "${PROJECT_ROOT}/${deploy_script}" "deploy"; then
            log_error "deploy 失败: $app_key"
            return 1
        fi
    fi

    # Health check
    log_info "执行健康检查..."
    if ! do_health_check "$CONFIG_FILE" "$app_key"; then
        log_error "健康检查失败: $app_key"
        return 1
    fi
    log_success "健康检查通过: $app_key"

    # Post-deploy
    if [[ -n "$post_script" ]]; then
        log_info "执行 post-deploy..."
        if ! run_script "${PROJECT_ROOT}/${post_script}" "post_deploy"; then
            log_warn "post_deploy 执行失败（不阻塞部署）: $app_key"
        fi
    fi

    return 0
}

# =============================================================================
# 函数：回滚已部署的应用（逆序）
# =============================================================================
do_rollback() {
    if [[ ${#DEPLOYED_APPS[@]} -eq 0 ]]; then
        log_warn "没有已部署的应用需要回滚"
        return
    fi

    log_step "开始回滚（逆序）"

    local reversed=()
    for (( i=${#DEPLOYED_APPS[@]}-1; i>=0; i-- )); do
        reversed+=("${DEPLOYED_APPS[$i]}")
    done

    for app_key in "${reversed[@]}"; do
        local name rollback_script rollback_support
        name=$(get_app_field "$CONFIG_FILE" "$app_key" "name")
        rollback_support=$(get_app_field "$CONFIG_FILE" "$app_key" "rollback_support")
        rollback_script=$(get_app_field "$CONFIG_FILE" "$app_key" "scripts.rollback")

        log_info "回滚: $name ($app_key)"

        if [[ "$rollback_support" != "true" ]]; then
            log_warn "  该应用不支持回滚，跳过"
            continue
        fi

        if [[ -n "$rollback_script" && -f "${PROJECT_ROOT}/${rollback_script}" ]]; then
            if run_script "${PROJECT_ROOT}/${rollback_script}" "rollback"; then
                log_success "  回滚成功: $app_key"
            else
                log_error "  回滚失败: $app_key（需手动处理）"
            fi
        else
            log_warn "  无回滚脚本，跳过: $app_key"
        fi
    done

    log_info "回滚完成"
}

# =============================================================================
# 主流程
# =============================================================================
main() {
    # 检查依赖
    check_dependencies

    # 读取项目信息
    local proj_name proj_version proj_env
    proj_name=$(jq -r '.project.name // "未命名项目"' "$CONFIG_FILE")
    proj_version=$(jq -r '.project.version // "0.0.0"' "$CONFIG_FILE")
    proj_env=$(jq -r '.project.environment // "unknown"' "$CONFIG_FILE")

    # 打印 Banner
    print_banner "$proj_name" "$proj_version" "$proj_env"

    log_info "日志文件: $LOG_FILE"
    log_info "配置文件: $CONFIG_FILE"
    print_separator

    # 选择应用
    select_apps

    # 确认计划
    confirm_plan

    # 读取部署配置
    local stop_on_failure
    stop_on_failure=$(get_deploy_config "$CONFIG_FILE" "stop_on_failure" "true")

    # ── 逐个部署 ──────────────────────────────────────────────────────────────
    local total=${#SELECTED_APPS[@]}
    local success_count=0
    local fail_count=0

    for i in "${!SELECTED_APPS[@]}"; do
        local app_key="${SELECTED_APPS[$i]}"
        local progress="[$(( i + 1 ))/$total]"

        echo "" | tee -a "$LOG_FILE"
        echo -e "${BOLD_BLUE}${progress}${RESET} 部署中..." | tee -a "$LOG_FILE"

        if deploy_app "$app_key"; then
            log_success "$progress 部署成功: $app_key"
            DEPLOYED_APPS+=("$app_key")
            ((success_count++))
        else
            log_error "$progress 部署失败: $app_key"
            FAILED_APP="$app_key"
            ((fail_count++))

            if [[ "$stop_on_failure" == "true" ]]; then
                log_warn "stop_on_failure=true，停止后续部署"
                break
            fi
        fi
    done

    # ── 显示汇总 ──────────────────────────────────────────────────────────────
    echo "" | tee -a "$LOG_FILE"
    print_separator
    log_step "部署汇总"
    log_info "成功: ${success_count}  失败: ${fail_count}  共: ${total}"

    if [[ $fail_count -gt 0 ]]; then
        echo "" | tee -a "$LOG_FILE"
        log_error "部署过程中存在失败！首个失败应用: ${FAILED_APP}"

        if [[ ${#DEPLOYED_APPS[@]} -gt 0 ]]; then
            echo "" | tee -a "$LOG_FILE"
            if confirm "是否对已部署的应用执行回滚？" "n"; then
                do_rollback
            else
                log_warn "跳过回滚，请手动处理"
            fi
        fi

        echo "" | tee -a "$LOG_FILE"
        log_error "部署结束（有失败）。详细日志: $LOG_FILE"
        exit 1
    else
        echo "" | tee -a "$LOG_FILE"
        log_success "🎉 所有应用部署成功！"
        log_info "详细日志: $LOG_FILE"
        exit 0
    fi
}

main "$@"
