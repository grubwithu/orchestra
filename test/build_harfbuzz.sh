#!/bin/bash
set -e

cd harfbuzz
git checkout a1d9bfe

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -flto -g"

mkdir -p build
pushd build
rm -rf *
export CXXFLAGS="-fsanitize=fuzzer-no-link -fuse-ld=gold $DEFAULT_FLAGS"
export CFLAGS="-fsanitize=fuzzer-no-link -fuse-ld=gold $DEFAULT_FLAGS"
cmake .. -DBUILD_SHARED_LIBS=false -DCMAKE_INSTALL_PREFIX=$(pwd)/install -DCMAKE_BUILD_TYPE=Debug
FUZZ_INTROSPECTOR=1 make -j && make install
FUZZ_INTROSPECTOR=1 ${CC} -fsanitize=fuzzer -fuse-ld=gold $DEFAULT_FLAGS \
  -lstdc++ \
  -I$(pwd)/install/include \
  -I../src \
  ../test/fuzzing/hb-shape-fuzzer.cc \
  -o hb-shape-fuzzer \
  $(pwd)/install/lib/libharfbuzz.a
popd

mkdir -p build-runtime
pushd build-runtime
rm -rf *
export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
cmake .. -DBUILD_SHARED_LIBS=false -DCMAKE_INSTALL_PREFIX=$(pwd)/install -DCMAKE_BUILD_TYPE=Debug
make -j && make install
${CC} -fsanitize=fuzzer -fprofile-instr-generate -fcoverage-mapping $DEFAULT_FLAGS \
  -lstdc++ \
  -I$(pwd)/install/include \
  -I../src \
  ../test/fuzzing/hb-shape-fuzzer.cc \
  -o hb-shape-fuzzer_cov \
  $(pwd)/install/lib/libharfbuzz.a
${CC} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
  -lstdc++ \
  -I$(pwd)/install/include \
  -I../src \
  ../test/fuzzing/hb-shape-fuzzer.cc \
  -o hb-shape-fuzzer \
  $(pwd)/install/lib/libharfbuzz.a \
  ../../../pfuzzer/build/libfuzzer.a
popd
