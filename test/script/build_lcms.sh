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

cd lcms



wget -O cms_transform_fuzzer.c https://raw.githubusercontent.com/google/oss-fuzz/master/projects/lcms/cms_transform_fuzzer.c


if [ ! -f "configure" ]; then
  ./autogen.sh
fi

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -g"

if [ $UPDATE_PFUZZER -eq 1 ]; then
  cd build-runtime
  export CC="clang"
  export CXX="clang++"
  export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
  
  ${CXX} $CXXFLAGS \
    -I$(pwd)/install/include \
    ../cms_transform_fuzzer.c \
    -o cms_transform_fuzzer \
    $(pwd)/install/lib/liblcms2.a \
    -lm ${PFUZZER_LIB}
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
  --prefix=$(pwd)/install \
  --disable-shared \
  --with-jpeg=no \
  --with-tiff=no \
  --with-zlib=no

make -j$JOBS
make install

${CC} -fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  ../cms_transform_fuzzer.c \
  -o cms_transform_fuzzer_cov \
  $(pwd)/install/lib/liblcms2.a \
  -lm -lstdc++

${CC} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  ../cms_transform_fuzzer.c \
  -o cms_transform_fuzzer \
  $(pwd)/install/lib/liblcms2.a \
  -lm -lstdc++ ${PFUZZER_LIB}

popd


mkdir -p build__HFC_qzmp__
pushd build__HFC_qzmp__
rm -rf *
export CC="gclang"
export CXX="gclang++"
export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

../configure \
  --prefix=$(pwd)/install \
  --disable-shared \
  --with-jpeg=no \
  --with-tiff=no \
  --with-zlib=no

make -j$JOBS
make install

${CC} -c -fsanitize=fuzzer $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  ../cms_transform_fuzzer.c \
  -o cms_transform_fuzzer.o

${CXX} -fsanitize=fuzzer $DEFAULT_FLAGS \
  -o cms_transform_fuzzer \
  cms_transform_fuzzer.o \
  $(pwd)/install/lib/liblcms2.a \
  -lm -lstdc++

get-bc -o cms_transform_fuzzer.bc cms_transform_fuzzer
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" cms_transform_fuzzer.bc

popd
