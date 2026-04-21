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

cd libpcap
git checkout 44aa24f86dde8285f0cfa9f4624f61140951b1f6

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -g"

if [ $UPDATE_PFUZZER -eq 1 ]; then
  cd build-runtime
  export CC="clang"
  export CXX="clang++"
  export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
  export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
  
  ${CC} $CFLAGS \
    -I$(pwd)/install/include \
    -c ../testprogs/fuzz/fuzz_both.c \
    -o fuzz_both.o

  ${CXX} $CXXFLAGS \
    -o fuzz_both \
    fuzz_both.o \
    $(pwd)/install/lib/libpcap.a \
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

cmake .. \
  -DCMAKE_INSTALL_PREFIX=$(pwd)/install \
  -DCMAKE_INSTALL_LIBDIR=lib \
  -DBUILD_SHARED_LIBS=OFF \
  -DBUILD_PROGRAMS=OFF \
  -DDISABLE_BLUETOOTH=ON \
  -DDISABLE_DBUS=ON \
  -DDISABLE_RDMA=ON \
  -DDISABLE_DAG=ON \
  -DDISABLE_TC=ON \
  -DDISABLE_SNF=ON

make -j$JOBS
make install

${CC} -fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  -c ../testprogs/fuzz/fuzz_both.c \
  -o fuzz_both_cov.o

${CXX} -fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer $DEFAULT_FLAGS \
  -o fuzz_both_cov \
  fuzz_both_cov.o \
  $(pwd)/install/lib/libpcap.a

${CC} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  -c ../testprogs/fuzz/fuzz_both.c \
  -o fuzz_both.o

${CXX} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
  -o fuzz_both \
  fuzz_both.o \
  $(pwd)/install/lib/libpcap.a \
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
  -DBUILD_PROGRAMS=OFF \
  -DDISABLE_BLUETOOTH=ON \
  -DDISABLE_DBUS=ON \
  -DDISABLE_RDMA=ON \
  -DDISABLE_DAG=ON \
  -DDISABLE_TC=ON \
  -DDISABLE_SNF=ON

make -j$JOBS
make install

${CC} -c -fsanitize=fuzzer $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  ../testprogs/fuzz/fuzz_both.c \
  -o fuzz_both.o

${CXX} -fsanitize=fuzzer $DEFAULT_FLAGS \
  -o fuzz_both \
  fuzz_both.o \
  $(pwd)/install/lib/libpcap.a

get-bc -o fuzz_both.bc fuzz_both
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" fuzz_both.bc

popd
