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

cd zlib
git checkout f9dd6009be3ed32415edf1e89d1bc38380ecb95d

if [ ! -f "${PFUZZER_LIB}" ]; then
  echo "PFUZZER_LIB not found: ${PFUZZER_LIB}" >&2
  exit 1
fi

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -g"

if [ $UPDATE_PFUZZER -eq 1 ]; then
  cd build-runtime

  export CC=clang
  export CXX=clang++
  export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
  export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

  ${CXX} $CXXFLAGS \
    ../zlib_uncompress_fuzzer.cc \
    -o zlib_uncompress_fuzzer \
    -I"$(pwd)" \
    -I"$(pwd)/.." \
    "$(pwd)/libz.a" \
    "${PFUZZER_LIB}"
  exit 0
fi

mkdir -p build-runtime
pushd build-runtime
rm -rf ./*

export CC=clang
export CXX=clang++
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

cmake .. \
  -DCMAKE_BUILD_TYPE=Debug \
  -DCMAKE_C_FLAGS="${CFLAGS}" \
  -DCMAKE_CXX_FLAGS="${CXXFLAGS}" \
  -DZLIB_BUILD_SHARED=OFF \
  -DZLIB_BUILD_STATIC=ON \
  -DZLIB_BUILD_TESTING=OFF \
  -DZLIB_INSTALL=OFF
make -j$JOBS zlibstatic

${CXX} -fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer $DEFAULT_FLAGS \
  ../zlib_uncompress_fuzzer.cc \
  -o zlib_uncompress_fuzzer_cov \
  -I"$(pwd)" \
  -I"$(pwd)/.." \
  "$(pwd)/libz.a"

${CXX} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
  ../zlib_uncompress_fuzzer.cc \
  -o zlib_uncompress_fuzzer \
  -I"$(pwd)" \
  -I"$(pwd)/.." \
  "$(pwd)/libz.a" \
  "${PFUZZER_LIB}"

popd

mkdir -p build__HFC_qzmp__
pushd build__HFC_qzmp__
rm -rf ./*

export CC=gclang
export CXX=gclang++
export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

cmake .. \
  -DCMAKE_BUILD_TYPE=Debug \
  -DCMAKE_C_FLAGS="${CFLAGS}" \
  -DCMAKE_CXX_FLAGS="${CXXFLAGS}" \
  -DZLIB_BUILD_SHARED=OFF \
  -DZLIB_BUILD_STATIC=ON \
  -DZLIB_BUILD_TESTING=OFF \
  -DZLIB_INSTALL=OFF
make -j$JOBS zlibstatic

${CXX} -fsanitize=fuzzer $DEFAULT_FLAGS \
  ../zlib_uncompress_fuzzer.cc \
  -o zlib_uncompress_fuzzer \
  -I"$(pwd)" \
  -I"$(pwd)/.." \
  "$(pwd)/libz.a"

get-bc -o zlib_uncompress_fuzzer.bc zlib_uncompress_fuzzer
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" zlib_uncompress_fuzzer.bc

popd
