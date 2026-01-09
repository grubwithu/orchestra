#!/bin/bash
set -e

cd libpng
git checkout ba980b8 

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -flto -g"

mkdir -p build
cd build
rm -rf *

# Make sure CC and CXX is specified version
# Make sure ld, ar, ranlib is corresponding to the compiler

export CXXFLAGS="-fsanitize=fuzzer-no-link -fuse-ld=gold $DEFAULT_FLAGS"
export CFLAGS="-fsanitize=fuzzer-no-link -fuse-ld=gold $DEFAULT_FLAGS"

../configure --disable-shared --prefix=$(pwd)/install && FUZZ_INTROSPECTOR=1 make -j && make install

FUZZ_INTROSPECTOR=1 ${CC} -g -fsanitize=fuzzer -fuse-ld=gold $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  ../contrib/oss-fuzz/libpng_read_fuzzer.cc \
  .libs/libpng16.a -lz -lm -lstdc++ \
  -o libpng_read_fuzzer

export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

cd ..
mkdir -p build-runtime
cd build-runtime
rm -rf *

../configure --disable-shared --prefix=$(pwd)/install && make -j && make install

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

