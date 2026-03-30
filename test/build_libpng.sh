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

cd libpng
git checkout ba980b8 

export CC="gclang"
export CXX="gclang++"

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -g"

if [ $UPDATE_PFUZZER -eq 1 ]; then
  cd build-runtime
  clang++ -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
    -I$(pwd)/install/include \
    ../contrib/oss-fuzz/libpng_read_fuzzer.cc \
    ../../../pfuzzer/build/libfuzzer.a \
    .libs/libpng16.a -lz -lm -lstdc++ \
    -o libpng_read_fuzzer
  exit 0
fi

mkdir -p build-runtime
pushd build-runtime
rm -rf *
export CC="clang"
export CXX="clang++"
export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
../configure --disable-shared --prefix=$(pwd)/install && make -j$JOBS && make install

${CC} -fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  ../contrib/oss-fuzz/libpng_read_fuzzer.cc \
  .libs/libpng16.a -lz -lm -lstdc++ \
  -o libpng_read_fuzzer_cov

${CC} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  ../contrib/oss-fuzz/libpng_read_fuzzer.cc \
  ../../../pfuzzer/build/libfuzzer.a \
  .libs/libpng16.a -lz -lm -lstdc++ \
  -o libpng_read_fuzzer
popd

mkdir -p build__HFC_qzmp__
pushd build__HFC_qzmp__
rm -rf *
export CC="gclang"
export CXX="gclang++"
export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
../configure --disable-shared --prefix=$(pwd)/install && make -j$JOBS && make install

${CXX} -c -fsanitize=fuzzer $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  ../contrib/oss-fuzz/libpng_read_fuzzer.cc \
  .libs/libpng16.a -lz -lm -lstdc++ \
  -o libpng_read_fuzzer.o

${CXX} -fsanitize=fuzzer $DEFAULT_FLAGS \
  -o libpng_read_fuzzer \
  libpng_read_fuzzer.o \
  .libs/libpng16.a -lz -lm -lstdc++

get-bc -o libpng_read_fuzzer.bc libpng_read_fuzzer
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" libpng_read_fuzzer.bc

popd
