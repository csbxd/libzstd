#!/bin/bash
set -e  # 遇到错误立即退出

# 配置变量
ZSTD_SRC="libc/zstd"
TMP_DIR="tmp"
PATCH_DIR="libc/patches"
PATCH_FILE="fix_ccgo_compilation.patch"
COMBINE_SCRIPT="combine.py"
OUTPUT_C_FILE="zstd.c"
OUTPUT_GO_FILE="zstd_linux_amd64.go"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 检查依赖
check_dependencies() {
    local deps=("git" "python3" "ccgo" "sed")
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            error "Required dependency '$dep' not found"
            return 1
        fi
    done
    info "All dependencies checked"
}

# 清理和准备目录
prepare_directory() {
    info "Cleaning up temporary directory..."
    rm -rf "$TMP_DIR"
    mkdir -p "$TMP_DIR"

    info "Copying zstd source..."
    cp -r "$ZSTD_SRC" "$TMP_DIR/zstd"
}

# 应用补丁
apply_patch() {
    info "Applying patch..."
    cd "$TMP_DIR/zstd"

    if [[ -f "../../$PATCH_DIR/$PATCH_FILE" ]]; then
        if git apply "../../$PATCH_DIR/$PATCH_FILE"; then
            info "Patch applied successfully"
        else
            error "Failed to apply patch"
            return 1
        fi
    else
        warn "Patch file not found: $PATCH_FILE"
    fi
    cd - > /dev/null
}

# 合并源文件
combine_sources() {
    info "Combining source files..."
    cd "$TMP_DIR/zstd/build/single_file_libs"

    if [[ -f "$COMBINE_SCRIPT" ]]; then
        python3 "$COMBINE_SCRIPT" -r "../../lib" -x "legacy/zstd_legacy.h" -o "../../../$OUTPUT_C_FILE" zstd-in.c
        if [[ $? -eq 0 ]]; then
            info "Source files combined successfully: $OUTPUT_C_FILE"
        else
            error "Failed to combine source files"
            return 1
        fi
    else
        error "Combine script not found: $COMBINE_SCRIPT"
        return 1
    fi
    cd - > /dev/null
}

# 编译为Go代码
compile_to_go() {
    info "Compiling to Go code..."
    if [[ -f "tmp/$OUTPUT_C_FILE" ]]; then
        ccgo -std=c17 "tmp/$OUTPUT_C_FILE" -o "$OUTPUT_GO_FILE"
        if [[ $? -eq 0 ]]; then
            info "Go code generated successfully: $OUTPUT_GO_FILE"
        else
            error "Failed to compile to Go code"
            return 1
        fi
    else
        error "Source file not found: $OUTPUT_C_FILE"
        return 1
    fi
}

# 替换函数调用
replace_function_calls() {
    info "Replacing function calls in Go code..."

    if [[ -f "$OUTPUT_GO_FILE" ]]; then
        # 替换 iqlibc.__builtin_memmove 为 libc.Xmemmove
        sed -i 's/iqlibc\.__builtin_memmove(tls,/libc.Xmemmove(tls,/g' "$OUTPUT_GO_FILE"

        info "Function calls replaced successfully"
    else
        error "Go file not found: $OUTPUT_GO_FILE"
        return 1
    fi
}

# 清理临时文件（可选）
cleanup() {
    local keep_temp=${1:-false}
    if [[ "$keep_temp" == "false" ]]; then
        info "Cleaning up temporary files..."
        rm -rf "$TMP_DIR"
    else
        info "Temporary files kept in: $TMP_DIR"
    fi
}

# 主函数
main() {
    local keep_temp=false

    # 解析参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -k|--keep-temp)
                keep_temp=true
                shift
                ;;
            -h|--help)
                echo "Usage: $0 [options]"
                echo "Options:"
                echo "  -k, --keep-temp    Keep temporary files"
                echo "  -h, --help        Show this help message"
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    info "Starting zstd build process..."

    check_dependencies
    prepare_directory
    apply_patch
    combine_sources
    compile_to_go
    replace_function_calls  # 新增的替换步骤
    cleanup "$keep_temp"
    
    info "Build completed successfully!"
}

# 运行主函数
main "$@"