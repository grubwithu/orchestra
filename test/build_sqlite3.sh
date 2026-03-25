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

cd sqlite3
git checkout 4d9384cba35ce7971431da9b543e0f9d68975947

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -g"

if [ $UPDATE_PFUZZER -eq 1 ]; then
  cd build-runtime
  clang -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
    -lstdc++ \
    -I$(pwd)/install/include \
    ../test/ossfuzz.c \
    -o ossfuzz \
    $(pwd)/install/lib/libsqlite3.a \
    ../../../pfuzzer/build/libfuzzer.a
  exit 0
fi

mkdir -p build-runtime
pushd build-runtime
rm -rf *
export CC="clang"
export CXX="clang++"
export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
../configure --shared=0 --prefix=$(pwd)/install --disable-tcl && make -j$JOBS && make install
${CC} -fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer $DEFAULT_FLAGS \
  -lstdc++ \
  -I$(pwd)/install/include \
  ../test/ossfuzz.c \
  -o ossfuzz_cov \
  $(pwd)/install/lib/libsqlite3.a
${CC} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
  -lstdc++ \
  -I$(pwd)/install/include \
  ../test/ossfuzz.c \
  -o ossfuzz \
  $(pwd)/install/lib/libsqlite3.a \
  ../../../pfuzzer/build/libfuzzer.a
popd


mkdir -p build__HFC_qzmp__
pushd build__HFC_qzmp__
rm -rf *
export CC="gclang"
export CXX="gclang++"
export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
../configure --shared=0 --prefix=$(pwd)/install --disable-tcl && make -j$JOBS && make install

${CC} -c -fsanitize=fuzzer $DEFAULT_FLAGS \
  -lstdc++ \
  -I$(pwd)/install/include \
  ../test/ossfuzz.c \
  -o ossfuzz.o \
  $(pwd)/install/lib/libsqlite3.a

${CC} -fsanitize=fuzzer $DEFAULT_FLAGS \
  -o ossfuzz \
  ossfuzz.o \
  $(pwd)/install/lib/libsqlite3.a

get-bc -o ossfuzz.bc ossfuzz
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" ossfuzz.bc

popd
