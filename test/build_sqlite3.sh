#!/bin/bash
set -e

cd sqlite3
git checkout 4d9384cba35ce7971431da9b543e0f9d68975947

mkdir -p build
pushd build
rm -rf *

# Make sure CC and CXX is specified version
# Make sure ld, ar, ranlib is corresponding to the compiler

export CXXFLAGS="-fsanitize=fuzzer-no-link -fuse-ld=gold -flto -g"
export CFLAGS="-fsanitize=fuzzer-no-link -fuse-ld=gold -flto -g"

../configure --shared=0 --prefix=$(pwd)/install --disable-tcl && FUZZ_INTROSPECTOR=1 make -j && make install
FUZZ_INTROSPECTOR=1 ${CC} -g -fsanitize=fuzzer -fuse-ld=gold -flto -lstdc++ \
  -I$(pwd)/install/include \
  ../test/ossfuzz.c \
  -o ossfuzz \
  $(pwd)/install/lib/libsqlite3.a
popd

mkdir -p build-runtime
pushd build-runtime
rm -rf *
export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link -g"
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link -g"
../configure --shared=0 --prefix=$(pwd)/install --disable-tcl && make -j && make install
${CC} -fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer -g -lstdc++ \
  -I$(pwd)/install/include \
  ../test/ossfuzz.c \
  -o ossfuzz_cov \
  $(pwd)/install/lib/libsqlite3.a
${CC} -fsanitize=fuzzer-no-link -g -lstdc++ \
  -I$(pwd)/install/include \
  ../test/ossfuzz.c \
  -o ossfuzz \
  $(pwd)/install/lib/libsqlite3.a \
  ../../../pfuzzer/build/libfuzzer.a
popd
