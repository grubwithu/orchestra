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

cd libxslt
git checkout 35323d6a15f6e63c9919ddbc0abe64c90a0dd88a

if [ ! -d "libxml2" ]; then
  echo "Cloning libxml2 repository into libxslt/libxml2..."
  git clone --depth 1 https://gitlab.gnome.org/GNOME/libxml2.git
fi

if [ ! -f "libxml2/configure" ]; then
  pushd libxml2
  NOCONFIGURE=1 ./autogen.sh
  popd
fi

if [ ! -f "configure" ]; then
  NOCONFIGURE=1 ./autogen.sh
fi

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -g"

if [ $UPDATE_PFUZZER -eq 1 ]; then
  cd build-runtime/libxslt-build
  export CC="clang"
  export CXX="clang++"
  export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
  
  ${CXX} $CXXFLAGS \
    tests/fuzz/xpath.o tests/fuzz/fuzz.o \
    -o xpath \
    $(pwd)/install/lib/libexslt.a \
    $(pwd)/install/lib/libxslt.a \
    $(pwd)/../libxml2-build/install/lib/libxml2.a \
    -lm -lpthread ${PFUZZER_LIB}
  exit 0
fi

mkdir -p build-runtime
pushd build-runtime
# rm -rf *

export CC="clang"
export CXX="clang++"
export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

mkdir -p libxml2-build
pushd libxml2-build
../../libxml2/configure \
  --prefix=$(pwd)/install \
  --disable-shared \
  --without-debug \
  --without-http \
  --without-python \
  --without-zlib \
  --without-lzma

make -j$JOBS
make install
popd

mkdir -p libxslt-build
pushd libxslt-build
export PATH="$(pwd)/../libxml2-build/install/bin:$PATH"
export PKG_CONFIG_PATH="$(pwd)/../libxml2-build/install/lib/pkgconfig:$PKG_CONFIG_PATH"

../../configure \
  --prefix=$(pwd)/install \
  --with-libxml-prefix=$(pwd)/../libxml2-build/install \
  --disable-shared \
  --without-python \
  --without-crypto

make -j$JOBS
make install

pushd tests/fuzz
make xpath.o fuzz.o
popd

${CXX} -fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer $DEFAULT_FLAGS \
  tests/fuzz/xpath.o tests/fuzz/fuzz.o \
  -o xpath_cov \
  install/lib/libexslt.a \
  install/lib/libxslt.a \
  ../libxml2-build/install/lib/libxml2.a \
  -lm -lpthread

${CXX} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
  tests/fuzz/xpath.o tests/fuzz/fuzz.o \
  -o xpath \
  install/lib/libexslt.a \
  install/lib/libxslt.a \
  ../libxml2-build/install/lib/libxml2.a \
  -lm -lpthread ${PFUZZER_LIB}

popd
popd


mkdir -p build__HFC_qzmp__
pushd build__HFC_qzmp__
rm -rf *
export CC="gclang"
export CXX="gclang++"
export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

mkdir -p libxml2-build
pushd libxml2-build
../../libxml2/configure \
  --prefix=$(pwd)/install \
  --disable-shared \
  --without-debug \
  --without-http \
  --without-python \
  --without-zlib \
  --without-lzma

make -j$JOBS
make install
popd

mkdir -p libxslt-build
pushd libxslt-build
export PATH="$(pwd)/../libxml2-build/install/bin:$PATH"
export PKG_CONFIG_PATH="$(pwd)/../libxml2-build/install/lib/pkgconfig:$PKG_CONFIG_PATH"

../../configure \
  --prefix=$(pwd)/install \
  --with-libxml-prefix=$(pwd)/../libxml2-build/install \
  --disable-shared \
  --without-python \
  --without-crypto

make -j$JOBS
make install

pushd tests/fuzz
make xpath.o fuzz.o
popd

${CXX} -fsanitize=fuzzer $DEFAULT_FLAGS \
  -o xpath \
  tests/fuzz/xpath.o tests/fuzz/fuzz.o \
  install/lib/libexslt.a \
  install/lib/libxslt.a \
  ../libxml2-build/install/lib/libxml2.a \
  -lm -lpthread

get-bc -o xpath.bc xpath
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" xpath.bc

popd
popd

