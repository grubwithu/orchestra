#!/bin/bash
set -e

# Not running now
exit 0


UPDATE_PFUZZER=0
while [[ $# -gt 0 ]]; do
  case $1 in
    --update-pfuzzer)
      UPDATE_PFUZZER=1
      shift
      ;;
    *)
      shift
      ;;
  esac
done

JOBS=$(($(grep -c ^processor /proc/cpuinfo) < 16 ? $(grep -c ^processor /proc/cpuinfo) : 16))

WORKSPACE_DIR=$(pwd)
PFUZZER_LIB="${WORKSPACE_DIR}/../pfuzzer/build/libfuzzer.a"

# ========================================================
# 0. 准备 systemd 仓库
# ========================================================
if [ ! -d "systemd" ]; then
  echo "Cloning systemd repository..."
  git clone --depth 1 https://github.com/systemd/systemd.git
fi

cd systemd

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -g"

# 找到生成的私有共享库文件的函数（systemd 编译后会生成类似 libsystemd-shared-255.a 的文件）
find_shared_lib() {
    find "$1" -name "libsystemd-shared-*.a" | head -n 1
}

if [ $UPDATE_PFUZZER -eq 1 ]; then
  cd build-runtime
  SHARED_LIB=$(find_shared_lib ".")
  # 重新链接 fuzz-link-parser
  clang++ -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
    -I$(pwd)/../src/basic \
    -I$(pwd)/../src/shared \
    -I$(pwd)/../src/systemd \
    -I$(pwd) \
    ../src/fuzz/fuzz-link-parser.c \
    -o fuzz-link-parser \
    -Wl,--whole-archive "$SHARED_LIB" -Wl,--no-whole-archive \
    -lrt -lpthread -lm -lcap -lmount \
    ${PFUZZER_LIB}
  exit 0
fi

# ========================================================
# 1. 构建 Coverage 和 pfuzzer 目标
# ========================================================
mkdir -p build-runtime
pushd build-runtime
# rm -rf *
export CC="clang"
export CXX="clang++"
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

# Meson 配置：禁用大部分不必要的组件，开启 fuzz 支持
# b_lundef=false 是必须的，因为 fuzzing 链接时会有部分符号在引擎中
meson setup .. \
    --prefix=$(pwd)/install \
    -Dbuildtype=debugoptimized \
    -Doptimization=1 \
    -Dllvm-fuzz=true \
    -Dtests=false \
    -Dman=false \
    -Dimportd=false \
    -Db_lundef=false \
    -Dstatic-libsystemd=pic

# 编译所有目标，确保生成 libsystemd-shared-xxx.a
ninja -j$JOBS

SHARED_LIB=$(find_shared_lib ".")

# (A) 构建 fuzz-link-parser_cov
clang -fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer $DEFAULT_FLAGS \
    -I$(pwd)/../src/basic \
    -I$(pwd)/../src/shared \
    -I$(pwd)/../src/systemd \
    -I$(pwd) \
    ../src/fuzz/fuzz-link-parser.c \
    -o fuzz-link-parser_cov \
    -Wl,--whole-archive "$SHARED_LIB" -Wl,--no-whole-archive \
    -lrt -lpthread -lm -lcap -lmount

# (B) 构建 fuzz-link-parser (pfuzzer)
clang -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
    -I$(pwd)/../src/basic \
    -I$(pwd)/../src/shared \
    -I$(pwd)/../src/systemd \
    -I$(pwd) \
    ../src/fuzz/fuzz-link-parser.c \
    -o fuzz-link-parser \
    -Wl,--whole-archive "$SHARED_LIB" -Wl,--no-whole-archive \
    -lrt -lpthread -lm -lcap -lmount \
    ${PFUZZER_LIB}

popd

# ========================================================
# 2. 构建 Fuzz Introspector (gclang) 目标
# ========================================================
mkdir -p build__HFC_qzmp__
pushd build__HFC_qzmp__
rm -rf *
export CC="gclang"
export CXX="gclang++"
export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

meson setup .. \
    -Dbuildtype=debugoptimized \
    -Doptimization=1 \
    -Dllvm-fuzz=true \
    -Doss-fuzz=true \
    -Dtests=false \
    -Db_lundef=false

ninja -j$JOBS

SHARED_LIB=$(find_shared_lib ".")

gclang -fsanitize=fuzzer $DEFAULT_FLAGS \
    -I$(pwd)/../src/basic \
    -I$(pwd)/../src/shared \
    -I$(pwd)/../src/systemd \
    -I$(pwd) \
    ../src/fuzz/fuzz-link-parser.c \
    -o fuzz-link-parser \
    -Wl,--whole-archive "$SHARED_LIB" -Wl,--no-whole-archive \
    -lrt -lpthread -lm -lcap -lmount

get-bc -o fuzz-link-parser.bc fuzz-link-parser
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" fuzz-link-parser.bc

popd

# ========================================================
# 3. 清理
# ========================================================
echo "Cleaning up..."
# 删掉庞大的 ninja 编译中间件
find "${WORKSPACE_DIR}/systemd" -name "*.o" -type f ! -name "fuzz-link-parser.o" -delete
find "${WORKSPACE_DIR}/systemd" -name ".git" -type d -exec rm -rf {} +