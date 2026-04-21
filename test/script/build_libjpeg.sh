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

cd libjpeg-turbo
git checkout 174da652ef60ef7191357620eaf1d8fe776de723

wget -O ./fuzz/libjpeg_turbo_fuzzer.cc https://raw.githubusercontent.com/google/oss-fuzz/0c95cf2c940b880949f23e6e2460954fedbe5d61/projects/libjpeg-turbo/libjpeg_turbo_fuzzer.cc

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -g"

if [ $UPDATE_PFUZZER -eq 1 ]; then
  cd build-runtime
  clang++ -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
    -I$(pwd)/install/include \
    ../fuzz/libjpeg_turbo_fuzzer.cc \
    -o libjpeg_turbo_fuzzer \
    $(pwd)/install/lib/libturbojpeg.a \
    ../../../pfuzzer/build/libfuzzer.a
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
  -DENABLE_STATIC=1 \
  -DENABLE_SHARED=0 \
  -DWITH_TURBOJPEG=1 \
  -DWITH_SIMD=0

make -j$JOBS
make install

${CXX} -fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  ../fuzz/libjpeg_turbo_fuzzer.cc \
  -o libjpeg_turbo_fuzzer_cov \
  $(pwd)/install/lib/libturbojpeg.a

${CXX} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  ../fuzz/libjpeg_turbo_fuzzer.cc \
  -o libjpeg_turbo_fuzzer \
  $(pwd)/install/lib/libturbojpeg.a \
  ../../../pfuzzer/build/libfuzzer.a

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
  -DENABLE_STATIC=1 \
  -DENABLE_SHARED=0 \
  -DWITH_TURBOJPEG=1 \
  -DWITH_SIMD=0

make -j$JOBS
make install

${CXX} -c -fsanitize=fuzzer $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  ../fuzz/libjpeg_turbo_fuzzer.cc \
  -o libjpeg_turbo_fuzzer.o

${CXX} -fsanitize=fuzzer $DEFAULT_FLAGS \
  -o libjpeg_turbo_fuzzer \
  libjpeg_turbo_fuzzer.o \
  $(pwd)/install/lib/libturbojpeg.a

get-bc -o libjpeg_turbo_fuzzer.bc libjpeg_turbo_fuzzer
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" libjpeg_turbo_fuzzer.bc

popd