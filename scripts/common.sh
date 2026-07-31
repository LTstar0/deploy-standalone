#!/usr/bin/env bash
# =============================================================================
# scripts/common.sh — 公共函数库
# 被 init.sh 及各应用脚本 source 引用
# =============================================================================

# ── 颜色 & 样式 ───────────────────────────────────────────────────────────────
RESET='\033[0m'
BOLD='\033[1m'
DIM='\033[2m'

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
WHITE='\033[0;37m'

BOLD_RED='\033[1;31m'
BOLD_GREEN='\033[1;32m'
BOLD_YELLOW='\033[1;33m'
BOLD_BLUE='\033[1;34m'
BOLD_CYAN='\033[1;36m'
BOLD_WHITE='\033[1;37m'

# ── 全局日志文件（由 init.sh 设置） ──────────────────────────────────────────
LOG_FILE="${LOG_FILE:-/tmp/deploy-$(date +%Y%m%d-%H%M%S).log}"

# ── 日志函数 ──────────────────────────────────────────────────────────────────

_ts() { date '+%Y-%m-%d %H:%M:%S'; }

log_info() {
    local msg="$*"
    echo -e "${CYAN}[INFO]${RESET}  $(_ts) — $msg" | tee -a "$LOG_FILE"
}

log_success() {
    local msg="$*"
    echo -e "${BOLD_GREEN}[OK]${RESET}    $(_ts) — $msg" | tee -a "$LOG_FILE"
}

log_warn() {
    local msg="$*"
    echo -e "${BOLD_YELLOW}[WARN]${RESET}  $(_ts) — $msg" | tee -a "$LOG_FILE"
}

log_error() {
    local msg="$*"
    echo -e "${BOLD_RED}[ERROR]${RESET} $(_ts) — $msg" | tee -a "$LOG_FILE"
}

log_step() {
    local msg="$*"
    echo -e "\n${BOLD_BLUE}▶ ${msg}${RESET}" | tee -a "$LOG_FILE"
}

log_raw() {
    echo -e "$*" | tee -a "$LOG_FILE"
}

# ── Banner ────────────────────────────────────────────────────────────────────

print_banner() {
    local project="${1:-项目发布系统}"
    local version="${2:-1.0.0}"
    local env="${3:-production}"
    echo -e "${BOLD_CYAN}" | tee -a "$LOG_FILE"
    echo '  ╔══════════════════════════════════════════════════╗' | tee -a "$LOG_FILE"
    echo "  ║        🚀  Deploy Standalone  — v${version}        ║" | tee -a "$LOG_FILE"
    echo "  ║           项目: ${project}" | tee -a "$LOG_FILE"
    echo "  ║           环境: ${env}" | tee -a "$LOG_FILE"
    echo "  ║           时间: $(date '+%Y-%m-%d %H:%M:%S')" | tee -a "$LOG_FILE"
    echo '  ╚══════════════════════════════════════════════════╝' | tee -a "$LOG_FILE"
    echo -e "${RESET}" | tee -a "$LOG_FILE"
}

# ── 分隔线 ────────────────────────────────────────────────────────────────────

print_separator() {
    echo -e "${DIM}──────────────────────────────────────────────────────${RESET}" | tee -a "$LOG_FILE"
}

# ── Spinner 动画 ──────────────────────────────────────────────────────────────

SPINNER_PID=""

start_spinner() {
    local msg="${1:-Processing...}"
    local spinner='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
    (
        i=0
        while true; do
            char="${spinner:$((i % ${#spinner})):1}"
            printf "\r  ${BOLD_CYAN}%s${RESET} %s" "$char" "$msg"
            sleep 0.1
            ((i++))
        done
    ) &
    SPINNER_PID=$!
    disown "$SPINNER_PID" 2>/dev/null || true
}

stop_spinner() {
    local status="${1:-ok}"  # ok | fail
    if [[ -n "$SPINNER_PID" ]]; then
        kill "$SPINNER_PID" 2>/dev/null || true
        wait "$SPINNER_PID" 2>/dev/null || true
        SPINNER_PID=""
    fi
    printf "\r  "  # 清除 spinner 行
    if [[ "$status" == "ok" ]]; then
        echo -e "${BOLD_GREEN}✔${RESET}  完成"
    else
        echo -e "${BOLD_RED}✖${RESET}  失败"
    fi
}

