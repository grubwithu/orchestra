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

cd vorbis
git checkout 68a7fc2246e734f1a311b06083fc1249551c4412

if [ ! -d "ogg" ]; then
  echo "Cloning ogg repository into vorbis/ogg..."
  git clone --depth 1 https://github.com/xiph/ogg.git ogg
fi

if [ ! -f "${PFUZZER_LIB}" ]; then
  echo "PFUZZER_LIB not found: ${PFUZZER_LIB}" >&2
  exit 1
fi

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -g"

if [ ! -f "ogg/configure" ]; then
  pushd ogg
  ./autogen.sh
  popd
fi

if [ ! -f "configure" ]; then
  ./autogen.sh
fi

if [ $UPDATE_PFUZZER -eq 1 ]; then
  cd build-runtime

  export CC=clang
  export CXX=clang++
  export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
  export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

  ${CXX} $CXXFLAGS \
    ../contrib/oss-fuzz/decode_fuzzer.cc \
    -o decode_fuzzer \
    -I"$(pwd)/ogg-build/install/include" \
    -I"$(pwd)/install/include" \
    "${PFUZZER_LIB}" \
    "$(pwd)/install/lib/libvorbisfile.a" \
    "$(pwd)/install/lib/libvorbis.a" \
    "$(pwd)/ogg-build/install/lib/libogg.a" \
    -lm
  exit 0
fi

mkdir -p build-runtime
pushd build-runtime
rm -rf ./*

export CC=clang
export CXX=clang++
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

mkdir -p ogg-build
pushd ogg-build
../../ogg/configure \
  --prefix=$(pwd)/install \
  --enable-static \
  --disable-shared \
  --disable-crc

make clean
make -j$JOBS
make install
popd

export PKG_CONFIG_PATH="$(pwd)/ogg-build/install/lib/pkgconfig${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"

../configure \
  --prefix=$(pwd)/install \
  --enable-static \
  --disable-shared

make clean
make -j$JOBS
make install

${CXX} -fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer $DEFAULT_FLAGS \
  ../contrib/oss-fuzz/decode_fuzzer.cc \
  -o decode_fuzzer_cov \
  -I"$(pwd)/ogg-build/install/include" \
  -I"$(pwd)/install/include" \
  "$(pwd)/install/lib/libvorbisfile.a" \
  "$(pwd)/install/lib/libvorbis.a" \
  "$(pwd)/ogg-build/install/lib/libogg.a" \
  -lm

${CXX} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
  ../contrib/oss-fuzz/decode_fuzzer.cc \
  -o decode_fuzzer \
  -I"$(pwd)/ogg-build/install/include" \
  -I"$(pwd)/install/include" \
  "${PFUZZER_LIB}" \
  "$(pwd)/install/lib/libvorbisfile.a" \
  "$(pwd)/install/lib/libvorbis.a" \
  "$(pwd)/ogg-build/install/lib/libogg.a" \
  -lm

popd

mkdir -p build__HFC_qzmp__
pushd build__HFC_qzmp__
rm -rf ./*

export CC=gclang
export CXX=gclang++
export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

mkdir -p ogg-build
pushd ogg-build
../../ogg/configure \
  --prefix=$(pwd)/install \
  --enable-static \
  --disable-shared \
  --disable-crc

make clean
make -j$JOBS
make install
popd

export PKG_CONFIG_PATH="$(pwd)/ogg-build/install/lib/pkgconfig${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"

../configure \
  --prefix=$(pwd)/install \
  --enable-static \
  --disable-shared

make clean
make -j$JOBS
make install

${CXX} -fsanitize=fuzzer $DEFAULT_FLAGS \
  ../contrib/oss-fuzz/decode_fuzzer.cc \
  -o decode_fuzzer \
  -I"$(pwd)/ogg-build/install/include" \
  -I"$(pwd)/install/include" \
  "$(pwd)/install/lib/libvorbisfile.a" \
  "$(pwd)/install/lib/libvorbis.a" \
  "$(pwd)/ogg-build/install/lib/libogg.a" \
  -lm

get-bc -o decode_fuzzer.bc decode_fuzzer
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" decode_fuzzer.bc

popd
