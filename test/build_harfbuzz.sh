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


cd harfbuzz
git checkout a1d9bfe

export CC="gclang"
export CXX="gclang++"

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -g"

if [ $UPDATE_PFUZZER -eq 1 ]; then
  cd build-runtime
  ${CC} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
    -lstdc++ \
    -I$(pwd)/install/include \
    -I../src \
    ../test/fuzzing/hb-shape-fuzzer.cc \
    -o hb-shape-fuzzer \
    $(pwd)/install/lib/libharfbuzz.a \
    ../../../pfuzzer/build/libfuzzer.a
  exit 0
fi

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


mkdir -p build__HFC_qzmp__
pushd build__HFC_qzmp__
rm -rf *
export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
cmake .. -DBUILD_SHARED_LIBS=false -DCMAKE_INSTALL_PREFIX=$(pwd)/install -DCMAKE_BUILD_TYPE=Debug
make -j && make install
${CXX} -c -fsanitize=fuzzer $DEFAULT_FLAGS \
  -lstdc++ \
  -I$(pwd)/install/include \
  -I../src \
  ../test/fuzzing/hb-shape-fuzzer.cc \
  -o hb-shape-fuzzer.o \
  $(pwd)/install/lib/libharfbuzz.a

${CXX} -c -fsanitize=fuzzer $DEFAULT_FLAGS -lstdc++ \
  hb-shape-fuzzer.o -o hb-shape-fuzzer \
  $(pwd)/install/lib/libharfbuzz.a

get-bc -o hb-shape-fuzzer.bc hb-shape-fuzzer
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" hb-shape-fuzzer.bc

popd
