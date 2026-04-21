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

cd mbedtls
git checkout 19580935569e05255709752c13ee285a9625cb01

mkdir -p libfuzzer/lib
cp ${PFUZZER_LIB} libfuzzer/lib/libFuzzingEngine.a

rm -rf venv
python3 -m venv venv
source venv/bin/activate
pip install -r scripts/basic.requirements.txt

scripts/config.py full
scripts/config.py set MBEDTLS_PLATFORM_TIME_ALT
scripts/config.py unset MBEDTLS_USE_PSA_CRYPTO

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -g"

if [ $UPDATE_PFUZZER -eq 1 ]; then
  cd build-runtime
  export CC="clang"
  export CXX="clang++"
  export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

  ${CXX} $CXXFLAGS \
    -I$(pwd)/install/include \
    ../fuzz_dtlsclient.c \
    -o fuzz_dtlsclient \
    $(pwd)/install/lib/libmbedtls.a \
    $(pwd)/install/lib/libmbedx509.a \
    $(pwd)/install/lib/libmbedcrypto.a \
    -lm ${PFUZZER_LIB}
  exit 0
fi

mkdir -p build-runtime
pushd build-runtime
# rm -rf *

export CC="clang"
export CXX="clang++"
export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export LIB_FUZZING_ENGINE=${PFUZZER_LIB}

cmake -DCMAKE_BUILD_TYPE=Debug \
      -DCMAKE_INSTALL_PREFIX=$(pwd)/install \
      -DCMAKE_PREFIX_PATH=$(pwd)/../libfuzzer \
      ../
make -j$JOBS
make install

${CC} -fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  -I../tests/include \
  ../programs/fuzz/fuzz_dtlsclient.c \
  -o fuzz_dtlsclient_cov \
  $(pwd)/install/lib/libmbedtls.a \
  $(pwd)/install/lib/libmbedx509.a \
  $(pwd)/install/lib/libmbedcrypto.a

${CC} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  -I../tests/include \
  ../programs/fuzz/fuzz_dtlsclient.c \
  -o fuzz_dtlsclient \
  $(pwd)/install/lib/libmbedtls.a \
  $(pwd)/install/lib/libmbedx509.a \
  $(pwd)/install/lib/libmbedcrypto.a
  ${PFUZZER_LIB}

popd

# 第二次构建：干净构建用于静态分析
mkdir -p build__HFC_qzmp__
pushd build__HFC_qzmp__
rm -rf *

export CC="gclang"
export CXX="gclang++"
export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

cmake -DCMAKE_BUILD_TYPE=Release \
      -DCMAKE_INSTALL_PREFIX=$(pwd)/install \
      -DENABLE_TESTING=OFF \
      -DENABLE_PROGRAMS=OFF \
      -DENABLE_DEBUG=OFF \
      ../

make -j$JOBS
make install

# 构建 object 文件
${CC} -c -fsanitize=fuzzer $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  ../programs/fuzz/fuzz_dtlsclient.c \
  -o fuzz_dtlsclient.o

# 链接生成 fuzz target
${CXX} -fsanitize=fuzzer $DEFAULT_FLAGS \
  -o fuzz_dtlsclient \
  fuzz_dtlsclient.o \
  $(pwd)/install/lib/libmbedtls.a \
  $(pwd)/install/lib/libmbedx509.a \
  $(pwd)/install/lib/libmbedcrypto.a \
  -lm -lstdc++

# 生成 bitcode 并做静态分析
get-bc -o fuzz_dtlsclient.bc fuzz_dtlsclient
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" fuzz_dtlsclient.bc

popd