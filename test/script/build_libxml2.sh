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

cd libxml2
git checkout ca6a8cf94672e6f3c48c08d6af2201599788fc17

if [ ! -f "configure" ]; then
  NOCONFIGURE=1 ./autogen.sh
fi

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -g"

if [ $UPDATE_PFUZZER -eq 1 ]; then
  cd build-runtime
  export CC="clang"
  export CXX="clang++"
  export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
  export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
  
  ${CXX} $CXXFLAGS \
    fuzz/xml.o fuzz/fuzz.o \
    -o xml \
    .libs/libxml2.a \
    ${PFUZZER_LIB}
  exit 0
fi

mkdir -p build-runtime
pushd build-runtime
# rm -rf *

export CC="clang"
export CXX="clang++"
export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

../configure \
  --disable-shared \
  --without-debug \
  --without-http \
  --without-python \
  --without-zlib \
  --without-lzma

make -j$JOBS

pushd fuzz
make fuzz.o xml.o
popd

${CXX} -fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer $DEFAULT_FLAGS \
  fuzz/xml.o fuzz/fuzz.o \
  -o xml_cov \
  .libs/libxml2.a

${CXX} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
  fuzz/xml.o fuzz/fuzz.o \
  -o xml \
  .libs/libxml2.a \
  ${PFUZZER_LIB}

popd


mkdir -p build__HFC_qzmp__
pushd build__HFC_qzmp__
rm -rf *

export CC="gclang"
export CXX="gclang++"
export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

../configure \
  --disable-shared \
  --without-debug \
  --without-http \
  --without-python \
  --without-zlib \
  --without-lzma

make -j$JOBS

pushd fuzz
make fuzz.o xml.o
popd

${CXX} -fsanitize=fuzzer $DEFAULT_FLAGS \
  -o xml \
  fuzz/xml.o fuzz/fuzz.o \
  .libs/libxml2.a

get-bc -o xml.bc xml
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" xml.bc

popd