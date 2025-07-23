#!/bin/bash

# 测试运行脚本
# 用于统一管理和运行项目中的所有测试

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TESTS_DIR="$PROJECT_ROOT/tests"

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 显示帮助信息
show_help() {
    echo "测试运行脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  -h, --help              显示此帮助信息"
    echo "  -a, --all               运行所有测试"
    echo "  -u, --unit              运行单元测试"
    echo "  -i, --integration       运行集成测试"
    echo "  -e, --e2e               运行端到端测试"
    echo "  -c, --coverage          生成覆盖率报告"
    echo "  -v, --verbose           详细输出"
    echo "  -p, --parallel          并行运行测试"
    echo "  -m, --module <模块>     运行特定模块的测试"
    echo ""
    echo "示例:"
    echo "  $0 -a                   运行所有测试"
    echo "  $0 -u -v                运行单元测试（详细输出）"
    echo "  $0 -m gpio              运行GPIO模块测试"
    echo "  $0 -c -u                运行单元测试并生成覆盖率报告"
}

# 检查Go环境
check_go_env() {
    if ! command -v go &> /dev/null; then
        log_error "Go未安装或不在PATH中"
        exit 1
    fi
    
    log_info "Go版本: $(go version)"
}

# 运行单元测试
run_unit_tests() {
    log_info "运行单元测试..."
    
    local args=""
    if [[ "$VERBOSE" == "true" ]]; then
        args="-v"
    fi
    
    if [[ "$PARALLEL" == "true" ]]; then
        args="$args -parallel 4"
    fi
    
    cd "$TESTS_DIR"
    
    # 运行所有单元测试
    if go test ./unit/... $args; then
        log_success "单元测试通过"
    else
        log_error "单元测试失败"
        return 1
    fi
}

# 运行集成测试
run_integration_tests() {
    log_info "运行集成测试..."
    
    local args=""
    if [[ "$VERBOSE" == "true" ]]; then
        args="-v"
    fi
    
    cd "$TESTS_DIR"
    
    # 运行所有集成测试
    if go test ./integration/... $args; then
        log_success "集成测试通过"
    else
        log_error "集成测试失败"
        return 1
    fi
}

# 运行端到端测试
run_e2e_tests() {
    log_info "运行端到端测试..."
    
    local args=""
    if [[ "$VERBOSE" == "true" ]]; then
        args="-v"
    fi
    
    cd "$TESTS_DIR"
    
    # 运行所有端到端测试
    if go test ./e2e/... $args; then
        log_success "端到端测试通过"
    else
        log_error "端到端测试失败"
        return 1
    fi
}

# 运行特定模块测试
run_module_tests() {
    if [[ -z "$MODULE" ]]; then
        log_error "未指定模块名称"
        return 1
    fi
    
    log_info "运行模块测试: $MODULE"
    
    local args=""
    if [[ "$VERBOSE" == "true" ]]; then
        args="-v"
    fi
    
    cd "$TESTS_DIR"
    
    # 查找模块测试
    local test_path=""
    case "$MODULE" in
        "gpio")
            test_path="./unit/internal/gpio/..."
            ;;
        "models")
            test_path="./unit/internal/models/..."
            ;;
        "client")
            test_path="./unit/cmd/client/..."
            ;;
        "server")
            test_path="./unit/cmd/server/..."
            ;;
        *)
            log_error "未知模块: $MODULE"
            return 1
            ;;
    esac
    
    if go test $test_path $args; then
        log_success "模块 $MODULE 测试通过"
    else
        log_error "模块 $MODULE 测试失败"
        return 1
    fi
}

# 生成覆盖率报告
generate_coverage_report() {
    log_info "生成覆盖率报告..."
    
    cd "$TESTS_DIR"
    
    # 创建覆盖率目录
    mkdir -p reports/coverage
    
    # 生成覆盖率文件
    local coverage_file="reports/coverage/coverage.out"
    local html_file="reports/coverage/coverage.html"
    
    # 运行测试并生成覆盖率
    if go test -coverprofile="$coverage_file" ./unit/...; then
        # 生成HTML报告
        go tool cover -html="$coverage_file" -o="$html_file"
        log_success "覆盖率报告已生成: $html_file"
        
        # 显示覆盖率统计
        go tool cover -func="$coverage_file" | tail -1
    else
        log_error "生成覆盖率报告失败"
        return 1
    fi
}

# 运行所有测试
run_all_tests() {
    log_info "运行所有测试..."
    
    local failed=0
    
    # 运行单元测试
    if ! run_unit_tests; then
        failed=1
    fi
    
    # 运行集成测试
    if ! run_integration_tests; then
        failed=1
    fi
    
    # 运行端到端测试
    if ! run_e2e_tests; then
        failed=1
    fi
    
    if [[ $failed -eq 0 ]]; then
        log_success "所有测试通过"
    else
        log_error "部分测试失败"
        return 1
    fi
}

# 主函数
main() {
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -a|--all)
                RUN_ALL=true
                shift
                ;;
            -u|--unit)
                RUN_UNIT=true
                shift
                ;;
            -i|--integration)
                RUN_INTEGRATION=true
                shift
                ;;
            -e|--e2e)
                RUN_E2E=true
                shift
                ;;
            -c|--coverage)
                GENERATE_COVERAGE=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -p|--parallel)
                PARALLEL=true
                shift
                ;;
            -m|--module)
                MODULE="$2"
                shift 2
                ;;
            *)
                log_error "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 检查Go环境
    check_go_env
    
    # 如果没有指定任何测试类型，默认运行单元测试
    if [[ "$RUN_ALL" != "true" && "$RUN_UNIT" != "true" && "$RUN_INTEGRATION" != "true" && "$RUN_E2E" != "true" && -z "$MODULE" ]]; then
        RUN_UNIT=true
    fi
    
    # 执行测试
    local exit_code=0
    
    if [[ "$RUN_ALL" == "true" ]]; then
        if ! run_all_tests; then
            exit_code=1
        fi
    else
        if [[ "$RUN_UNIT" == "true" ]]; then
            if ! run_unit_tests; then
                exit_code=1
            fi
        fi
        
        if [[ "$RUN_INTEGRATION" == "true" ]]; then
            if ! run_integration_tests; then
                exit_code=1
            fi
        fi
        
        if [[ "$RUN_E2E" == "true" ]]; then
            if ! run_e2e_tests; then
                exit_code=1
            fi
        fi
        
        if [[ -n "$MODULE" ]]; then
            if ! run_module_tests; then
                exit_code=1
            fi
        fi
    fi
    
    # 生成覆盖率报告
    if [[ "$GENERATE_COVERAGE" == "true" ]]; then
        if ! generate_coverage_report; then
            exit_code=1
        fi
    fi
    
    exit $exit_code
}

# 运行主函数
main "$@" 