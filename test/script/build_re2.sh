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
# PFUZZER_LIB="${WORKSPACE_DIR}/../pfuzzer/build/libfuzzer.a"

cd re2
git checkout 972a15cedd008d846f1a39b2e88ce48d7f166cbd

FUZZ_TARGET=$(pwd)/re2/fuzzing/re2_fuzzer.cc
DEFAULT_FLAGS="-O1 -g -fno-omit-frame-pointer -fsanitize-coverage=trace-cmp"

if [ $UPDATE_PFUZZER -eq 1 ]; then
  mkdir -p build-runtime
  cd build-runtime
  export CC=clang
  export CXX=clang++
  export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

  ${CXX} $CXXFLAGS \
    -I../re2 \
    $FUZZ_TARGET \
    -o fuzzer \
    ../re2/re2.cc \
    -lpthread -lm ${PFUZZER_LIB}
  exit 0
fi

if [ ! -d abseil-cpp ]; then
  git clone --depth=1 https://github.com/abseil/abseil-cpp
fi
pushd abseil-cpp
mkdir -p build && cd build && rm -rf *
cmake -DCMAKE_POSITION_INDEPENDENT_CODE=ON -DCMAKE_INSTALL_PREFIX=$(pwd)/install ..
make -j$JOBS && make install
ABSEIL_PATH=$(pwd)/install
popd

mkdir -p build-runtime
pushd build-runtime
# rm -rf *

export CC=clang
export CXX=clang++
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

cmake -DBUILD_SHARED_LIBS=OFF -DCMAKE_PREFIX_PATH=$ABSEIL_PATH -DCMAKE_INSTALL_PREFIX=$(pwd)/install  ..
make -j$JOBS && make install

${CXX} -fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer $DEFAULT_FLAGS \
  -I$(pwd)/install/include -I$ABSEIL_PATH/include -I.. \
  $FUZZ_TARGET -o fuzzer_cov libre2.a \
  -Wl,--start-group $ABSEIL_PATH/lib/*.a -Wl,--end-group \
  -lpthread -lstdc++

${CXX} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
  -I$(pwd)/install/include -I$ABSEIL_PATH/include -I.. \
  $FUZZ_TARGET -o fuzzer libre2.a  \
  -Wl,--start-group $ABSEIL_PATH/lib/*.a -Wl,--end-group \
  -lpthread -lstdc++ ${PFUZZER_LIB}

popd


mkdir -p build__HFC_qzmp__
pushd build__HFC_qzmp__
rm -rf *

export CC=gclang
export CXX=gclang++
export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

cmake -DBUILD_SHARED_LIBS=OFF -DCMAKE_PREFIX_PATH=$ABSEIL_PATH -DCMAKE_INSTALL_PREFIX=$(pwd)/install  ..
make -j$JOBS && make install

${CXX} -c -fsanitize=fuzzer $DEFAULT_FLAGS \
  -I$(pwd)/install/include -I$ABSEIL_PATH/include -I.. \
  $FUZZ_TARGET -o fuzzer.o

${CXX} -fsanitize=fuzzer $DEFAULT_FLAGS \
  -o fuzzer fuzzer.o libre2.a \
  -Wl,--start-group $ABSEIL_PATH/lib/*.a -Wl,--end-group \
  -lpthread -lstdc++

get-bc -o fuzzer.bc fuzzer
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" fuzzer.bc

popd