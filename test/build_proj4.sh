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

cd proj4
git checkout 5c64452ecd74a99e05556d1f4e9c1d0af1ed06b7

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -g"

if [ $UPDATE_PFUZZER -eq 1 ]; then
  cd build-runtime
  export CC="clang"
  export CXX="clang++"
  export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
  
  ${CXX} $CXXFLAGS \
    -I$(pwd)/install/include \
    ../test/fuzzers/proj_crs_to_crs_fuzzer.cpp \
    -o proj_crs_to_crs_fuzzer \
    $(pwd)/install/lib/libproj.a \
    -lsqlite3 -lpthread \
    ${PFUZZER_LIB}
  exit 0
fi

mkdir -p build-runtime
pushd build-runtime
rm -rf *
export CC="clang"
export CXX="clang++"
export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

cmake .. \
  -DCMAKE_INSTALL_PREFIX=$(pwd)/install \
  -DCMAKE_INSTALL_LIBDIR=lib \
  -DBUILD_SHARED_LIBS=OFF \
  -DENABLE_TIFF=OFF \
  -DENABLE_CURL=OFF \
  -DBUILD_APPS=OFF \
  -DBUILD_TESTING=OFF

make -j$JOBS
make install

${CXX} -fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  ../test/fuzzers/proj_crs_to_crs_fuzzer.cpp \
  -o proj_crs_to_crs_fuzzer_cov \
  $(pwd)/install/lib/libproj.a \
  -lsqlite3 -lpthread

${CXX} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  ../test/fuzzers/proj_crs_to_crs_fuzzer.cpp \
  -o proj_crs_to_crs_fuzzer \
  $(pwd)/install/lib/libproj.a \
  -lsqlite3 -lpthread \
  ${PFUZZER_LIB}

popd


mkdir -p build__HFC_qzmp__
pushd build__HFC_qzmp__
rm -rf *
export CC="gclang"
export CXX="gclang++"
export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

cmake .. \
  -DCMAKE_INSTALL_PREFIX=$(pwd)/install \
  -DCMAKE_INSTALL_LIBDIR=lib \
  -DBUILD_SHARED_LIBS=OFF \
  -DENABLE_TIFF=OFF \
  -DENABLE_CURL=OFF \
  -DBUILD_APPS=OFF \
  -DBUILD_TESTING=OFF

make -j$JOBS
make install

${CXX} -c -fsanitize=fuzzer $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  ../test/fuzzers/proj_crs_to_crs_fuzzer.cpp \
  -o proj_crs_to_crs_fuzzer.o

${CXX} -fsanitize=fuzzer $DEFAULT_FLAGS \
  -o proj_crs_to_crs_fuzzer \
  proj_crs_to_crs_fuzzer.o \
  $(pwd)/install/lib/libproj.a \
  -lsqlite3 -lpthread

get-bc -o proj_crs_to_crs_fuzzer.bc proj_crs_to_crs_fuzzer
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" proj_crs_to_crs_fuzzer.bc

popd