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
PFUZZER_LIB="${WORKSPACE_DIR}/../pfuzzer/build/libfuzzer.a"

cd openssl
git checkout openssl-3.6

DEFAULT_FLAGS="-O1 -g -fno-omit-frame-pointer -fsanitize-coverage=trace-cmp"

# 更新 PFuzzer
if [ $UPDATE_PFUZZER -eq 1 ]; then
  mkdir -p build-runtime
  cd build-runtime
  export CC=clang
  export CXX=clang++
  export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

  ${CXX} $CXXFLAGS \
    -I$(pwd)/../include \
    ../fuzz/fuzz_x509.c \
    -o fuzz_x509 \
    ../libcrypto.a \
    -lm ${PFUZZER_LIB}
  exit 0
fi

mkdir -p build-runtime
pushd build-runtime
# rm -rf *

export CC=clang
export CXX=clang++
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

../config enable-fuzz-libfuzzer \
  --with-fuzzer-lib=$PFUZZER_LIB \
  $CFLAGS -lstdc++ \
  no-shared no-tests no-ssl3 no-comp \
  --prefix=$(pwd)/install
make -j$JOBS
mv fuzz/x509 ./x509_cov

../config enable-fuzz-libfuzzer \
  --with-fuzzer-lib=$PFUZZER_LIB \
  -fsanitize=fuzzer-no-link $DEFAULT_FLAGS -lstdc++ \
  no-shared no-tests no-ssl3 no-comp \
  --prefix=$(pwd)/install
make -j$JOBS
mv fuzz/x509 ./x509

popd

mkdir -p build__HFC_qzmp__
pushd build__HFC_qzmp__
rm -rf *

export CC=gclang
export CXX=gclang++
export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

../config enable-fuzz-libfuzzer \
  --with-fuzzer-lib=$PFUZZER_LIB \
  $CFLAGS -lstdc++ \
  no-shared no-tests no-ssl3 no-comp \
  --prefix=$(pwd)/install
make -j$JOBS

get-bc -o x509.bc fuzz/x509
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" x509.bc

popd