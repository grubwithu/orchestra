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

cd woff2
git checkout 1c69169e9e1811dccd6c54c532fedda300233968
git submodule update --init --recursive .

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
    -o convert_woff2ttf_fuzzer \
    -Wl,--whole-archive ./convert_woff2ttf_fuzzer.a -Wl,--no-whole-archive \
    "${PFUZZER_LIB}"
  exit 0
fi

mkdir -p build-runtime
rm -rf build-runtime/*

export CC=clang
export CXX=clang++
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

make clean
rm -f src/convert_woff2ttf_fuzzer.o src/convert_woff2ttf_fuzzer_new_entry.o
rm -f src/convert_woff2ttf_fuzzer.a src/convert_woff2ttf_fuzzer_new_entry.a
make -j$JOBS convert_woff2ttf_fuzzer

cp src/convert_woff2ttf_fuzzer.a build-runtime/convert_woff2ttf_fuzzer.a

pushd build-runtime
${CXX} -fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer $DEFAULT_FLAGS \
  -o convert_woff2ttf_fuzzer_cov \
  -Wl,--whole-archive ./convert_woff2ttf_fuzzer.a -Wl,--no-whole-archive

${CXX} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
  -o convert_woff2ttf_fuzzer \
  -Wl,--whole-archive ./convert_woff2ttf_fuzzer.a -Wl,--no-whole-archive \
  "${PFUZZER_LIB}"
popd

mkdir -p build__HFC_qzmp__
rm -rf build__HFC_qzmp__/*

export CC=gclang
export CXX=gclang++
export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

make clean
rm -f src/convert_woff2ttf_fuzzer.o src/convert_woff2ttf_fuzzer_new_entry.o
rm -f src/convert_woff2ttf_fuzzer.a src/convert_woff2ttf_fuzzer_new_entry.a
make -j$JOBS convert_woff2ttf_fuzzer

cp src/convert_woff2ttf_fuzzer.a build__HFC_qzmp__/convert_woff2ttf_fuzzer.a

pushd build__HFC_qzmp__
${CXX} -fsanitize=fuzzer $DEFAULT_FLAGS \
  -o convert_woff2ttf_fuzzer \
  -Wl,--whole-archive ./convert_woff2ttf_fuzzer.a -Wl,--no-whole-archive

get-bc -o convert_woff2ttf_fuzzer.bc convert_woff2ttf_fuzzer
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" convert_woff2ttf_fuzzer.bc
popd