# ── 依赖检查 ──────────────────────────────────────────────────────────────────

check_dependencies() {
    local missing=()
    local deps=("jq" "curl")

    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &>/dev/null; then
            missing+=("$dep")
        fi
    done

    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "缺少依赖工具: ${missing[*]}"
        log_error "请安装后重试："
        for tool in "${missing[@]}"; do
            case "$tool" in
                jq)   echo -e "  ${YELLOW}brew install jq${RESET}  /  apt-get install -y jq" ;;
                curl) echo -e "  ${YELLOW}brew install curl${RESET}  /  apt-get install -y curl" ;;
            esac
        done
        exit 1
    fi

    log_info "依赖检查通过 (jq, curl)"
}

# ── JSON 工具函数（需要 jq） ───────────────────────────────────────────────────

# 获取所有应用 key 列表
get_all_app_keys() {
    local config="$1"
    jq -r '.apps | keys[]' "$config"
}

# 获取指定应用的字段值
get_app_field() {
    local config="$1" app_key="$2" field="$3"
    jq -r ".apps[\"${app_key}\"].${field} // empty" "$config"
}

# 获取应用是否启用
is_app_enabled() {
    local config="$1" app_key="$2"
    local val
    val=$(jq -r ".apps[\"${app_key}\"].enabled // true" "$config")
    [[ "$val" == "true" ]]
}

# 获取应用依赖列表
get_app_deps() {
    local config="$1" app_key="$2"
    jq -r ".apps[\"${app_key}\"].dependencies[]? // empty" "$config"
}

# 获取全局部署配置字段
get_deploy_config() {
    local config="$1" field="$2" default="${3:-}"
    local val
    val=$(jq -r ".deploy_config.${field} // empty" "$config")
    echo "${val:-$default}"
}

# ── 拓扑排序 ─────────────────────────────────────────────────────────────────
#
# 输入：配置文件路径、选中的 app key 数组（已展开含依赖）
# 输出：按 deploy_order 排序的 key 列表（stdout，每行一个）
#
sort_by_deploy_order() {
    local config="$1"
    shift
    local apps=("$@")

    local sorted_apps=()

    for app in "${apps[@]}"; do
        local order
        order=$(jq -r ".apps[\"${app}\"].deploy_order // 99" "$config")
        sorted_apps+=("${order}:${app}")
    done

    # 按 order 数字排序
    IFS=$'\n' sorted_apps=($(printf '%s\n' "${sorted_apps[@]}" | sort -t: -k1 -n))
    unset IFS

    for entry in "${sorted_apps[@]}"; do
        echo "${entry#*:}"
    done
}

# ── 脚本执行工具 ─────────────────────────────────────────────────────────────

# 执行一个脚本，返回退出码，并将输出追加到日志
run_script() {
    local script_path="$1"
    local label="${2:-script}"

    if [[ -z "$script_path" ]]; then
        return 0
    fi

    if [[ ! -f "$script_path" ]]; then
        log_warn "脚本不存在，跳过 [$label]: $script_path"
        return 0
    fi

    if [[ ! -x "$script_path" ]]; then
        chmod +x "$script_path"
    fi

    log_info "执行 $label: $script_path"
    if bash "$script_path" >> "$LOG_FILE" 2>&1; then
        return 0
    else
        local exit_code=$?
        log_error "$label 执行失败 (exit code: $exit_code)"
        return $exit_code
    fi
}

# ── 用户确认 ─────────────────────────────────────────────────────────────────

confirm() {
    local msg="${1:-是否继续？}"
    local default="${2:-n}"  # y 或 n
    local prompt

    if [[ "$default" == "y" ]]; then
        prompt="${msg} [Y/n]: "
    else
        prompt="${msg} [y/N]: "
    fi

    while true; do
        echo -en "${BOLD_WHITE}${prompt}${RESET}"
        read -r answer
        answer="${answer:-$default}"
        case "${answer,,}" in
            y|yes) return 0 ;;
            n|no)  return 1 ;;
            *) echo "  请输入 y 或 n" ;;
        esac
    done
}
