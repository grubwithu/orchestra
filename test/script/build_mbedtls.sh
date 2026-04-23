#!/bin/bash
set -e

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
PFUZZER_LIB="${PFUZZER_LIB:-${WORKSPACE_DIR}/../pfuzzer/build/libfuzzer.a}"

cd mbedtls
git checkout 19580935569e05255709752c13ee285a9625cb01

if [ ! -f "${PFUZZER_LIB}" ]; then
  echo "PFUZZER_LIB not found: ${PFUZZER_LIB}" >&2
  exit 1
fi

rm -rf venv
python3 -m venv venv
source venv/bin/activate
pip install -r scripts/basic.requirements.txt

python3 scripts/config.py full
python3 scripts/config.py set MBEDTLS_PLATFORM_TIME_ALT
python3 scripts/config.py unset MBEDTLS_USE_PSA_CRYPTO

mkdir -p libfuzzer/lib
cp "${PFUZZER_LIB}" libfuzzer/lib/libFuzzingEngine.a

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -g"

if [ $UPDATE_PFUZZER -eq 1 ]; then
  cd build-runtime

  FUZZ_OBJ_DIR="$(pwd)/programs/fuzz/CMakeFiles/fuzz_dtlsclient.dir"
  TEST_OBJ_DIR="$(pwd)/CMakeFiles/mbedtls_test.dir"

  clang++ -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
    -o fuzz_dtlsclient \
    "${FUZZ_OBJ_DIR}/fuzz_dtlsclient.c.o" \
    "${FUZZ_OBJ_DIR}/common.c.o" \
    ${TEST_OBJ_DIR}/framework/tests/src/*.o \
    ${TEST_OBJ_DIR}/framework/tests/src/drivers/*.o \
    ${TEST_OBJ_DIR}/tests/src/*.o \
    "$(pwd)/library/libmbedtls.a" \
    "$(pwd)/library/libmbedx509.a" \
    "$(pwd)/library/libmbedcrypto.a" \
    "$(pwd)/3rdparty/everest/libeverest.a" \
    "$(pwd)/3rdparty/p256-m/libp256m.a" \
    ${PFUZZER_LIB}
  exit 0
fi

mkdir -p build-runtime
pushd build-runtime
rm -rf ./*

export CC=clang
export CXX=clang++
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export LDFLAGS="-L$(pwd)/../libfuzzer/lib"

cmake \
  -DCMAKE_BUILD_TYPE=Debug \
  -DCMAKE_C_FLAGS="${CFLAGS}" \
  -DCMAKE_CXX_FLAGS="${CXXFLAGS}" \
  -DCMAKE_EXE_LINKER_FLAGS="${LDFLAGS}" \
  -DCMAKE_PREFIX_PATH="$(pwd)/../libfuzzer" \
  ..
make -j$JOBS fuzz_dtlsclient

FUZZ_OBJ_DIR="$(pwd)/programs/fuzz/CMakeFiles/fuzz_dtlsclient.dir"
TEST_OBJ_DIR="$(pwd)/CMakeFiles/mbedtls_test.dir"

${CXX} -fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer $DEFAULT_FLAGS \
  -o fuzz_dtlsclient_cov \
  "${FUZZ_OBJ_DIR}/fuzz_dtlsclient.c.o" \
  "${FUZZ_OBJ_DIR}/common.c.o" \
  ${TEST_OBJ_DIR}/framework/tests/src/*.o \
  ${TEST_OBJ_DIR}/framework/tests/src/drivers/*.o \
  ${TEST_OBJ_DIR}/tests/src/*.o \
  "$(pwd)/library/libmbedtls.a" \
  "$(pwd)/library/libmbedx509.a" \
  "$(pwd)/library/libmbedcrypto.a" \
  "$(pwd)/3rdparty/everest/libeverest.a" \
  "$(pwd)/3rdparty/p256-m/libp256m.a"

cp programs/fuzz/fuzz_dtlsclient ./fuzz_dtlsclient

popd

mkdir -p build__HFC_qzmp__
pushd build__HFC_qzmp__
rm -rf ./*

export CC=gclang
export CXX=gclang++
export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export LDFLAGS="-L$(pwd)/../libfuzzer/lib"

cmake \
  -DCMAKE_BUILD_TYPE=Debug \
  -DCMAKE_C_FLAGS="${CFLAGS}" \
  -DCMAKE_CXX_FLAGS="${CXXFLAGS}" \
  -DCMAKE_EXE_LINKER_FLAGS="${LDFLAGS}" \
  -DCMAKE_PREFIX_PATH="$(pwd)/../libfuzzer" \
  ..
make -j$JOBS fuzz_dtlsclient

FUZZ_OBJ_DIR="$(pwd)/programs/fuzz/CMakeFiles/fuzz_dtlsclient.dir"
TEST_OBJ_DIR="$(pwd)/CMakeFiles/mbedtls_test.dir"

${CXX} -fsanitize=fuzzer $DEFAULT_FLAGS \
  -o fuzz_dtlsclient \
  "${FUZZ_OBJ_DIR}/fuzz_dtlsclient.c.o" \
  "${FUZZ_OBJ_DIR}/common.c.o" \
  ${TEST_OBJ_DIR}/framework/tests/src/*.o \
  ${TEST_OBJ_DIR}/framework/tests/src/drivers/*.o \
  ${TEST_OBJ_DIR}/tests/src/*.o \
  "$(pwd)/library/libmbedtls.a" \
  "$(pwd)/library/libmbedx509.a" \
  "$(pwd)/library/libmbedcrypto.a" \
  "$(pwd)/3rdparty/everest/libeverest.a" \
  "$(pwd)/3rdparty/p256-m/libp256m.a"

get-bc -o fuzz_dtlsclient.bc fuzz_dtlsclient
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" fuzz_dtlsclient.bc

popd
